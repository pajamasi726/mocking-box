package sniff

import (
	"fmt"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const testPort = 8080

func mkPacket(t *testing.T, fromClient bool, seq uint32, payload string) gopacket.Packet {
	t.Helper()
	srcIP, dstIP := net.IP{10, 0, 0, 1}, net.IP{10, 0, 0, 2}
	srcPort, dstPort := layers.TCPPort(5555), layers.TCPPort(testPort)
	if !fromClient {
		srcIP, dstIP = dstIP, srcIP
		srcPort, dstPort = dstPort, srcPort
	}
	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{1, 2, 3, 4, 5, 6},
		DstMAC:       net.HardwareAddr{6, 5, 4, 3, 2, 1},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{Version: 4, TTL: 64, SrcIP: srcIP, DstIP: dstIP, Protocol: layers.IPProtocolTCP}
	tcp := layers.TCP{SrcPort: srcPort, DstPort: dstPort, Seq: seq, ACK: true, PSH: true, Window: 65535}
	if err := tcp.SetNetworkLayerForChecksum(&ip); err != nil {
		t.Fatal(err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, &eth, &ip, &tcp, gopacket.Payload(payload)); err != nil {
		t.Fatal(err)
	}
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

func httpReq(body string) string {
	return fmt.Sprintf(
		"POST /wallet/1/charge HTTP/1.1\r\nHost: demo\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
		len(body), body)
}

func httpResp(body string) string {
	return fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
		len(body), body)
}

func TestPipelineReassemblesAndPairs(t *testing.T) {
	var got []Exchange
	p := NewPipeline(testPort, func(ex Exchange) { got = append(got, ex) })

	reqBody := `{"amount": 5000}`
	respBody := `{"balance": 55000, "wallet_id": 1}`
	p.HandlePacket(mkPacket(t, true, 1000, httpReq(reqBody)))
	p.HandlePacket(mkPacket(t, false, 2000, httpResp(respBody)))
	p.Flush()

	if len(got) != 1 {
		t.Fatalf("expected 1 exchange, got %d", len(got))
	}
	ex := got[0]
	if ex.Method != "POST" || ex.Path != "/wallet/1/charge" {
		t.Errorf("unexpected request: %+v", ex)
	}
	if ex.Status != 200 || ex.RespBody != respBody {
		t.Errorf("unexpected response: status=%d body=%q", ex.Status, ex.RespBody)
	}
	body, ok := ex.Body.(map[string]any)
	if !ok || body["amount"] != float64(5000) {
		t.Errorf("body not parsed as JSON: %#v", ex.Body)
	}
	if _, has := ex.Headers["host"]; has {
		t.Errorf("host header should be skipped")
	}
}

func TestPipelineMultipleRequestsSameConnection(t *testing.T) {
	var got []Exchange
	p := NewPipeline(testPort, func(ex Exchange) { got = append(got, ex) })

	req1, resp1 := httpReq(`{"amount": 1}`), httpResp(`{"n": 1}`)
	req2, resp2 := httpReq(`{"amount": 2}`), httpResp(`{"n": 2}`)
	p.HandlePacket(mkPacket(t, true, 1000, req1))
	p.HandlePacket(mkPacket(t, false, 2000, resp1))
	p.HandlePacket(mkPacket(t, true, 1000+uint32(len(req1)), req2))
	p.HandlePacket(mkPacket(t, false, 2000+uint32(len(resp1)), resp2))
	p.Flush()

	if len(got) != 2 {
		t.Fatalf("expected 2 exchanges, got %d", len(got))
	}
	if got[0].RespBody != `{"n": 1}` || got[1].RespBody != `{"n": 2}` {
		t.Errorf("pairing order wrong: %+v", got)
	}
}

func TestVXLANDecap(t *testing.T) {
	var got []Exchange
	p := NewPipeline(testPort, func(ex Exchange) { got = append(got, ex) })

	wrap := func(pkt gopacket.Packet) []byte {
		// VXLAN header: flags(0x08 = VNI valid) + reserved + VNI(123) + reserved
		hdr := []byte{0x08, 0, 0, 0, 0, 0, 123 >> 0, 0}
		hdr[4], hdr[5], hdr[6] = 0, 0, 123
		return append(hdr, pkt.Data()...)
	}

	reqDatagram := wrap(mkPacket(t, true, 1000, httpReq(`{"amount": 9}`)))
	respDatagram := wrap(mkPacket(t, false, 2000, httpResp(`{"ok": true}`)))

	if inner := decapVXLANPayload(reqDatagram); inner != nil {
		p.HandlePacket(inner)
	} else {
		t.Fatal("request decap failed")
	}
	if inner := decapVXLANPayload(respDatagram); inner != nil {
		p.HandlePacket(inner)
	} else {
		t.Fatal("response decap failed")
	}
	p.Flush()

	if len(got) != 1 {
		t.Fatalf("expected 1 exchange via VXLAN, got %d", len(got))
	}
	if got[0].Status != 200 || got[0].Path != "/wallet/1/charge" {
		t.Errorf("unexpected exchange: %+v", got[0])
	}
}

func TestOtherPortsIgnored(t *testing.T) {
	var got []Exchange
	p := NewPipeline(9999, func(ex Exchange) { got = append(got, ex) })
	p.HandlePacket(mkPacket(t, true, 1000, httpReq(`{}`)))
	p.Flush()
	if len(got) != 0 {
		t.Errorf("expected no exchanges for filtered port, got %d", len(got))
	}
}
