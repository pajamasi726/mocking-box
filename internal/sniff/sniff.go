// Package sniff turns raw packets into corpus/golden entries: TCP reassembly
// -> HTTP/1.x parsing -> request/response pairing. It powers three passive
// capture inputs that never sit in the request path:
//
//   - live NIC sniffing (tcpdump-style, CAP_NET_RAW)
//   - offline .pcap file conversion (zero footprint: capture with tcpdump)
//   - AWS VPC Traffic Mirroring receiver (VXLAN/UDP 4789, zero-touch hosts)
package sniff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/tcpassembly"
	"github.com/gopacket/gopacket/tcpassembly/tcpreader"

	"github.com/pajamasi726/mocking-box/internal/golden"
)

const maxBodyBytes = 1 << 20

var skipHeaders = map[string]bool{
	"host": true, "content-length": true, "connection": true,
	"accept-encoding": true, "cookie": true,
}

// Exchange is one reconstructed request/response pair.
type Exchange struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    any
	Status  int
	RespBody string
}

// Sink receives completed exchanges in completion order.
type Sink func(Exchange)

// Pipeline consumes packets and emits exchanges.
type Pipeline struct {
	port      int // service port: packets TO this port are requests
	assembler *tcpassembly.Assembler
	pairs     *pairer
	wg        *sync.WaitGroup
}

func NewPipeline(port int, sink Sink) *Pipeline {
	wg := &sync.WaitGroup{}
	pairs := &pairer{sink: sink, conns: map[string]*conn{}}
	factory := &streamFactory{port: port, pairs: pairs, wg: wg}
	pool := tcpassembly.NewStreamPool(factory)
	return &Pipeline{
		port:      port,
		assembler: tcpassembly.NewAssembler(pool),
		pairs:     pairs,
		wg:        wg,
	}
}

// HandlePacket feeds one decoded packet into the pipeline.
func (p *Pipeline) HandlePacket(pkt gopacket.Packet) {
	tcpLayer := pkt.Layer(layers.LayerTypeTCP)
	netLayer := pkt.NetworkLayer()
	if tcpLayer == nil || netLayer == nil {
		return
	}
	tcp := tcpLayer.(*layers.TCP)
	if int(tcp.SrcPort) != p.port && int(tcp.DstPort) != p.port {
		return
	}
	ts := time.Now()
	if md := pkt.Metadata(); md != nil && !md.Timestamp.IsZero() {
		ts = md.Timestamp
	}
	p.assembler.AssembleWithTimestamp(netLayer.NetworkFlow(), tcp, ts)
}

// Flush finishes all streams (end of file / shutdown) and waits for parsers.
func (p *Pipeline) Flush() {
	p.assembler.FlushAll()
	p.wg.Wait()
}

// FlushOld closes idle streams during long live captures.
func (p *Pipeline) FlushOld(age time.Duration) {
	p.assembler.FlushOlderThan(time.Now().Add(-age))
}

// -- TCP stream -> HTTP messages ---------------------------------------------

type streamFactory struct {
	port  int
	pairs *pairer
	wg    *sync.WaitGroup
}

func (f *streamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	r := tcpreader.NewReaderStream()
	isRequest := tcpFlow.Dst().String() == strconv.Itoa(f.port)
	// canonical connection key: always client-perspective (src of requests)
	var key string
	if isRequest {
		key = netFlow.String() + "|" + tcpFlow.String()
	} else {
		key = netFlow.Reverse().String() + "|" + tcpFlow.Reverse().String()
	}
	f.wg.Add(1)
	go f.consume(&r, isRequest, key)
	return &r
}

func (f *streamFactory) consume(r *tcpreader.ReaderStream, isRequest bool, key string) {
	defer f.wg.Done()
	buf := bufio.NewReader(r)
	for {
		if isRequest {
			req, err := http.ReadRequest(buf)
			if err != nil {
				if err != io.EOF && !errIsClosed(err) {
					tcpreader.DiscardBytesToEOF(buf)
				}
				return
			}
			body, _ := io.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
			req.Body.Close()
			f.pairs.addRequest(key, req, body)
		} else {
			resp, err := http.ReadResponse(buf, nil)
			if err != nil {
				if err != io.EOF && !errIsClosed(err) {
					tcpreader.DiscardBytesToEOF(buf)
				}
				return
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
			resp.Body.Close()
			f.pairs.addResponse(key, resp, body)
		}
	}
}

func errIsClosed(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "closed") ||
		strings.Contains(err.Error(), "EOF"))
}

// -- request/response pairing (FIFO per connection) ---------------------------

type pendingRequest struct {
	method  string
	path    string
	headers map[string]string
	body    any
}

type pendingResponse struct {
	status int
	body   string
}

type conn struct {
	requests  []pendingRequest
	responses []pendingResponse
}

type pairer struct {
	mu    sync.Mutex
	conns map[string]*conn
	sink  Sink
}

func (p *pairer) get(key string) *conn {
	c, ok := p.conns[key]
	if !ok {
		c = &conn{}
		p.conns[key] = c
	}
	return c
}

func (p *pairer) addRequest(key string, req *http.Request, body []byte) {
	headers := map[string]string{}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if skipHeaders[lk] {
			continue
		}
		headers[lk] = req.Header.Get(k)
	}
	var parsedBody any
	if len(body) > 0 {
		var parsed any
		if json.Unmarshal(body, &parsed) == nil {
			parsedBody = parsed
		} else {
			parsedBody = string(body)
		}
	}
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.get(key)
	c.requests = append(c.requests, pendingRequest{req.Method, path, headers, parsedBody})
	p.match(c)
}

func (p *pairer) addResponse(key string, resp *http.Response, body []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.get(key)
	c.responses = append(c.responses, pendingResponse{resp.StatusCode, string(body)})
	p.match(c)
}

func (p *pairer) match(c *conn) {
	for len(c.requests) > 0 && len(c.responses) > 0 {
		req, resp := c.requests[0], c.responses[0]
		c.requests, c.responses = c.requests[1:], c.responses[1:]
		p.sink(Exchange{
			Method: req.method, Path: req.path, Headers: req.headers, Body: req.body,
			Status: resp.status, RespBody: resp.body,
		})
	}
}

// -- output writers ------------------------------------------------------------

// Output appends exchanges to a corpus (.jsonl) or golden (.golden.jsonl) file.
type Output struct {
	mu      sync.Mutex
	golden  *golden.Writer
	rawPath string
	rawFile io.WriteCloser
	seq     int
	Count   int
}

func NewOutput(path, upstreamLabel string) (*Output, error) {
	o := &Output{}
	if golden.IsGoldenFile(path) {
		w, err := golden.NewWriter(path, golden.Meta{
			Upstream:   upstreamLabel,
			Serialized: false, // passive capture cannot attribute write-sets per request
			Source:     "sniff",
		})
		if err != nil {
			return nil, err
		}
		o.golden = w
		return o, nil
	}
	f, err := openAppend(path)
	if err != nil {
		return nil, err
	}
	o.rawFile, o.rawPath = f, path
	return o, nil
}

func (o *Output) Sink(ex Exchange) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seq++
	name := fmt.Sprintf("sniff-%d-%s-%s", o.seq, ex.Method, ex.Path)
	if o.golden != nil {
		if err := o.golden.Append(golden.Entry{
			Name: name, Method: ex.Method, Path: ex.Path,
			Headers: ex.Headers, Body: ex.Body,
			Expected: golden.Expected{Status: ex.Status, Body: ex.RespBody},
		}); err != nil {
			log.Printf("[sniff] write: %v", err)
			return
		}
	} else {
		entry := map[string]any{"name": name, "method": ex.Method, "path": ex.Path}
		if len(ex.Headers) > 0 {
			entry["headers"] = ex.Headers
		}
		if ex.Body != nil {
			entry["body"] = ex.Body
		}
		line, err := json.Marshal(entry)
		if err != nil {
			return
		}
		if _, err := o.rawFile.Write(append(line, '\n')); err != nil {
			log.Printf("[sniff] write: %v", err)
			return
		}
	}
	o.Count++
}

func (o *Output) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.golden != nil {
		o.golden.Close()
	}
	if o.rawFile != nil {
		o.rawFile.Close()
	}
}
