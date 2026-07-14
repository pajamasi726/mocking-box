package sniff

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/pajamasi726/mocking-box/internal/golden"
)

// writeTestPcap builds a .pcap file containing one HTTP session, exactly what
// `tcpdump -w` would produce — validates the offline convert path end-to-end.
func writeTestPcap(t *testing.T, path string, vxlan bool) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := pcapgo.NewWriter(f)
	if err := w.WriteFileHeader(65535, layers.LinkTypeEthernet); err != nil {
		t.Fatal(err)
	}

	write := func(pkt gopacket.Packet) {
		data := pkt.Data()
		if vxlan {
			data = wrapVXLANFrame(t, data)
		}
		err := w.WritePacket(gopacket.CaptureInfo{
			Timestamp: time.Now(), CaptureLength: len(data), Length: len(data),
		}, data)
		if err != nil {
			t.Fatal(err)
		}
	}
	write(mkPacket(t, true, 1000, httpReq(`{"amount": 7000}`)))
	write(mkPacket(t, false, 2000, httpResp(`{"balance": 57000}`)))
}

// wrapVXLANFrame wraps an inner Ethernet frame the way VPC Traffic Mirroring
// delivers it: outer Ethernet/IP/UDP(4789) + VXLAN header + inner frame.
func wrapVXLANFrame(t *testing.T, inner []byte) []byte {
	t.Helper()
	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 1, 2, 3, 4, 5},
		DstMAC:       net.HardwareAddr{0xbb, 5, 4, 3, 2, 1},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{Version: 4, TTL: 64, SrcIP: net.IP{192, 168, 0, 1},
		DstIP: net.IP{192, 168, 0, 2}, Protocol: layers.IPProtocolUDP}
	udp := layers.UDP{SrcPort: 50000, DstPort: 4789}
	if err := udp.SetNetworkLayerForChecksum(&ip); err != nil {
		t.Fatal(err)
	}
	vxlanHdr := []byte{0x08, 0, 0, 0, 0, 0, 42, 0}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	payload := append(vxlanHdr, inner...)
	if err := gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, gopacket.Payload(payload)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestConvertPcapFileToGolden(t *testing.T) {
	dir := t.TempDir()
	pcapPath := filepath.Join(dir, "capture.pcap")
	writeTestPcap(t, pcapPath, false)

	outPath := filepath.Join(dir, "converted.golden.jsonl")
	output, err := NewOutput(outPath, "test")
	if err != nil {
		t.Fatal(err)
	}
	pipeline := NewPipeline(testPort, output.Sink)
	if err := RunFile(pcapPath, testPort, false, pipeline); err != nil {
		t.Fatal(err)
	}
	output.Close()

	if output.Count != 1 {
		t.Fatalf("expected 1 exchange, got %d", output.Count)
	}
	_, entries, err := golden.Read(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Expected.Status != 200 {
		t.Fatalf("unexpected golden entries: %+v", entries)
	}
	if entries[0].Path != "/wallet/1/charge" {
		t.Errorf("unexpected path %q", entries[0].Path)
	}
}

func TestConvertVXLANPcap(t *testing.T) {
	dir := t.TempDir()
	pcapPath := filepath.Join(dir, "mirrored.pcap")
	writeTestPcap(t, pcapPath, true)

	outPath := filepath.Join(dir, "mirrored.jsonl")
	output, err := NewOutput(outPath, "test")
	if err != nil {
		t.Fatal(err)
	}
	pipeline := NewPipeline(testPort, output.Sink)
	if err := RunFile(pcapPath, testPort, true, pipeline); err != nil {
		t.Fatal(err)
	}
	output.Close()
	if output.Count != 1 {
		t.Fatalf("expected 1 exchange from VXLAN pcap, got %d", output.Count)
	}
}

// TestRunMirrorUDP exercises the real VPC-mirror receiver: VXLAN datagrams
// arrive over UDP exactly as AWS delivers them.
func TestRunMirrorUDP(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "mirror.golden.jsonl")
	output, err := NewOutput(outPath, "test")
	if err != nil {
		t.Fatal(err)
	}
	pipeline := NewPipeline(testPort, output.Sink)

	stop := make(chan struct{})
	done := make(chan error, 1)
	const listen = "127.0.0.1:14789"
	go func() { done <- RunMirror(listen, testPort, pipeline, stop) }()
	time.Sleep(150 * time.Millisecond)

	conn, err := net.Dial("udp", listen)
	if err != nil {
		t.Fatal(err)
	}
	vxlanWrap := func(pkt gopacket.Packet) []byte {
		hdr := []byte{0x08, 0, 0, 0, 0, 0, 42, 0}
		return append(hdr, pkt.Data()...)
	}
	if _, err := conn.Write(vxlanWrap(mkPacket(t, true, 1000, httpReq(`{"amount": 1}`)))); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(vxlanWrap(mkPacket(t, false, 2000, httpResp(`{"ok": 1}`)))); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	time.Sleep(300 * time.Millisecond)

	close(stop)
	if err := <-done; err != nil {
		t.Fatalf("mirror: %v", err)
	}
	output.Close()

	if output.Count != 1 {
		t.Fatalf("expected 1 exchange via UDP mirror, got %d", output.Count)
	}
}
