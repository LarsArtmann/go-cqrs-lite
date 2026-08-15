package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend ---
//
// Models stream-keyed append-only logs via Dgraph nodes indexed by
// (collection, streamID) with a nanosecond-timestamp global sequence.
// This mirrors the LogBackend approach but adds per-stream partitioning.
//
// The global sequence (cqrs.stream_log_seq) uses UnixNano timestamps for
// JournalReadAll ordering. JournalReadFrom does NOT filter on raw seq values:
// its afterSeq parameter is a position-based resumption cursor (see the
// method comment), so it skips entries by index instead.

func (e *dgraphEngine) StreamAppend(
	ctx context.Context,
	col, sid string,
	values []any,
) error {
	if len(values) == 0 {
		return nil
	}

	nodes := make([]map[string]any, 0, len(values))

	for i, v := range values {
		seq := time.Now().UnixNano() + int64(i)
		nodes = append(nodes, map[string]any{
			"uid":                        fmt.Sprintf("_:sl%d", i),
			"cqrs.stream_log_collection": col,
			"cqrs.stream_log_stream":     sid,
			"cqrs.stream_log_seq":        seq,
			"cqrs.stream_log_value":      metaengine.EncodeStreamValue(v),
			"dgraph.type":                []string{"StreamLogEntry"},
		})
	}

	data, err := json.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("dgraphengine.StreamAppend: marshal: %w", err)
	}

	if _, err := e.client.NewTxn().Mutate(ctx, &api.Mutation{
		SetJson:   data,
		CommitNow: true,
	}); err != nil {
		return fmt.Errorf("dgraphengine.StreamAppend: %w", err)
	}

	return nil
}

func (e *dgraphEngine) StreamRead(
	ctx context.Context,
	col, sid string,
) ([]any, error) {
	q := `query sl($col: string, $sid: string) {
		entries(func: eq(cqrs.stream_log_collection, $col), orderasc: cqrs.stream_log_seq) @filter(eq(cqrs.stream_log_stream, $sid)) {
			cqrs.stream_log_seq
			cqrs.stream_log_value
		}
	}`

	resp, err := e.client.NewReadOnlyTxn().QueryWithVars(ctx, q,
		map[string]string{"$col": col, "$sid": sid})
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.StreamRead: %w", err)
	}

	var result struct {
		Entries []struct {
			Seq   int64  `json:"cqrs.stream_log_seq"`
			Value string `json:"cqrs.stream_log_value"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.StreamRead: unmarshal: %w", err)
	}

	out := make([]any, 0, len(result.Entries))
	for _, e := range result.Entries {
		out = append(out, metaengine.DecodeStreamValue(e.Value))
	}

	return out, nil
}

func (e *dgraphEngine) StreamVersion(
	ctx context.Context,
	col, sid string,
) (int64, error) {
	q := `query sl($col: string, $sid: string) {
		entries(func: eq(cqrs.stream_log_collection, $col)) @filter(eq(cqrs.stream_log_stream, $sid)) {
			count(uid)
		}
	}`

	resp, err := e.client.NewReadOnlyTxn().QueryWithVars(ctx, q,
		map[string]string{"$col": col, "$sid": sid})
	if err != nil {
		return 0, fmt.Errorf("dgraphengine.StreamVersion: %w", err)
	}

	var result struct {
		Entries []struct {
			Count int `json:"count"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return 0, fmt.Errorf("dgraphengine.StreamVersion: unmarshal: %w", err)
	}

	if len(result.Entries) == 0 {
		return 0, nil
	}

	return int64(result.Entries[0].Count), nil
}

func (e *dgraphEngine) JournalReadAll(
	ctx context.Context,
	col string,
) ([]any, error) {
	q := `query sl($col: string) {
		entries(func: eq(cqrs.stream_log_collection, $col), orderasc: cqrs.stream_log_seq) {
			cqrs.stream_log_value
		}
	}`

	resp, err := e.client.NewReadOnlyTxn().QueryWithVars(ctx, q,
		map[string]string{"$col": col})
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.JournalReadAll: %w", err)
	}

	var result struct {
		Entries []struct {
			Value string `json:"cqrs.stream_log_value"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.JournalReadAll: unmarshal: %w", err)
	}

	out := make([]any, 0, len(result.Entries))
	for _, e := range result.Entries {
		out = append(out, metaengine.DecodeStreamValue(e.Value))
	}

	return out, nil
}

func (e *dgraphEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	// afterSeq is a POSITION-based resumption cursor, not a raw seq value:
	// the system adapters (EventAdapter.lookupSeq / ReadFrom) derive it from
	// entry indexes over JournalReadAll, so it is only a valid seq on engines
	// with dense per-collection sequences. Dgraph seqs are sparse UnixNano
	// timestamps, so a gt(seq) filter would re-deliver the entire journal on
	// every resume. Skip afterSeq leading entries instead, matching the
	// dense-seq engines (memory, pebble, sqlite) where position == seq.
	if afterSeq < 0 {
		afterSeq = 0
	}

	firstClause := ""
	if limit > 0 {
		// Fetch afterSeq+limit entries server-side, then drop the first
		// afterSeq client-side: one round-trip, no reliance on DQL offset
		// semantics.
		firstClause = fmt.Sprintf(", first: %d", afterSeq+int64(limit))
	}

	// afterSeq and limit are non-negative int64s — %d emits only digits,
	// injection-safe by construction.
	q := fmt.Sprintf(`query sl($col: string) {
		entries(func: eq(cqrs.stream_log_collection, $col), orderasc: cqrs.stream_log_seq%s) {
			cqrs.stream_log_value
		}
	}`, firstClause)

	resp, err := e.client.NewReadOnlyTxn().QueryWithVars(ctx, q,
		map[string]string{"$col": col})
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.JournalReadFrom: %w", err)
	}

	var result struct {
		Entries []struct {
			Value string `json:"cqrs.stream_log_value"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.JournalReadFrom: unmarshal: %w", err)
	}

	if afterSeq >= int64(len(result.Entries)) {
		return []any{}, nil
	}

	entries := result.Entries[afterSeq:]

	out := make([]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, metaengine.DecodeStreamValue(e.Value))
	}

	return out, nil
}

// StreamAppendExpected appends values only if the stream's current version
// matches expectedVersion. Uses a Dgraph upsert with a conditional mutation:
// the query captures existing entries, and the mutation fires only if the
// count matches. If the condition fails, no UIDs are assigned, which we
// detect as a version conflict.
func (e *dgraphEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	if len(values) == 0 {
		return nil
	}

	req := &api.Request{CommitNow: true}
	req.Query = `query entry($col: string, $sid: string) {
		entry as var(func: eq(cqrs.stream_log_collection, $col)) @filter(eq(cqrs.stream_log_stream, $sid))
	}`
	req.Vars = map[string]string{"$col": col, "$sid": sid}

	nodes := make([]map[string]any, 0, len(values))

	for i, v := range values {
		seq := time.Now().UnixNano() + int64(i)
		nodes = append(nodes, map[string]any{
			"uid":                        fmt.Sprintf("_:sl%d", i),
			"cqrs.stream_log_collection": col,
			"cqrs.stream_log_stream":     sid,
			"cqrs.stream_log_seq":        seq,
			"cqrs.stream_log_value":      metaengine.EncodeStreamValue(v),
			"dgraph.type":                []string{"StreamLogEntry"},
		})
	}

	data, _ := json.Marshal(nodes)

	// expectedVersion is int64 — %d emits only digits, injection-safe.
	req.Mutations = []*api.Mutation{
		{SetJson: data, Cond: fmt.Sprintf("@if(eq(len(entry), %d))", expectedVersion)},
	}

	resp, err := e.client.NewTxn().Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgraphengine.StreamAppendExpected: %w", err)
	}

	if len(resp.GetUids()) == 0 {
		return metaengine.ErrVersionConflict
	}

	return nil
}
