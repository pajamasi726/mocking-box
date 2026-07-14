package sniff

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gopacket/gopacket/pcapgo"
)

func openAppend(path string) (io.WriteCloser, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

// RunLive sniffs a network interface (requires root or CAP_NET_RAW).
// stop: closing the channel ends the capture.
func RunLive(iface string, port int, pipeline *Pipeline, stop <-chan struct{}) error {
	handle, err := pcap.OpenLive(iface, 65535, false, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("open %s (need root or CAP_NET_RAW — see README): %w", iface, err)
	}
	defer handle.Close()
	if err := handle.SetBPFFilter(fmt.Sprintf("tcp port %d", port)); err != nil {
		return err
	}
	log.Printf("[sniff] listening on %s (tcp port %d)", iface, port)

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	source.NoCopy = true
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			pipeline.Flush()
			return nil
		case <-ticker.C:
			pipeline.FlushOld(2 * time.Minute)
		case pkt, ok := <-source.Packets():
			if !ok {
				pipeline.Flush()
				return nil
			}
			pipeline.HandlePacket(pkt)
		}
	}
}

// RunFile converts an offline .pcap file (captured with tcpdump/wireshark).
// Pure Go (pcapgo) — no libpcap or privileges needed.
func RunFile(path string, port int, vxlan bool, pipeline *Pipeline) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	reader, err := pcapgo.NewReader(f)
	if err != nil {
		return fmt.Errorf("parse pcap %s: %w", path, err)
	}
	linkType := layers.LinkType(reader.LinkType())
	for {
		data, ci, err := reader.ReadPacketData()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		pkt := gopacket.NewPacket(data, linkType, gopacket.Default)
		pkt.Metadata().Timestamp = ci.Timestamp
		if vxlan {
			if inner := decapVXLAN(pkt); inner != nil {
				pipeline.HandlePacket(inner)
			}
			continue
		}
		pipeline.HandlePacket(pkt)
	}
	pipeline.Flush()
	return nil
}

// RunMirror receives AWS VPC Traffic Mirroring streams: VXLAN-encapsulated
// packets over UDP (default port 4789). Zero-touch on the mirrored host.
func RunMirror(listen string, port int, pipeline *Pipeline, stop <-chan struct{}) error {
	addr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer sock.Close()
	log.Printf("[mirror] receiving VXLAN on %s (inner tcp port %d)", listen, port)

	go func() {
		<-stop
		sock.Close()
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			pipeline.FlushOld(2 * time.Minute)
		}
	}()

	buf := make([]byte, 65535)
	for {
		n, _, err := sock.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-stop:
				pipeline.Flush()
				return nil
			default:
				return err
			}
		}
		if inner := decapVXLANPayload(buf[:n]); inner != nil {
			pipeline.HandlePacket(inner)
		}
	}
}

// decapVXLANPayload decodes a raw VXLAN datagram payload (header + inner frame).
func decapVXLANPayload(datagram []byte) gopacket.Packet {
	vx := gopacket.NewPacket(datagram, layers.LayerTypeVXLAN, gopacket.Default)
	vxLayer := vx.Layer(layers.LayerTypeVXLAN)
	if vxLayer == nil {
		return nil
	}
	inner := gopacket.NewPacket(vxLayer.LayerPayload(), layers.LayerTypeEthernet, gopacket.Default)
	return inner
}

// decapVXLAN extracts the inner frame from a fully captured VXLAN/UDP packet
// (as seen in a pcap of mirrored traffic).
func decapVXLAN(pkt gopacket.Packet) gopacket.Packet {
	udpLayer := pkt.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return nil
	}
	return decapVXLANPayload(udpLayer.LayerPayload())
}
