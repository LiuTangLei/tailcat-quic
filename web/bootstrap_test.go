package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"tailscale.com/tstest/integration"
)

func TestBrowserBootstrapState(t *testing.T) {
	bin := preflight(t)
	dm := integration.RunDERPAndSTUN(t, mkLogf(t, "bootstrap"), "127.0.0.1")
	srv := newWebServer(t, dm)
	ctx, cancel := context.WithTimeout(launchChrome(t, bin), 40*time.Second)
	defer cancel()
	if err := chromedp.Run(ctx, chromedp.Navigate(srv.URL+"/?mode=listen&sink=hash")); err != nil {
		t.Fatal(err)
	}
	var ready bool
	err := chromedp.Run(ctx, chromedp.Poll("window.tcTest && (window.tcTest.listenAddr !== null || window.tcTest.errors.length > 0)", &ready, chromedp.WithPollingTimeout(20*time.Second)))
	var state json.RawMessage
	diagErr := chromedp.Run(ctx, chromedp.Evaluate(`({status:document.getElementById('status')?.textContent, ready:window.tcTest?.ready, h3Address:window.tcTest?.listenAddr?.startsWith('tch3'), errors:window.tcTest?.errors, wasm:typeof Go, listen:typeof tailcatListen})`, &state))
	t.Logf("bootstrap state: %s (diagnostics: %v)", state, diagErr)
	if err != nil {
		t.Fatal(err)
	}
	var address string
	if err := chromedp.Run(ctx, chromedp.Evaluate("window.tcTest.listenAddr || ''", &address)); err != nil || address == "" {
		t.Fatalf("browser listener not available: %v", err)
	}
}
