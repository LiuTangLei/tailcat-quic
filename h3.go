// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/fxamacker/cbor/v2"
	go4mem "go4.org/mem"
	"tailscale.com/types/key"
	"tailscale.com/wgengine/wgtransport/quicbind"
)

const (
	h3AddressPrefix  = "tch3"
	h3AddressVersion = 1
	h3MaxClients     = 256
	h3MeowSize       = 4 + 1 + 32 + 32 + 32 + sha256.Size
)

var h3MeowMagic = [4]byte{'m', 'h', '3', 1}

var h3CBORDecoder = func() cbor.DecMode {
	mode, err := (cbor.DecOptions{
		DupMapKey:       cbor.DupMapKeyEnforcedAPF,
		MaxNestedLevels: 16, MaxArrayElements: 1024, MaxMapPairs: 128,
		IndefLength: cbor.IndefLengthForbidden, TagsMd: cbor.TagsForbidden,
	}).DecMode()
	if err != nil {
		panic(err)
	} // fixed compile-time decoder policy
	return mode
}()

// newH3Factory installs one native-IP H3 backend, never a WireGuard device.
// Browser builds use the same authenticated backend over magicsock's DERP
// WebSocket PacketConn; they do not require a browser-accessible raw UDP socket.
// The ephemeral TLS certificate is authenticated by the persistent node key,
// not by TOFU or a public certificate authority. Both TLS directions additionally
// prove the connection secret through the session-bound node-auth transcript.
func newH3Factory(lb *locoBackend) (*quicbind.Factory, error) {
	if lb.presharedKey.IsZero() {
		return nil, errors.New("H3 requires a non-zero connection secret; generate a new H3 key or connection code")
	}
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("H3 TLS identity: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("H3 TLS serial: %w", err)
	}
	serial.Add(serial, big.NewInt(1))
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "tailcat.invalid"},
		DNSNames:     []string{"tailcat.invalid"},
		NotBefore:    now.Add(-time.Hour), NotAfter: now.AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &private.PublicKey, private)
	if err != nil {
		return nil, fmt.Errorf("H3 TLS certificate: %w", err)
	}
	return quicbind.NewFactoryWithCertificate(quicbind.Config{
		Version: 2, Payload: "ip", IO: "magicsock",
		LocalPublicKey: lb.pub.String(),
		HTTP3:          true, AutoTrust: true, Server: lb.isServer, BBRv3: true, TCPMSS: 1080,
		TCPStreams: true, TCPHandler: lb.tcpHandler, TCPNodeAddress: h3NodeAddress,
		AuthenticationSecret: [32]byte(lb.presharedKey),
		HTTP3URL:             "https://tailcat.invalid/.well-known/masque/ip/*/*/",
	}, tls.Certificate{Certificate: [][]byte{der}, PrivateKey: private})
}

// h3Meow is an authenticated bootstrap announcement, not application-data
// authentication. The same secret is independently bound to the actual H3 TLS
// session by quicbind. Thus a DERP observer cannot reuse this MAC to terminate an
// application tunnel of its own. Direction, identities and a fresh client
// lifetime nonce prevent reflection, substitution and stale acknowledgment.
func encodeH3Meow(secret PresharedKey, server, client key.NodePublic, disco key.DiscoPublic, nonce [32]byte, reply bool) []byte {
	if secret.IsZero() || server.IsZero() || client.IsZero() || disco.IsZero() || nonce == ([32]byte{}) {
		return nil
	}
	p := make([]byte, 0, h3MeowSize)
	p = append(p, h3MeowMagic[:]...)
	direction := byte(1)
	if reply {
		direction = 2
	}
	p = append(p, direction)
	p = client.AppendTo(p)
	p = disco.AppendTo(p)
	p = append(p, nonce[:]...)
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte("tailcat-h3/bootstrap/v1\x00"))
	mac.Write(server.AppendTo(nil))
	mac.Write(p)
	return append(p, mac.Sum(nil)...)
}

func isH3Meow(p []byte) bool {
	return len(p) >= len(h3MeowMagic) && [4]byte(p[:4]) == h3MeowMagic
}

func parseH3Meow(p []byte, secret PresharedKey, server, expectedClient key.NodePublic, reply bool) (key.DiscoPublic, [32]byte, bool) {
	var nonce [32]byte
	var zero key.DiscoPublic
	if len(p) != h3MeowSize || !isH3Meow(p) || secret.IsZero() || server.IsZero() || expectedClient.IsZero() {
		return zero, nonce, false
	}
	direction := byte(1)
	if reply {
		direction = 2
	}
	if p[4] != direction || key.NodePublicFromRaw32(go4mem.B(p[5:37])) != expectedClient {
		return zero, nonce, false
	}
	disco := key.DiscoPublicFromRaw32(go4mem.B(p[37:69]))
	copy(nonce[:], p[69:101])
	if disco.IsZero() || nonce == ([32]byte{}) {
		return zero, nonce, false
	}
	expected := encodeH3Meow(secret, server, expectedClient, disco, nonce, reply)
	if !hmac.Equal(p, expected) {
		return zero, [32]byte{}, false
	}
	return disco, nonce, true
}
