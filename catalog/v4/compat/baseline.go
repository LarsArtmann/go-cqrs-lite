package compat

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// BaselineFormatVersion is the schema version of the baseline file itself.
// Bump it when the on-disk layout changes; [ParseBaseline] rejects anything
// else so an old file never silently compares as empty.
const BaselineFormatVersion = 1

// baselineFileMode is deliberately owner-only: gosec's G306 caps a newly
// created file at 0600, and a baseline is a build artifact, not shared state.
const baselineFileMode = 0o600

// Baseline is a committed snapshot of the payload shapes at the last release.
// It records shipped truth, not a live declaration: regenerate it with
// [NewBaseline] only after a deliberate, version-bumped change.
type Baseline struct {
	// FormatVersion identifies the file layout.
	FormatVersion int `json:"formatVersion"`

	// Catalog and CatalogVersion identify the snapshot for human readers;
	// neither participates in comparison.
	Catalog        string `json:"catalog,omitempty"`
	CatalogVersion string `json:"catalogVersion,omitempty"`

	// Messages is keyed by message ID.
	Messages map[string]BaselineMessage `json:"messages"`
}

// BaselineMessage is one message's shipped payload shape.
type BaselineMessage struct {
	Kind    string         `json:"kind"`
	ID      string         `json:"id"`
	Version string         `json:"version"`
	Schema  *schema.Schema `json:"schema,omitempty"`
}

// NewBaseline renders a catalog into a comparable, deterministic snapshot.
func NewBaseline(c *catalog.Catalog) *Baseline {
	baseline := &Baseline{
		FormatVersion: BaselineFormatVersion,
		Messages:      make(map[string]BaselineMessage),
	}

	if c == nil {
		return baseline
	}

	baseline.Catalog = string(c.Title)
	baseline.CatalogVersion = string(c.Version)

	for _, svc := range c.Services {
		for _, msg := range serviceMessages(svc) {
			id := string(catalog.Key(msg))
			baseline.Messages[id] = BaselineMessage{
				Kind:    msg.Kind.String(),
				ID:      id,
				Version: string(msg.Version),
				Schema:  msg.Schema.Clone(),
			}
		}
	}

	return baseline
}

// MarshalIndent renders the baseline as stable, indented JSON suitable for a
// committed artifact. Messages are keyed by ID, and encoding/json sorts map
// keys, so repeated renders are byte-identical.
func (b *Baseline) MarshalIndent() ([]byte, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("compat: marshal baseline: %w", err)
	}

	return append(data, '\n'), nil
}

// WriteFile renders the baseline and writes it to path.
func (b *Baseline) WriteFile(path string) error {
	data, err := b.MarshalIndent()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, baselineFileMode); err != nil {
		return fmt.Errorf("compat: write baseline %s: %w", path, err)
	}

	return nil
}

// ParseBaseline reads a baseline file, rejecting an unknown layout version and
// naming the file so the error is actionable.
func ParseBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("compat: read baseline %s: %w", path, err)
	}

	baseline := &Baseline{}
	if err := json.Unmarshal(data, baseline); err != nil {
		return nil, fmt.Errorf("compat: parse baseline %s: %w", path, err)
	}

	if baseline.FormatVersion != BaselineFormatVersion {
		return nil, fmt.Errorf(
			"%w: baseline %s has formatVersion %d, want %d: regenerate it with --update-baseline",
			ErrBaselineFormatVersion,
			path,
			baseline.FormatVersion,
			BaselineFormatVersion,
		)
	}

	return baseline, nil
}

// CompareBaseline classifies the current catalog against a committed baseline.
// The old catalog is reconstructed from the baseline; the version-must-move
// rule then works exactly as in [CompatCatalog].
func CompareBaseline(baseline *Baseline, c *catalog.Catalog) Report {
	var b reportBuilder

	current := indexMessages(c)

	for id, old := range baseline.Messages {
		key := catalog.MessageID(id)

		newMsg, ok := current[key]
		if !ok {
			b.add(Finding{
				Path:    id,
				Kind:    KindMessageRemoved,
				Class:   ClassNeedsVersionBump,
				Message: fmt.Sprintf("message %q was shipped but is no longer declared", id),
			})

			continue
		}

		diffMessage(
			&b,
			key,
			catalog.Version(old.Version),
			old.Schema,
			newMsg.Version,
			newMsg.Schema,
		)
	}

	for key := range current {
		if _, ok := baseline.Messages[string(key)]; ok {
			continue
		}

		b.add(Finding{
			Path:    string(key),
			Kind:    KindMessageAdded,
			Class:   ClassAdditive,
			Message: fmt.Sprintf("message %q is not in the baseline", key),
		})
	}

	return b.report()
}

// MessageIDs returns the baseline's message IDs in sorted order.
func (b *Baseline) MessageIDs() []string {
	ids := make([]string, 0, len(b.Messages))
	for id := range b.Messages {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return ids
}
