package command_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func matchGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	snaps.WithConfig(
		snaps.Dir(filepath.Join("testdata", "golden")),
		snaps.Filename(name),
	).MatchSnapshot(t, string(got))
}

// TestGolden_CommandMetadataWithActor pins the full command metadata JSON
// shape with every tracing option set, including WithActor. This is the shape
// command stores persist (metadataJSON column) and transports serialize —
// a tag change on Tracing.ActorID fails here before it can silently reshape
// stored commands.
func TestGolden_CommandMetadataWithActor(t *testing.T) {
	t.Parallel()

	actor := id.NewSystemActor("scheduler")

	cmd, err := command.New(
		"cancelOrder",
		idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		command.WithCorrelationID(idtest.ParseCorrelationID(t, "01HK1540X0841Y0A6BSX1VKR97")),
		command.WithCausationID(idtest.ParseCausationID(t, "01HK1540X0841Y0A6BSX1VKR98")),
		command.WithUserID(idtest.ParseUserID(t, "01HK1540X0841Y0A6BSX1VKR99")),
		command.WithActor(actor),
		command.WithRequestID(idtest.ParseRequestID(t, "01HK1540X0841Y0A6BSX1VKRA1")),
		command.WithCustomMetadata("tenant", "acme"),
		command.WithCustomMetadata("custom.trace", "abc123"),
	)
	if err != nil {
		t.Fatalf("new command: %v", err)
	}

	got, err := json.Marshal(
		cmd.Metadata(),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
		json.Deterministic(true),
	)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	matchGolden(t, "command-metadata-actor", got)

	// The persisted shape must reconstruct the same metadata — the load path
	// every command store takes when scanning the metadataJSON column.
	var loaded command.Metadata

	if err := json.Unmarshal(got, &loaded); err != nil {
		t.Fatalf("unmarshal metadata JSON: %v", err)
	}

	if !loaded.ActorID.Equal(actor) {
		t.Errorf("actor lost through metadata JSON round-trip: got %q, want %q",
			loaded.ActorID.PrefixedString(), actor.PrefixedString())
	}

	if loaded.CorrelationID != cmd.Metadata().CorrelationID {
		t.Errorf("correlation ID lost through round-trip: got %q", loaded.CorrelationID)
	}

	if loaded.Custom["tenant"] != "acme" || loaded.Custom["custom.trace"] != "abc123" {
		t.Errorf("custom metadata lost through round-trip: got %v", loaded.Custom)
	}
}
