package tailcat

import (
	"net"
	"net/netip"

	"tailscale.com/types/key"
	"tailscale.com/wgengine/wgtransport"
	"tailscale.com/wgengine/wgtransport/quicbind"
)

type h3CaptureFactory struct {
	*quicbind.Factory
	lb *locoBackend
}
func (f h3CaptureFactory) New(host wgtransport.Host) (wgtransport.Backend, error) {
	b, err := f.Factory.New(host)
	if err != nil { return nil, err }
	f.lb.h3Backend = b.(*quicbind.Backend)
	return b, nil
}

func (s *Server) h3TCPHandler(lb *locoBackend) func([32]byte, netip.AddrPort) func(net.Conn) {
	return func(peer [32]byte, dst netip.AddrPort) func(net.Conn) {
		// The backend obtains peer exclusively from the authenticated QUIC
		// session. Keep the same live allowlist and service/port gates as the
		// IP path, even though TCP payload no longer enters the packet filter.
		var pub key.NodePublic
		if err := pub.UnmarshalText([]byte("nodekey:"+hexNodeKey(peer))); err != nil { return nil }
		if _, ok := lb.peerConfig(pub); !ok { return nil }
		if dst.Addr() == lb.addr {
			allowed := s.ServedTCPPorts == nil
			for _, r := range s.ServedTCPPorts {
				if dst.Port() >= r.First && dst.Port() <= r.Last { allowed = true; break }
			}
			if !allowed || s.OnTCP == nil { return nil }
			return s.OnTCP(dst.Port())
		}
		if s.OnTCPForward == nil { return nil }
		if nat64Prefix.Contains(dst.Addr()) {
			a := dst.Addr().As16()
			var v4 [4]byte
			copy(v4[:], a[12:])
			dst = netip.AddrPortFrom(netip.AddrFrom4(v4), dst.Port())
		}
		return s.OnTCPForward(dst)
	}
}

func hexNodeKey(k [32]byte) string {
	const digits = "0123456789abcdef"
	var b [64]byte
	for i, v := range k { b[i*2],b[i*2+1] = digits[v>>4],digits[v&15] }
	return string(b[:])
}

func h3NodeAddress(k [32]byte) netip.Addr {
	var address [16]byte
	copy(address[:], []byte{0xfd, 0x7a, 0x11, 0x5c, 0xa1, 0xe0})
	copy(address[6:], k[:10])
	return netip.AddrFrom16(address)
}
