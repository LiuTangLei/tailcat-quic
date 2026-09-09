//go:build tailcat_perf

package tailcat

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"tailscale.com/wgengine/wgtransport"
	"tailscale.com/wgengine/wgtransport/quicbind"
)

// Explicit diagnostic builds only. Never logs credentials, addresses or peer keys.
func instrumentH3Factory(f wgtransport.Factory) wgtransport.Factory { return perfFactory{f} }

type perfFactory struct{ wgtransport.Factory }
type perfBackend struct {
	*quicbind.Backend
	done chan struct{}
	once sync.Once
}

func (f perfFactory) New(h wgtransport.Host) (wgtransport.Backend, error) {
	backend, err := f.Factory.New(h)
	if err != nil { return nil, err }
	b := &perfBackend{Backend: backend.(*quicbind.Backend), done: make(chan struct{})}
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-b.done: return
			case <-tick.C:
				snapshot := b.Snapshot()
				out := make(map[string]any)
				for _, key := range []string{"sent_packets", "received_packets", "send_drops", "send_errors", "receive_drops", "raw_drops", "fragmented_packets", "malformed_frames", "enqueue_waits", "tx_queued_bytes", "rx_queued_bytes", "connections", "active_connections", "tcp_streams", "tcp_stream_bytes_sent", "tcp_stream_bytes_received", "connection_stats", "quic_receive_queue_drops", "ip_data_plane"} {
					if value, ok := snapshot[key]; ok { out[key] = value }
				}
				if len(out) > 0 {
					data, err := json.Marshal(out)
					if err == nil { log.Printf("tailcat-perf %s", data) }
				}
			}
		}
	}()
	return b, nil
}
func (b *perfBackend) Close() error {
	b.once.Do(func(){ close(b.done) })
	return b.Backend.Close()
}
