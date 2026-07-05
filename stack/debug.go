package stack

import "strings"

// DebugStructured returns a map of capability field names to their configured
// status (true = set, false = nil). This is the programmatic equivalent of
// [Bundle.Debug] for health endpoints and automated checks:
//
//	status := bundle.DebugStructured()
//	if !status["EventSink"] { return errors.New("event store not configured") }
func (b *Bundle) DebugStructured() map[string]bool {
	return map[string]bool{
		"EventSink":              b.EventSink != nil,
		"EventSource":            b.EventSource != nil,
		"Journal":                b.Journal != nil,
		"SeekableJournal":        b.SeekableJournal != nil,
		"BackwardsSource":        b.BackwardsSource != nil,
		"Publisher":              b.Publisher != nil,
		"Subscriber":             b.Subscriber != nil,
		"CommandSink":            b.CommandSink != nil,
		"CommandSource":          b.CommandSource != nil,
		"CommandJournal":         b.CommandJournal != nil,
		"SeekableCommandJournal": b.SeekableCommandJournal != nil,
		"QuerySink":              b.QuerySink != nil,
		"QuerySource":            b.QuerySource != nil,
		"QueryJournal":           b.QueryJournal != nil,
		"SeekableQueryJournal":   b.SeekableQueryJournal != nil,
		"SnapshotStore":          b.SnapshotStore != nil,
		"CheckpointStore":        b.CheckpointStore != nil,
		"ReadModels":             b.ReadModels != nil,
	}
}

// Debug returns a human-readable summary of the Bundle's configured
// capabilities. Each line reports whether a capability field is set (✓) or
// nil (✗). Use this to verify wiring after constructing a Bundle from a preset
// or manual assembly.
//
// Example output:
//
//	Bundle capabilities:
//	  EventSink:         ✓
//	  EventSource:       ✓
//	  Journal:           ✓
//	  SeekableJournal:   ✓
//	  Publisher:         ✓
//	  Subscriber:        ✓
//	  CommandSink:       ✓
//	  CommandSource:     ✓
//	  QuerySink:         ✗
//	  QuerySource:       ✗
//	  SnapshotStore:     ✓
//	  CheckpointStore:   ✓
//	  ReadModels:        ✓
func (b *Bundle) Debug() string {
	var builder strings.Builder

	builder.WriteString("Bundle capabilities:\n")

	type field struct {
		name string
		val  any
	}

	fields := []field{
		{"EventSink", b.EventSink},
		{"EventSource", b.EventSource},
		{"Journal", b.Journal},
		{"SeekableJournal", b.SeekableJournal},
		{"BackwardsSource", b.BackwardsSource},
		{"Publisher", b.Publisher},
		{"Subscriber", b.Subscriber},
		{"CommandSink", b.CommandSink},
		{"CommandSource", b.CommandSource},
		{"CommandJournal", b.CommandJournal},
		{"SeekableCommandJournal", b.SeekableCommandJournal},
		{"QuerySink", b.QuerySink},
		{"QuerySource", b.QuerySource},
		{"QueryJournal", b.QueryJournal},
		{"SeekableQueryJournal", b.SeekableQueryJournal},
		{"SnapshotStore", b.SnapshotStore},
		{"CheckpointStore", b.CheckpointStore},
		{"ReadModels", b.ReadModels},
	}

	for _, f := range fields {
		mark := "✗"
		if f.val != nil {
			mark = "✓"
		}

		builder.WriteString("  ")
		builder.WriteString(padName(f.name))
		builder.WriteString(": ")
		builder.WriteString(mark)
		builder.WriteByte('\n')
	}

	return builder.String()
}

func padName(s string) string {
	const width = 20

	if len(s) >= width {
		return s
	}

	return s + strings.Repeat(" ", width-len(s))
}
