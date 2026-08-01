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
// Non-blocking: if a watcher's channel is full, the notification is dropped.
//
// When a replay recorder is registered for the collection, the value is
// recorded with a monotonic sequence number and sent as a watcherNotification
// wrapper so WatchWithSeq can recover the seq.
func (h *subscriberHub) notify(collection string, key any, value any) {
	h.mu.Lock()
	entries := h.watchers[collection]
	recorder := h.replays[collection]
	h.mu.Unlock()

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
			select {
			case entry.ch <- notif:
			default: // drop if consumer is slow
			}
		} else {
			select {
			case entry.ch <- value:
			default: // drop if consumer is slow
			}
		}
	}
}
