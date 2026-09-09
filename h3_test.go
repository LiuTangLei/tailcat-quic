package tailcat

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"tailscale.com/types/key"
)

func TestH3BootstrapAuthentication(t *testing.T) {
	secret := NewPresharedKey()
	server, client := key.NewNode().Public(), key.NewNode().Public()
	disco := key.NewDisco().Public()
	nonce := [32]byte{1, 2, 3, 4, 5}
	for _, reply := range []bool{false, true} {
		packet := encodeH3Meow(secret, server, client, disco, nonce, reply)
		if len(packet) != h3MeowSize {
			t.Fatal("wrong bootstrap size")
		}
		got, n, ok := parseH3Meow(packet, secret, server, client, reply)
		if !ok || got != disco || n != nonce {
			t.Fatal("valid bootstrap rejected")
		}
		for i := range packet {
			tampered := bytes.Clone(packet)
			tampered[i] ^= 1
			if _, _, ok := parseH3Meow(tampered, secret, server, client, reply); ok {
				t.Fatalf("tampered byte %d accepted", i)
			}
		}
		for _, bad := range [][]byte{nil, packet[:4], packet[:len(packet)-1], append(bytes.Clone(packet), 0)} {
			if _, _, ok := parseH3Meow(bad, secret, server, client, reply); ok {
				t.Fatal("malformed bootstrap accepted")
			}
		}
		if _, _, ok := parseH3Meow(packet, secret, server, client, !reply); ok {
			t.Fatal("reflected bootstrap accepted")
		}
		if _, _, ok := parseH3Meow(packet, NewPresharedKey(), server, client, reply); ok {
			t.Fatal("wrong secret accepted")
		}
		if _, _, ok := parseH3Meow(packet, secret, key.NewNode().Public(), client, reply); ok {
			t.Fatal("wrong server accepted")
		}
		if _, _, ok := parseH3Meow(packet, secret, server, key.NewNode().Public(), reply); ok {
			t.Fatal("substituted DERP source accepted")
		}
	}
	if encodeH3Meow(PresharedKey{}, server, client, disco, nonce, false) != nil {
		t.Fatal("zero secret encoded")
	}
	if encodeH3Meow(secret, server, client, disco, [32]byte{}, false) != nil {
		t.Fatal("zero nonce encoded")
	}
}

func TestH3CodeValidation(t *testing.T) {
	p := NewPrivateKey()
	for _, tc := range []struct {
		name, field string
		value       any
	}{
		{"version-missing", "v", nil}, {"version-future", "v", 2}, {"version-negative", "v", -1},
		{"secret-missing", "q", nil}, {"secret-zero", "q", make([]byte, 32)}, {"secret-short", "q", make([]byte, 31)}, {"secret-long", "q", make([]byte, 33)},
		{"node-zero", "p", make([]byte, 32)}, {"node-short", "p", make([]byte, 31)}, {"node-long", "p", make([]byte, 33)},
		{"disco-missing", "k", nil}, {"disco-zero", "k", make([]byte, 32)}, {"disco-short", "k", make([]byte, 31)}, {"disco-long", "k", make([]byte, 33)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{"v": 1, "p": p.Public.ServerPublic.AppendTo(nil), "k": p.Public.ServerDiscoPublic.AppendTo(nil), "q": p.Public.PresharedKey[:], "i": 1}
			fields[tc.field] = tc.value
			raw, err := cbor.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			addr := Addr(h3AddressPrefix + base64.RawURLEncoding.EncodeToString(raw))
			if _, err := ParseAddr(addr); err == nil {
				t.Fatal("invalid H3 credential accepted")
			}
		})
	}
	p.Public.RegionID = 1
	code := p.Public.Addr()
	if _, err := ParseAddr(code); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAddr(Addr("tc" + strings.TrimPrefix(string(code), h3AddressPrefix))); err == nil {
		t.Fatal("official protocol prefix accepted")
	}
	if _, err := ParseAddr(Addr(h3AddressPrefix + strings.Repeat("A", 64<<10))); err == nil {
		t.Fatal("oversized code accepted")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(string(code), h3AddressPrefix))
	if err != nil {
		t.Fatal(err)
	}
	if raw[0] != 0xa5 {
		t.Fatalf("unexpected fixture map header %x", raw[0])
	}
	raw[0]++
	raw = append(raw, 0x61, 'v', 0x01)
	if _, err := ParseAddr(Addr(h3AddressPrefix + base64.RawURLEncoding.EncodeToString(raw))); err == nil {
		t.Fatal("duplicate CBOR credential field accepted")
	}
}

func TestH3FactoryRequiresSecret(t *testing.T) {
	priv := key.NewNode()
	lb := newLocoBackend(priv, PresharedKey{})
	if _, err := newH3Factory(lb); err == nil {
		t.Fatal("factory admitted a missing application credential")
	}
	lb.presharedKey = NewPresharedKey()
	if _, err := newH3Factory(lb); err != nil {
		t.Fatal(err)
	}
}

func FuzzH3ConnectionCode(f *testing.F) {
	p := NewPrivateKey()
	p.Public.RegionID = 1
	for _, s := range []string{"", "tc", "tch3", "tch3%", string(p.Public.Addr()), "tch3oWF2AQ"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 128<<10 {
			return
		}
		_, _ = ParseAddrRaw(Addr(s))
		ci, err := ParseAddr(Addr(s))
		if err != nil {
			return
		}
		back, err := ParseAddr(ci.Addr())
		if err != nil {
			t.Fatal(err)
		}
		if back.ServerPublic != ci.ServerPublic || back.ServerDiscoPublic != ci.ServerDiscoPublic || back.PresharedKey != ci.PresharedKey {
			t.Fatal("credential changed on round trip")
		}
	})
}
