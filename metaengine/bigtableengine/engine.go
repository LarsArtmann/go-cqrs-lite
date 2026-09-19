// Package bigtableengine implements a metaengine.Engine on Google Cloud
// BigTable — the storage engine whose NATIVE versioned cells inspired the
// temporal capability stack (ADR-0141).
//
// Every map value is stored as a timestamped cell:
//
//	(row key "collection\x00key", family "cqrs", column "v", timestamp) → JSON bytes
//
// Writes never overwrite — they append a new version. As-of reads, history
// ranges, and tombstones are BigTable primitives (LatestNFilter,
// TimestampRangeFilterMicros, empty-value cells), not emulations. Version
// retention is the column-family GC policy (WithGCPolicy), set at table
// creation; BigTable is millisecond-granularity, so same-millisecond writes
// to one cell collapse last-writer-wins (documented in ADR-0141 §1).
//
// Tests run against the in-process bttest fake — no emulator binary needed.
package bigtableengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/bigtable"
	"google.golang.org/api/option"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

const (
	// family holds versioned map cells; column "v" carries the JSON value.
	family = "cqrs"
	column = "v"
	// counterFamily holds atomic counter cells (ReadModifyWrite increments).
	counterFamily = "cqrs_c"
)

// bigtableEngine implements metaengine.Engine on Google Cloud BigTable with
// native versioned cells (ADR-0141).
type bigtableEngine struct {
	tbl    *bigtable.Table
	admin  *bigtable.AdminClient
	client *bigtable.Client

	table     string
	gc        bigtable.GCPolicy
	ownsConns bool // constructed by New: Close closes client + admin
	cal       metaengine.Calibration
}

// Option tunes a Bigtable engine at construction time.
type Option func(*bigtableEngine)

// WithGCPolicy sets the column-family garbage-collection policy applied at
// table creation (e.g. bigtable.MaxVersionsPolicy(10),
// bigtable.MaxAgePolicy(7*24*time.Hour), or a UnionPolicy of both). This is
// the native retention knob — the engine never prunes versions client-side.
func WithGCPolicy(policy bigtable.GCPolicy) Option {
	return func(e *bigtableEngine) { e.gc = policy }
}

// New dials a BigTable instance, creates the table (with column families and
// the optional GC policy) when missing, and returns a versioned engine.
// clientOpts forward connection options — for the Bigtable emulator pass
// option.WithEndpoint(addr) and option.WithoutAuthentication().
func New(
	ctx context.Context,
	project, instance, table string,
	clientOpts ...option.ClientOption,
) (metaengine.Engine, error) {
	admin, err := bigtable.NewAdminClient(ctx, project, instance, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("bigtableengine: admin client: %w", err)
	}

	client, err := bigtable.NewClient(ctx, project, instance, clientOpts...)
	if err != nil {
		_ = admin.Close()

		return nil, fmt.Errorf("bigtableengine: data client: %w", err)
	}

	eng, err := newWithClients(ctx, admin, client, table)
	if err != nil {
		_ = admin.Close()
		_ = client.Close()

		return nil, err
	}

	eng.(*bigtableEngine).ownsConns = true //nolint:forcetypeassert // newWithClients returns *bigtableEngine

	return eng, nil
}

// NewWithClients builds an engine over injected clients (tests, dependency
// injection). The engine does NOT close the clients on Close.
func NewWithClients(
	admin *bigtable.AdminClient,
	client *bigtable.Client,
	table string,
	opts ...Option,
) (metaengine.Engine, error) {
	return newWithClients(context.Background(), admin, client, table, opts...)
}

func newWithClients(
	ctx context.Context,
	admin *bigtable.AdminClient,
	client *bigtable.Client,
	table string,
	opts ...Option,
) (metaengine.Engine, error) {
	e := &bigtableEngine{
		admin:  admin,
		client: client,
		table:  table,
		tbl:    client.Open(table),
	}

	for _, opt := range opts {
		opt(e)
	}

	if err := e.ensureTable(ctx); err != nil {
		return nil, err
	}

	return e, nil
}

// ensureTable creates the table and its column families when missing.
func (e *bigtableEngine) ensureTable(ctx context.Context) error {
	conf := &bigtable.TableConf{
		TableID: e.table,
		ColumnFamilies: map[string]bigtable.Family{
			family:        {GCPolicy: e.gc},
			counterFamily: {GCPolicy: e.gc},
		},
	}

	if err := e.admin.CreateTableFromConf(ctx, conf); err != nil {
		info, infoErr := e.admin.TableInfo(ctx, e.table)
		if infoErr != nil || !hasFamilies(info, family, counterFamily) {
			return fmt.Errorf("bigtableengine: create table: %w", err)
		}
	}

	return nil
}

func hasFamilies(info *bigtable.TableInfo, want ...string) bool {
	if info == nil {
		return false
	}

	have := make(map[string]bool, len(info.FamilyInfos))
	for _, f := range info.FamilyInfos {
		have[f.Name] = true
	}

	for _, w := range want {
		if !have[w] {
			return false
		}
	}

	return true
}

// Prior costs (compile-time; replaced by live probes once ProbeEngine runs —
// see METAENGINE-LIVE-LATENCY-MODEL.md). BigTable same-region RTT is a few
// milliseconds; NsPerOp prices one round trip conservatively.
const (
	BigtableNsPerOp    = 2_000_000.0
	BigtableNetworkRTT = 3 * time.Millisecond
)

func (e *bigtableEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:            "bigtable",
		NsPerOp:         BigtableNsPerOp,
		RequiresNetwork: true,
		NetworkRTT:      BigtableNetworkRTT,
		Persistence:     metaengine.PersistencePersistent, // remote replicated service
		ReadCosts: metaengine.ReadCosts{
			// One RPC round trip for a point lookup (uncalibrated prior).
			NsPerPointLookup:  2_000_000,
			NsPerAggregate:    2_000_000, // CounterGet = one prefix scan RPC
			NsPerScan:         900,       // per-row within one streamed ReadRows
			NsPerFilteredScan: 900,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:     metaengine.ComplexityO1, // key-addressed LSM lookup
			metaengine.ADTCounter: metaengine.ComplexityO1, // native ReadModifyWrite
		},
		// ADR-0142 explicit capability refusal (never silence): the shared
		// Map runtimes (MapDueClaimer/MapDedupStore) need atomic
		// arbitrary-value read-modify-write (MapUpdater) plus collection
		// scans (ScanBackend) — this engine implements neither today.
		// Native paths exist (CheckAndMutate CAS + ReadRows prefix scan);
		// revisit when they are wired.
		RefusedADTs: map[metaengine.ADT]string{
			metaengine.ADTDueClaim: "no atomic arbitrary-value RMW (MapUpdate) or collection scan yet — claims need CheckAndMutate CAS + ReadRows",
			metaengine.ADTDedup:    "no atomic arbitrary-value RMW (MapUpdate) or collection scan yet — dedup needs CAS-with-TTL",
		},
	}
	e.cal.ApplyCalibration(&p)

	return p
}

func (e *bigtableEngine) Close() error {
	if !e.ownsConns {
		return nil
	}

	if err := e.client.Close(); err != nil {
		_ = e.admin.Close()

		return fmt.Errorf("bigtableengine: close client: %w", err)
	}

	return e.admin.Close() //nolint:wrapcheck // driver-provided close error
}

// HealthCheck implements [metaengine.HealthChecker]: a one-row read proves
// the data path answers.
func (e *bigtableEngine) HealthCheck(ctx context.Context) error {
	_, err := e.tbl.ReadRow(
		ctx,
		"__health__",
		bigtable.RowFilter(bigtable.CellsPerRowLimitFilter(1)),
	)
	if err != nil {
		return fmt.Errorf("bigtableengine: health read: %w", err)
	}

	return nil
}

// ResetEngine implements [metaengine.EngineResetter]: deletes every row in
// the table (a full wipe — BigTable has no cheaper table-wide truncate).
func (e *bigtableEngine) ResetEngine(ctx context.Context) error {
	var muts []*bigtable.Mutation
	var keys []string

	err := e.tbl.ReadRows(ctx, bigtable.InfiniteRange(""), func(row bigtable.Row) bool {
		m := bigtable.NewMutation()
		m.DeleteRow()
		muts = append(muts, m)
		keys = append(keys, row.Key())

		return true
	}, bigtable.RowFilter(bigtable.StripValueFilter()))
	if err != nil {
		return fmt.Errorf("bigtableengine: reset scan: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := e.applyAll(ctx, keys, muts); err != nil {
		return fmt.Errorf("bigtableengine: reset delete: %w", err)
	}

	return nil
}

func (e *bigtableEngine) applyAll(
	ctx context.Context,
	keys []string,
	muts []*bigtable.Mutation,
) error {
	errs, err := e.tbl.ApplyBulk(ctx, keys, muts)
	if err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	for _, rowErr := range errs {
		if rowErr != nil {
			return rowErr //nolint:wrapcheck // first row error surfaces
		}
	}

	return nil
}

// rowKey builds "collection\x00key" — null separator keeps collection rows
// contiguous for prefix operations.
func rowKey(col string, key any) string {
	return col + "\x00" + encodeKey(key)
}

func encodeKey(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	default:
		return fmt.Sprint(k)
	}
}

// encodeJSON serializes a cell value; the decode side is
// metaengine.DecodeStreamValue, so plain strings round-trip as strings and
// composite values as JSON.
func encodeJSON(v any) []byte {
	//art-dupl:accept 6-line marshal-with-fallback mirrors keycodec's encoder; dep isolation beats cross-module coupling for this size
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf("%v", v))
	}

	return b
}

// SetCalibration implements [metaengine.Calibratable].
func (e *bigtableEngine) SetCalibration(costs metaengine.CalibrationCosts) {
	e.cal.SetCalibration(costs)
}
