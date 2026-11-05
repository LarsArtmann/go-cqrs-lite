package metaengine

import "sync"

// subscriberHub manages watcher subscriptions and replay recorders per collection.
// Extracted from Store to isolate the concurrency concerns of watcher management.
type subscriberHub struct {
	mu       sync.Mutex
	watchers map[string][]*watcherEntry // collection → watcher entries
	replays  map[string]replayRecorder  // collection → replay recorder (nil = no replay)
}

func newSubscriberHub() *subscriberHub {
	return &subscriberHub{
		watchers: make(map[string][]*watcherEntry),
		replays:  make(map[string]replayRecorder),
	}
}

func (h *subscriberHub) registerWatcher(collection string, entry *watcherEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.watchers[collection] = append(h.watchers[collection], entry)
}

func (h *subscriberHub) registerReplay(collection string, recorder replayRecorder) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.replays[collection] = recorder
}

func (h *subscriberHub) unregisterReplay(collection string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.replays, collection)
}

// notify sends a value update to all watchers of a collection that
// match the given key. A watcher with a nil key receives all notifications;
// a watcher with a non-nil key receives only notifications for that key.
// Delivery is latest-wins: if a watcher's channel is full, the oldest
// buffered notification is evicted so the newest state always gets through.
//
// The hub mutex is held for the entire iteration (including channel sends)
// so that Watcher.Close — which closes channels under the same lock via
// closeEntries — cannot race with a send. Without this, close(ch) and ch<-v
// run under different locks and the send panics on a closed channel.
//
// When a replay recorder is registered for the collection, the value is
// recorded with a monotonic sequence number and sent as a watcherNotification
// wrapper so WatchWithSeq can recover the seq.
func (h *subscriberHub) notify(collection string, key any, value any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries := h.watchers[collection]
	recorder := h.replays[collection]

	var notif watcherNotification
	if recorder != nil {
		notif = watcherNotification{seq: recorder.recordValue(value), value: value}
	}

	for _, entry := range entries {
		if entry.closed {
			continue
		}

		if entry.key != nil && !keysMatch(entry.key, key) {
			continue
		}

		if recorder != nil {
			sendLatest(entry, notif)
		} else {
			sendLatest(entry, value)
		}
	}
}

// sendLatest delivers val to an entry channel with latest-wins semantics:
// when the buffer is full, the OLDEST buffered notification is evicted to
// make room for the newest. Watcher channels carry projection state, where
// the newest value subsumes every older one, and the SSE pipeline downstream
// (forwardWithDropOld) applies the same rule — dropping the newest value
// instead would leave a slow consumer stuck on stale state forever.
func sendLatest(entry *watcherEntry, val any) {
	select {
	case entry.ch <- val:
		return
	default:
	}

	select {
	case <-entry.ch: // evict the stale buffered value
	default:
	}

	select {
	case entry.ch <- val:
	default:
	}
}

// closeEntries closes all channels for the given entries under the hub mutex.
// This serializes channel closes against notify's non-blocking sends, which
// also happen under the hub mutex. Setting entry.closed before close(ch)
// ensures that if notify runs next it skips the entry; if Close runs first
// the channel is already closed and notify's closed-check prevents sends.
func (h *subscriberHub) closeEntries(entries []*watcherEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, entry := range entries {
		entry.closed = true
		close(entry.ch)
	}
}
