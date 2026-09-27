package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/tailscale/tailcat"
	"tailscale.com/tstest/integration"
)

// Reuse one page for many real authenticated connections. Exercise saved
// methods and repeated close after each complete, hash-checked transfer.
func TestBrowserRepeatedSends(t *testing.T) {
	bin := preflight(t)
	dm := integration.RunDERPAndSTUN(t, mkLogf(t, "lifecycle"), "127.0.0.1")
	web := newWebServer(t, dm)
	const rounds, size = 8, 4 << 20
	type received struct {
		n   int64
		sum string
		err error
	}
	results := make(chan received, rounds)
	s := &tailcat.Server{Logf: mkLogf(t, "server"), Region: dm.Regions[1]}
	s.OnTCP = func(uint16) func(net.Conn) {
		return func(c net.Conn) {
			defer c.Close()
			h := sha256.New()
			n, err := io.Copy(h, c)
			results <- received{n, hex.EncodeToString(h.Sum(nil)), err}
		}
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(launchChrome(t, bin), 120*time.Second)
	defer cancel()
	var ready bool
	if err := chromedp.Run(ctx, chromedp.Navigate(web.URL), chromedp.Poll("window.tcTest?.ready === true", &ready)); err != nil {
		t.Fatal(err)
	}
	addr, _ := json.Marshal(string(s.TailcatAddr()))
	script := fmt.Sprintf(`(async () => {
 const payload = new Uint8Array(65536);
 window.lifecycle = {completed:0,error:null};
 try {
  for(let i=0;i<%d;i++) {
   const conn=await tailcatDial({addr:%s,derpMapURL:location.origin+'/derpmap.json'});
   const savedClose=conn.close, savedRead=conn.read;
   try {
    for(let n=0;n<%d;n+=payload.length) await conn.write(payload);
    await conn.closeWrite();
    while(await conn.read() !== null) {}
   } finally { conn.close(); }
   savedClose();
   let rejected=false;
   try { await savedRead(); } catch { rejected=true; }
   if(!rejected) throw new Error('closed method remained active');
   window.lifecycle.completed++;
  }
 } catch(e) {window.lifecycle.error=String(e);}
})();`, rounds, string(addr), size)
	var done bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil), chromedp.Poll(fmt.Sprintf("window.lifecycle.completed === %d || window.lifecycle.error !== null", rounds), &done, chromedp.WithPollingTimeout(100*time.Second))); err != nil {
		t.Fatal(err)
	}
	var state struct {
		Completed int
		Error     *string
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate("window.lifecycle", &state)); err != nil {
		t.Fatal(err)
	}
	if state.Error != nil || state.Completed != rounds {
		t.Fatalf("incomplete browser lifecycle: %+v", state)
	}
	want := sha256.Sum256(make([]byte, size))
	for range rounds {
		select {
		case r := <-results:
			if r.err != nil || r.n != size || r.sum != hex.EncodeToString(want[:]) {
				t.Fatalf("transfer mismatch: %+v", r)
			}
		case <-ctx.Done():
			t.Fatal("missing receiver result")
		}
	}
	checkPageErrors(t, ctx)
}
