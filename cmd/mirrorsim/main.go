// mirrorsim simulates AWS VPC Traffic Mirroring for local testing:
// a TCP passthrough (stand-in for the gateway hop AWS would mirror) that
// emits VXLAN-encapsulated copies of every byte to a mirror target —
// the exact wire format `mockingbox mirror` receives in production.
//
//	mirrorsim --listen :10000 --upstream 127.0.0.1:10103 --mirror 127.0.0.1:4789
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const chunkSize = 1400

type mirrorSender struct {
	mu   sync.Mutex
	conn net.Conn
}

func (m *mirrorSender) send(pkt []byte) {
	vxlan := append([]byte{0x08, 0, 0, 0, 0, 0, 42, 0}, pkt...)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn.Write(vxlan) //nolint:errcheck — mirroring is fail-open by design
}

type flowState struct {
	clientPort uint16
	serverPort uint16
	seqC, seqS uint32
	mirror     *mirrorSender
}

func (f *flowState) packet(fromClient bool, seq uint32, syn, fin bool, chunk []byte) {
	srcIP, dstIP := net.IP{10, 9, 0, 1}, net.IP{10, 9, 0, 2}
	srcPort, dstPort := f.clientPort, f.serverPort
	if !fromClient {
		srcIP, dstIP = dstIP, srcIP
		srcPort, dstPort = f.serverPort, f.clientPort
	}
	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{2, 0, 0, 0, 0, 1},
		DstMAC:       net.HardwareAddr{2, 0, 0, 0, 0, 2},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{Version: 4, TTL: 64, SrcIP: srcIP, DstIP: dstIP,
		Protocol: layers.IPProtocolTCP}
	tcp := layers.TCP{
		SrcPort: layers.TCPPort(srcPort), DstPort: layers.TCPPort(dstPort),
		Seq: seq, SYN: syn, FIN: fin, ACK: !syn || !fromClient, PSH: len(chunk) > 0,
		Window: 65535,
	}
	tcp.SetNetworkLayerForChecksum(&ip) //nolint:errcheck
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, &eth, &ip, &tcp, gopacket.Payload(chunk)); err != nil {
		return
	}
	f.mirror.send(buf.Bytes())
}

// handshake announces stream starts — without SYNs the reassembler cannot
// know the initial sequence and buffers everything until a forced flush
// (real VPC mirroring copies the actual handshake, so production sees SYNs).
func (f *flowState) handshake() {
	f.packet(true, f.seqC-1, true, false, nil)  // SYN
	f.packet(false, f.seqS-1, true, false, nil) // SYN-ACK
}

func (f *flowState) finish() {
	f.packet(true, f.seqC, false, true, nil)
	f.packet(false, f.seqS, false, true, nil)
}

func (f *flowState) emit(fromClient bool, payload []byte) {
	seq := &f.seqC
	if !fromClient {
		seq = &f.seqS
	}
	for off := 0; off < len(payload); off += chunkSize {
		end := min(off+chunkSize, len(payload))
		chunk := payload[off:end]
		f.packet(fromClient, *seq, false, false, chunk)
		*seq += uint32(len(chunk))
	}
}

func main() {
	listen := flag.String("listen", ":10000", "TCP listen address (gateway entry)")
	upstream := flag.String("upstream", "127.0.0.1:10103", "forward target (the old service)")
	mirror := flag.String("mirror", "127.0.0.1:4789", "VXLAN mirror target (mockingbox mirror)")
	flag.Parse()

	mirrorConn, err := net.Dial("udp", *mirror)
	if err != nil {
		log.Fatalf("mirror target: %v", err)
	}
	sender := &mirrorSender{conn: mirrorConn}

	_, portStr, _ := strings.Cut(*upstream, ":")
	servicePort, _ := strconv.Atoi(portStr)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("mirrorsim: %s -> %s (VXLAN copies -> %s, inner port %d)",
		*listen, *upstream, *mirror, servicePort)

	var connSeq atomic.Uint32
	for {
		client, err := ln.Accept()
		if err != nil {
			log.Fatalf("accept: %v", err)
		}
		go handle(client, *upstream, sender, uint16(servicePort), uint16(20000+connSeq.Add(1)))
	}
}

func handle(client net.Conn, upstream string, sender *mirrorSender, servicePort, clientPort uint16) {
	defer client.Close()
	server, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("upstream dial: %v", err)
		return
	}
	defer server.Close()

	flow := &flowState{
		clientPort: clientPort, serverPort: servicePort,
		seqC: 1000, seqS: 5000, mirror: sender,
	}
	flow.handshake()
	defer flow.finish()

	var wg sync.WaitGroup
	pipe := func(dst, src net.Conn, fromClient bool) {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				flow.emit(fromClient, buf[:n])
				if _, werr := dst.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					_ = err
				}
				break
			}
		}
		if tcp, ok := dst.(*net.TCPConn); ok {
			tcp.CloseWrite() //nolint:errcheck
		}
	}
	wg.Add(2)
	go pipe(server, client, true)
	go pipe(client, server, false)
	wg.Wait()
}
