//go:build cgo

package quic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onsi/gomega"

	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

func waitForPeerCount(g gomega.Gomega, tr *QuicTransport, want int) {
	g.Eventually(func() int { return tr.PeerCount() }, 10*time.Second, 50*time.Millisecond).
		Should(gomega.Equal(want))
}

// A peer that goes away must leave the connection map, or PeerCount and
// Healthy report it forever and every Publish keeps sending to it.
func TestHandleConnection_RemovesDisconnectedPeer(t *testing.T) {
	g := gomega.NewWithT(t)

	coord, err := New(WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer coord.Close()

	peer, err := New(WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ticket, err := coord.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(peer.Connect(ticket)).To(gomega.Succeed())
	waitForPeerCount(g, coord, 1)

	g.Expect(peer.Close()).To(gomega.Succeed())

	g.Eventually(func() int { return coord.PeerCount() }, 15*time.Second, 50*time.Millisecond).
		Should(gomega.Equal(0), "a disconnected peer must be removed from the connection map")
}

// silentEndpoint accepts connections and streams but never acks: the shape of
// a peer that wedges a pooled send.
func silentEndpoint(t *testing.T) (*iroh_ffi.Endpoint, string) {
	t.Helper()

	cfg := defaultConfig()

	preset := cfg.presetFn()
	bindAddr := "127.0.0.1:0"

	endpoint, err := iroh_ffi.EndpointBind(iroh_ffi.EndpointOptions{
		Preset:   &preset,
		BindAddr: &bindAddr,
		Alpns:    &[][]byte{DefaultALPN},
	})
	if err != nil {
		t.Fatalf("bind silent endpoint: %v", err)
	}

	ticket, err := iroh_ffi.EndpointTicketFromAddr(endpoint.Addr())
	if err != nil {
		t.Fatalf("silent endpoint ticket: %v", err)
	}

	go func() {
		for {
			incoming := endpoint.AcceptNext()
			if incoming == nil || *incoming == nil {
				return
			}

			accepting, err := (*incoming).Accept()
			if err != nil {
				continue
			}

			conn, err := accepting.Connect()
			if err != nil {
				continue
			}

			go func() {
				for {
					if _, err := conn.AcceptBi(); err != nil {
						return
					}
				}
			}()
		}
	}()

	t.Cleanup(func() { _ = endpoint.Close() })

	return endpoint, ticket.String()
}

// Publish must stop waiting when its context ends, even though a pooled send
// blocks in an ack read that has no deadline of its own.
func TestPublish_ReturnsWhenContextEndsOnSilentPeer(t *testing.T) {
	g := gomega.NewWithT(t)

	_, ticket := silentEndpoint(t)

	sender, err := New(WithLocalOnly(), WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer sender.Close()

	g.Expect(sender.Connect(ticket)).To(gomega.Succeed())
	waitForPeerCount(g, sender, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	op := irohengine.WriteOp{ID: "silent-1", Collection: "c", Kind: irohengine.OpMapSet, Key: "k", Value: "v"}

	errCh := make(chan error, 1)

	go func() { errCh <- sender.Publish(ctx, op) }()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Publish error = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Publish did not return after its context ended")
	}

	g.Eventually(func() int { return sender.PeerCount() }, 10*time.Second, 50*time.Millisecond).
		Should(gomega.Equal(0), "the wedged peer must be evicted")
}
