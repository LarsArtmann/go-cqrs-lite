#!/usr/bin/env python3
"""Fix the botched AggregateRef migration across the entire codebase.

The auto-migration changed interface signatures to use AggregateRef but:
1. Left method bodies referencing aggregateType/aggregateID as bare identifiers
2. Left callers passing separate (AggregateType, AggregateID) args
3. Left callback function types with old signatures
"""

import re
import subprocess
import sys

def read_file(path):
    with open(path) as f:
        return f.read()

def write_file(path, content):
    with open(path, 'w') as f:
        f.write(content)

# ============================================================
# 1. memory/store.go - use ref.StreamKey() instead of separate args
# ============================================================
def fix_memory_store():
    path = "memory/store.go"
    content = read_file(path)
    
    # Save method: replace event.StreamKey(aggregateType, aggregateID) with ref.StreamKey()
    content = content.replace('aggregateType, aggregateID := ref.Type, ref.ID\n\terr := s.CheckClosed', 'err := s.CheckClosed')
    # Replace event.StreamKey(aggregateType, aggregateID) with ref.StreamKey()
    content = content.replace('event.StreamKey(aggregateType, aggregateID)', 'ref.StreamKey()')
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 2. memory/store_load.go - use ref.StreamKey() 
# ============================================================
def fix_memory_store_load():
    path = "memory/store_load.go"
    content = read_file(path)
    
    # getEvents: replace stream key usage
    content = content.replace(
        'aggregateType, aggregateID := ref.Type, ref.ID\n\terr := s.CheckClosed',
        'err := s.CheckClosed'
    )
    content = content.replace(
        '\tkey := event.StreamKey(aggregateType, aggregateID)\n',
        '\tkey := ref.StreamKey()\n'
    )
    # Fix format strings that use aggregateType, aggregateID
    content = content.replace(
        '\t\t"aggregate %s/%s: %w",\n\t\t\taggregateType,\n\t\t\taggregateID,',
        '\t\t"aggregate %s: %w",\n\t\t\tref,'
    )
    # Remove unused id import if present
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 3. memory/snapshot.go - use ref.StreamKey()
# ============================================================
def fix_memory_snapshot():
    path = "memory/snapshot.go"
    content = read_file(path)
    
    # Replace aggregateType, aggregateID := ref.Type, ref.ID + event.StreamKey
    content = content.replace(
        'aggregateType, aggregateID := ref.Type, ref.ID\n\terr := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_load_failed",\n\t\t\t"snapshot store load",\n\t\t)',
        'err := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_load_failed",\n\t\t\t"snapshot store load",\n\t\t)'
    )
    content = content.replace(
        'aggregateType, aggregateID := ref.Type, ref.ID\n\terr := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_load_at_version_failed",\n\t\t\t"snapshot store load at version",\n\t\t)',
        'err := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_load_at_version_failed",\n\t\t\t"snapshot store load at version",\n\t\t)'
    )
    content = content.replace(
        'aggregateType, aggregateID := ref.Type, ref.ID\n\terr := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_delete_failed",\n\t\t\t"snapshot store delete",\n\t\t)',
        'err := s.CheckClosed(event.ErrSnapshotStoreClosed)\n\tif err != nil {\n\t\treturn event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"memory.snapshot_delete_failed",\n\t\t\t"snapshot store delete",\n\t\t)'
    )
    content = content.replace('event.StreamKey(aggregateType, aggregateID)', 'ref.StreamKey()')
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 4. memory/stream.go - already uses ref, just fix getEvents call
# ============================================================
def fix_memory_stream():
    path = "memory/stream.go"
    content = read_file(path)
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 5. testhelpers/fake_store.go - fix callback types and callers
# ============================================================
def fix_fake_store():
    path = "testhelpers/fake_store.go"
    content = read_file(path)
    
    # Fix callback field types to use AggregateRef
    content = content.replace(
        'loadFn            func(aggregateType event.AggregateType, aggregateID id.AggregateID) ([]event.Event, error)',
        'loadFn            func(ref event.AggregateRef) ([]event.Event, error)'
    )
    content = content.replace(
        'loadFromVersionFn func(aggregateType event.AggregateType, aggregateID id.AggregateID, version event.Version) ([]event.Event, error)',
        'loadFromVersionFn func(ref event.AggregateRef, version event.Version) ([]event.Event, error)'
    )
    content = content.replace(
        'loadToVersionFn   func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxVersion event.Version) ([]event.Event, error)',
        'loadToVersionFn   func(ref event.AggregateRef, maxVersion event.Version) ([]event.Event, error)'
    )
    content = content.replace(
        'loadToTimestampFn func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxTime time.Time) ([]event.Event, error)',
        'loadToTimestampFn func(ref event.AggregateRef, maxTime time.Time) ([]event.Event, error)'
    )
    content = content.replace(
        'appendBatchFn     func(aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event) error',
        'appendBatchFn     func(ref event.AggregateRef, events []event.Event) error'
    )
    
    # Fix VersionQueryFn return type
    content = content.replace(
        'func VersionQueryFn(\n\tcalled *bool,\n) func(event.AggregateType, id.AggregateID, event.Version) ([]event.Event, error) {\n\treturn func(_ event.AggregateType, _ id.AggregateID, _ event.Version) ([]event.Event, error) {',
        'func VersionQueryFn(\n\tcalled *bool,\n) func(event.AggregateRef, event.Version) ([]event.Event, error) {\n\treturn func(_ event.AggregateRef, _ event.Version) ([]event.Event, error) {'
    )
    
    # Fix Save: the saveFn callback is event.SaveFunc which takes AggregateRef
    # The body extracts aggregateType, aggregateID and calls fn(ctx, aggregateType, aggregateID, ...)
    # Need to call fn(ctx, ref, ...) instead
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.saveFn); fn != nil {\n\t\treturn fn(ctx, aggregateType, aggregateID, events, expectedVersion)\n\t}',
        '\tif fn := getOverride(s, &s.saveFn); fn != nil {\n\t\treturn fn(ctx, ref, events, expectedVersion)\n\t}'
    )
    
    # Fix AppendBatch: similar pattern
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.appendBatchFn); fn != nil {\n\t\treturn fn(aggregateType, aggregateID, events)\n\t}',
        '\tif fn := getOverride(s, &s.appendBatchFn); fn != nil {\n\t\treturn fn(ref, events)\n\t}'
    )
    
    # Fix Load
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.loadFn); fn != nil {\n\t\treturn fn(aggregateType, aggregateID)\n\t}',
        '\tif fn := getOverride(s, &s.loadFn); fn != nil {\n\t\treturn fn(ref)\n\t}'
    )
    
    # Fix loadEventsHelper
    content = content.replace(
        'func (s *FakeStore) loadEventsHelper(\n\tref event.AggregateRef,\n) []event.Event {\n\taggregateType, aggregateID := ref.Type, ref.ID\n\ts.mu.RLock()\n\tdefer s.mu.RUnlock()\n\n\tkey := event.StreamKey(aggregateType, aggregateID)',
        'func (s *FakeStore) loadEventsHelper(\n\tref event.AggregateRef,\n) []event.Event {\n\ts.mu.RLock()\n\tdefer s.mu.RUnlock()\n\n\tkey := ref.StreamKey()'
    )
    
    # Fix LoadFromVersion
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.loadFromVersionFn); fn != nil {\n\t\treturn fn(aggregateType, aggregateID, version)\n\t}\n\n\tresult := event.SliceFromVersion(s.loadEventsHelper(aggregateType, aggregateID), version)',
        '\tif fn := getOverride(s, &s.loadFromVersionFn); fn != nil {\n\t\treturn fn(ref, version)\n\t}\n\n\tresult := event.SliceFromVersion(s.loadEventsHelper(ref), version)'
    )
    
    # Fix LoadToVersion
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.loadToVersionFn); fn != nil {\n\t\treturn fn(aggregateType, aggregateID, maxVersion)\n\t}\n\n\treturn event.SliceToVersion(s.loadEventsHelper(aggregateType, aggregateID), maxVersion), nil',
        '\tif fn := getOverride(s, &s.loadToVersionFn); fn != nil {\n\t\treturn fn(ref, maxVersion)\n\t}\n\n\treturn event.SliceToVersion(s.loadEventsHelper(ref), maxVersion), nil'
    )
    
    # Fix LoadToTimestamp
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif fn := getOverride(s, &s.loadToTimestampFn); fn != nil {\n\t\treturn fn(aggregateType, aggregateID, maxTime)\n\t}\n\n\treturn event.FilterByTimestamp(s.loadEventsHelper(aggregateType, aggregateID), maxTime), nil',
        '\tif fn := getOverride(s, &s.loadToTimestampFn); fn != nil {\n\t\treturn fn(ref, maxTime)\n\t}\n\n\treturn event.FilterByTimestamp(s.loadEventsHelper(ref), maxTime), nil'
    )
    
    # Fix remaining StreamKey calls in Save and AppendBatch
    content = content.replace(
        '\tkey := event.StreamKey(aggregateType, aggregateID)\n\ts.events[key] = append(s.events[key], events...)\n\n\treturn nil\n}\n\n// AppendBatch',
        '\tkey := ref.StreamKey()\n\ts.events[key] = append(s.events[key], events...)\n\n\treturn nil\n}\n\n// AppendBatch'
    )
    content = content.replace(
        '\tkey := event.StreamKey(aggregateType, aggregateID)\n\ts.events[key] = append(s.events[key], events...)\n\n\treturn nil\n}\n\n// Load',
        '\tkey := ref.StreamKey()\n\ts.events[key] = append(s.events[key], events...)\n\n\treturn nil\n}\n\n// Load'
    )
    content = content.replace(
        '\tkey := event.StreamKey(aggregateType, aggregateID)\n\n\treturn append([]event.Event{}, s.events[key]...), nil\n}\n\n// loadEventsHelper',
        '\tkey := ref.StreamKey()\n\n\treturn append([]event.Event{}, s.events[key]...), nil\n}\n\n// loadEventsHelper'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 6. testhelpers/fake_snapshot.go - fix interface methods
# ============================================================
def fix_fake_snapshot():
    path = "testhelpers/fake_snapshot.go"
    content = read_file(path)
    
    # Fix Load signature
    content = content.replace(
        'func (s *FakeSnapshotStore) Load(\n\t_ context.Context,\n\t_ event.AggregateType,\n\t_ id.AggregateID,\n) (*event.Snapshot, error) {',
        'func (s *FakeSnapshotStore) Load(\n\t_ context.Context,\n\t_ event.AggregateRef,\n) (*event.Snapshot, error) {'
    )
    
    # Fix LoadAtVersion signature
    content = content.replace(
        'func (s *FakeSnapshotStore) LoadAtVersion(\n\t_ context.Context,\n\t_ event.AggregateType,\n\t_ id.AggregateID,\n\t_ event.Version,\n) (*event.Snapshot, error) {',
        'func (s *FakeSnapshotStore) LoadAtVersion(\n\t_ context.Context,\n\t_ event.AggregateRef,\n\t_ event.Version,\n) (*event.Snapshot, error) {'
    )
    
    # Fix Delete signature
    content = content.replace(
        'func (s *FakeSnapshotStore) Delete(\n\t_ context.Context,\n\t_ event.AggregateType,\n\t_ id.AggregateID,\n) error {',
        'func (s *FakeSnapshotStore) Delete(\n\t_ context.Context,\n\t_ event.AggregateRef,\n) error {'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 7. core/decider/decider.go - fix Execute method body
# ============================================================
def fix_decider():
    path = "core/decider/decider.go"
    content = read_file(path)
    
    # Build the ref early
    content = content.replace(
        '\tr.applyEnricher(ctx, newEvents)\n\n\tif ts, ok := r.store.(event.TransactionalSink); ok && r.outbox != nil {\n\t\terr = ts.SaveWithOutbox(ctx, aggregateType, aggregateID, newEvents, currentVersion)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(aggregateType, aggregateID, "%w: %w", ErrSaveFailed, err)\n\t\t}\n\t} else {\n\t\terr = r.store.Save(ctx, aggregateType, aggregateID, newEvents, currentVersion)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(aggregateType, aggregateID, "%w: %w", ErrSaveFailed, err)\n\t\t}\n\n\t\terr = event.PublishChanges(ctx, r.publisher, r.outbox, newEvents)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(aggregateType, aggregateID, "publish events: %w", err)\n\t\t}\n\t}\n\n\tnewVersion := currentVersion.Add(len(newEvents))\n\n\tr.saveSnapshotAfterEvents(ctx, aggregateType, aggregateID, newVersion, state, newEvents)',
        '\tref := event.NewAggregateRef(aggregateType, aggregateID)\n\tr.applyEnricher(ctx, newEvents)\n\n\tif ts, ok := r.store.(event.TransactionalSink); ok && r.outbox != nil {\n\t\terr = ts.SaveWithOutbox(ctx, ref, newEvents, currentVersion)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(ref, "%w: %w", ErrSaveFailed, err)\n\t\t}\n\t} else {\n\t\terr = r.store.Save(ctx, ref, newEvents, currentVersion)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(ref, "%w: %w", ErrSaveFailed, err)\n\t\t}\n\n\t\terr = event.PublishChanges(ctx, r.publisher, r.outbox, newEvents)\n\t\tif err != nil {\n\t\t\tcqrsotel.RecordError(span, err)\n\n\t\t\treturn opError(ref, "publish events: %w", err)\n\t\t}\n\t}\n\n\tnewVersion := currentVersion.Add(len(newEvents))\n\n\tr.saveSnapshotAfterEvents(ctx, ref, newVersion, state, newEvents)'
    )
    
    # Fix saveSnapshotAfterEvents body - it already takes ref, just fix internal refs
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif !r.shouldSnapshot(aggregateType, newVersion) {',
        '\tif !r.shouldSnapshot(ref.Type, newVersion) {'
    )
    content = content.replace(
        '\t\t_ = opError(aggregateType, aggregateID, "fold event %s for snapshot: %w", evt.Type(), foldErr)',
        '\t\t_ = opError(ref, "fold event %s for snapshot: %w", evt.Type(), foldErr)'
    )
    content = content.replace(
        '\t\t_ = opError(aggregateType, aggregateID, "encode snapshot: %w", encErr)',
        '\t\t_ = opError(ref, "encode snapshot: %w", encErr)'
    )
    content = content.replace(
        '\t_ = event.SaveSnapshot(ctx, r.snapshotStore, aggregateType, aggregateID, newVersion, encoded)',
        '\t_ = event.SaveSnapshot(ctx, r.snapshotStore, ref.Type, ref.ID, newVersion, encoded)'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 8. core/decider/load.go - fix all methods
# ============================================================
def fix_decider_load():
    path = "core/decider/load.go"
    content = read_file(path)
    
    # Fix loadFromStore
    content = content.replace(
        'func (r *Repository[State]) loadFromStore(\n\tctx context.Context,\n\taggregateID id.AggregateID,\n\taggregateType event.AggregateType,\n) (State, event.Version, error) {\n\treturn r.loadByEvents(\n\t\tfunc() ([]event.Event, error) { return r.store.Load(ctx, aggregateType, aggregateID) },\n\t\taggregateType,\n\t\taggregateID,\n\t)',
        'func (r *Repository[State]) loadFromStore(\n\tctx context.Context,\n\taggregateID id.AggregateID,\n\taggregateType event.AggregateType,\n) (State, event.Version, error) {\n\tref := event.NewAggregateRef(aggregateType, aggregateID)\n\treturn r.loadByEvents(\n\t\tfunc() ([]event.Event, error) { return r.store.Load(ctx, ref) },\n\t\tref,\n\t)'
    )
    
    # Fix foldEvents - uses aggregateType, aggregateID in body but takes ref
    content = content.replace(
        '\tvar err error\n\n\tfor _, evt := range events {\n\t\tstate, err = r.decider.Fold(state, evt)\n\t\tif err != nil {\n\t\t\tvar zero State\n\n\t\t\treturn zero, opError(\n\t\t\t\taggregateType,\n\t\t\t\taggregateID,\n\t\t\t\t"%w (event %s): %w",\n\t\t\t\tErrFoldFailed,\n\t\t\t\tevt.Type(),\n\t\t\t\terr,\n\t\t\t)',
        '\tvar err error\n\n\tfor _, evt := range events {\n\t\tstate, err = r.decider.Fold(state, evt)\n\t\tif err != nil {\n\t\t\tvar zero State\n\n\t\t\treturn zero, opError(\n\t\t\t\tref,\n\t\t\t\t"%w (event %s): %w",\n\t\t\t\tErrFoldFailed,\n\t\t\t\tevt.Type(),\n\t\t\t\terr,\n\t\t\t)'
    )
    
    # Fix opError body
    content = content.replace(
        '\tprefix := aggregateType.String() + " " + aggregateID.String() + ": "',
        '\tprefix := ref.String() + ": "'
    )
    
    # Fix LoadAtVersion
    content = content.replace(
        '\tstate, ver, err := r.loadByEvents(\n\t\tfunc() ([]event.Event, error) {\n\t\t\treturn r.store.LoadToVersion(ctx, aggregateType, aggregateID, maxVersion)\n\t\t},\n\t\taggregateType, aggregateID,\n\t)',
        '\tref := event.NewAggregateRef(aggregateType, aggregateID)\n\n\tstate, ver, err := r.loadByEvents(\n\t\tfunc() ([]event.Event, error) {\n\t\t\treturn r.store.LoadToVersion(ctx, ref, maxVersion)\n\t\t},\n\t\tref,\n\t)'
    )
    
    # Fix LoadAtTime
    content = content.replace(
        '\tstate, ver, err := r.loadByEvents(\n\t\tfunc() ([]event.Event, error) {\n\t\t\treturn r.store.LoadToTimestamp(ctx, aggregateType, aggregateID, maxTime)\n\t\t},\n\t\taggregateType, aggregateID,\n\t)',
        '\tref := event.NewAggregateRef(aggregateType, aggregateID)\n\n\tstate, ver, err := r.loadByEvents(\n\t\tfunc() ([]event.Event, error) {\n\t\t\treturn r.store.LoadToTimestamp(ctx, ref, maxTime)\n\t\t},\n\t\tref,\n\t)'
    )
    
    # Fix loadByEvents body - uses aggregateType, aggregateID
    content = content.replace(
        '\tevents, err := loadFn()\n\tif err != nil {\n\t\tif errors.Is(err, event.ErrAggregateNotFound) {\n\t\t\treturn r.decider.Initial, 0, nil\n\t\t}\n\n\t\tvar zero State\n\n\t\treturn zero, 0, opError(aggregateType, aggregateID, "%w: %w", ErrLoadFailed, err)\n\t}\n\n\tstate, err := r.foldEvents(r.decider.Initial, events, aggregateType, aggregateID)',
        '\tevents, err := loadFn()\n\tif err != nil {\n\t\tif errors.Is(err, event.ErrAggregateNotFound) {\n\t\t\treturn r.decider.Initial, 0, nil\n\t\t}\n\n\t\tvar zero State\n\n\t\treturn zero, 0, opError(ref, "%w: %w", ErrLoadFailed, err)\n\t}\n\n\tstate, err := r.foldEvents(r.decider.Initial, events, ref)'
    )
    
    # Fix loadFromSnapshot
    content = content.replace(
        'func (r *Repository[State]) loadFromSnapshot(\n\tctx context.Context,\n\taggregateID id.AggregateID,\n\taggregateType event.AggregateType,\n) (State, event.Version, error) {\n\tsnap, err := r.snapshotStore.Load(ctx, aggregateType, aggregateID)',
        'func (r *Repository[State]) loadFromSnapshot(\n\tctx context.Context,\n\taggregateID id.AggregateID,\n\taggregateType event.AggregateType,\n) (State, event.Version, error) {\n\tref := event.NewAggregateRef(aggregateType, aggregateID)\n\n\tsnap, err := r.snapshotStore.Load(ctx, ref)'
    )
    content = content.replace(
        '\t\t\treturn zero, 0, opError(aggregateType, aggregateID, "load snapshot: %w", err)\n\t\t}\n\n\t\treturn r.loadFromStore(ctx, aggregateID, aggregateType)\n\t}\n\n\tif snap == nil {\n\t\treturn r.loadFromStore(ctx, aggregateID, aggregateType)\n\t}',
        '\t\t\treturn zero, 0, opError(ref, "load snapshot: %w", err)\n\t\t}\n\n\t\treturn r.loadFromStore(ctx, aggregateID, aggregateType)\n\t}\n\n\tif snap == nil {\n\t\treturn r.loadFromStore(ctx, aggregateID, aggregateType)\n\t}'
    )
    content = content.replace(
        '\t\treturn zero, 0, opError(aggregateType, aggregateID, "decode snapshot: %w", err)',
        '\t\treturn zero, 0, opError(ref, "decode snapshot: %w", err)'
    )
    content = content.replace(
        '\tevents, err := r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snap.Version)\n\tif err != nil {\n\t\tvar zero State\n\n\t\treturn zero, 0, opError(aggregateType, aggregateID, "%w: %w", ErrLoadFailed, err)\n\t}\n\n\tstate, err = r.foldEvents(state, events, aggregateType, aggregateID)',
        '\tevents, err := r.store.LoadFromVersion(ctx, ref, snap.Version)\n\tif err != nil {\n\t\tvar zero State\n\n\t\treturn zero, 0, opError(ref, "%w: %w", ErrLoadFailed, err)\n\t}\n\n\tstate, err = r.foldEvents(state, events, ref)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 9. storage/otel.go - fix startSaveSpan body
# ============================================================
def fix_storage_otel():
    path = "storage/otel.go"
    content = read_file(path)
    
    content = content.replace(
        '\treturn cqrsotel.StartSpan(\n\t\tctx, tracer(), spanName,\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(aggregateType, aggregateID),',
        '\treturn cqrsotel.StartSpan(\n\t\tctx, tracer(), spanName,\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(ref.Type, ref.ID),'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 10. storage/event_store.go - fix Save/AppendBatch/checkVersion bodies
# ============================================================
def fix_storage_event_store():
    path = "storage/event_store.go"
    content = read_file(path)
    
    # Save method body: fix calls
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\tctx, span := startSaveSpan(\n\t\tctx,\n\t\t"event.store.save",\n\t\taggregateType,\n\t\taggregateID,',
        '\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\tctx, span := startSaveSpan(\n\t\tctx,\n\t\t"event.store.save",\n\t\tref,'
    )
    content = content.replace(
        '\terr = s.checkVersion(ctx, tx, aggregateType, aggregateID, expectedVersion)',
        '\terr = s.checkVersion(ctx, tx, ref, expectedVersion)'
    )
    content = content.replace(
        '\terr = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)\n\tif err != nil {\n\t\treturn s.wrapInsertEventsErr(span, err, events, aggregateType, aggregateID)\n\t}\n\n\terr = commitTx(tx)',
        '\terr = s.insertEvents(ctx, tx, ref, events)\n\tif err != nil {\n\t\treturn s.wrapInsertEventsErr(span, err, events, ref)\n\t}\n\n\terr = commitTx(tx)'
    )
    
    # AppendBatch method body
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "event.store.append_batch",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(aggregateType, aggregateID),',
        '\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "event.store.append_batch",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(ref.Type, ref.ID),'
    )
    content = content.replace(
        '\terr = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)\n\tif err != nil {\n\t\treturn s.wrapInsertEventsErr(span, err, events, aggregateType, aggregateID)\n\t}\n\n\terr = commitTx(tx)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\t}\n\n\treturn err\n}\n\nfunc (s *SQLEventStore) wrapInsertEventsErr',
        '\terr = s.insertEvents(ctx, tx, ref, events)\n\tif err != nil {\n\t\treturn s.wrapInsertEventsErr(span, err, events, ref)\n\t}\n\n\terr = commitTx(tx)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\t}\n\n\treturn err\n}\n\nfunc (s *SQLEventStore) wrapInsertEventsErr'
    )
    
    # wrapInsertEventsErr body
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tcqrsotel.RecordError(span, err)\n\n\treturn event.WrapInfrastructure(err, "storage.insert_events",\n\t\tfmt.Sprintf("insert %d events for %s %s", len(events), aggregateType, aggregateID))',
        '\tcqrsotel.RecordError(span, err)\n\n\treturn event.WrapInfrastructure(err, "storage.insert_events",\n\t\tfmt.Sprintf("insert %d events for %s", len(events), ref))'
    )
    
    # checkVersion body
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tp1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)\n\n\tquery := fmt.Sprintf(checkVersionQuery, p1, p2)\n\n\treturn sharedCheckVersion(ctx, tx, aggregateType, aggregateID, expectedVersion, query)',
        '\tp1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)\n\n\tquery := fmt.Sprintf(checkVersionQuery, p1, p2)\n\n\treturn sharedCheckVersion(ctx, tx, ref, expectedVersion, query)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 11. storage/event_store_load.go - fix all Load methods
# ============================================================
def fix_storage_event_store_load():
    path = "storage/event_store_load.go"
    content = read_file(path)
    
    # loadWithSpan: fix the body
    content = content.replace(
        '\tevents, err := s.queryEvents(\n\t\tctx, aggregateType, aggregateID,\n\t\tp.where, p.extraArgs,\n\t\tp.requireHit, p.errMsg,\n\t)',
        '\tevents, err := s.queryEvents(\n\t\tctx, ref,\n\t\tp.where, p.extraArgs,\n\t\tp.requireHit, p.errMsg,\n\t)'
    )
    
    # loadSimple: fix signature and body
    content = content.replace(
        'func (s *SQLEventStore) loadSimple(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tspanName string,\n\torder string,\n\terrMsg string,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, aggregateType, aggregateID, loadParams{',
        'func (s *SQLEventStore) loadSimple(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tspanName string,\n\torder string,\n\terrMsg string,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, ref, loadParams{'
    )
    content = content.replace(
        '\t\tattrs:      cqrsotel.AggregateAttrs(aggregateType, aggregateID),\n\t\twhere:      order,',
        '\t\tattrs:      cqrsotel.AggregateAttrs(ref.Type, ref.ID),\n\t\twhere:      order,'
    )
    
    # Load: fix to pass ref instead of aggregateType, aggregateID
    content = content.replace(
        'func (s *SQLEventStore) Load(\n\tctx context.Context,\n\tref event.AggregateRef,\n) ([]event.Event, error) {\n\treturn s.loadSimple(\n\t\tctx,\n\t\taggregateType,\n\t\taggregateID,\n\t\t"event.store.load",',
        'func (s *SQLEventStore) Load(\n\tctx context.Context,\n\tref event.AggregateRef,\n) ([]event.Event, error) {\n\treturn s.loadSimple(\n\t\tctx,\n\t\tref,\n\t\t"event.store.load",'
    )
    
    # LoadFromVersion
    content = content.replace(
        'func (s *SQLEventStore) LoadFromVersion(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tversion event.Version,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, aggregateType, aggregateID, loadParams{',
        'func (s *SQLEventStore) LoadFromVersion(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tversion event.Version,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, ref, loadParams{'
    )
    content = content.replace(
        '\t\tattrs: append(\n\t\t\tcqrsotel.AggregateAttrs(aggregateType, aggregateID),\n\t\t\tattribute.Int(cqrsotel.AttrAggregateVersion, version.Int()),\n\t\t),',
        '\t\tattrs: append(\n\t\t\tcqrsotel.AggregateAttrs(ref.Type, ref.ID),\n\t\t\tattribute.Int(cqrsotel.AttrAggregateVersion, version.Int()),\n\t\t),'
    )
    
    # LoadToVersion
    content = content.replace(
        'func (s *SQLEventStore) LoadToVersion(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tmaxVersion event.Version,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, aggregateType, aggregateID, loadParams{',
        'func (s *SQLEventStore) LoadToVersion(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tmaxVersion event.Version,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, ref, loadParams{'
    )
    content = content.replace(
        '\t\tattrs: append(\n\t\t\tcqrsotel.AggregateAttrs(aggregateType, aggregateID),\n\t\t\tattribute.Int(cqrsotel.AttrAggregateVersion, maxVersion.Int()),\n\t\t),',
        '\t\tattrs: append(\n\t\t\tcqrsotel.AggregateAttrs(ref.Type, ref.ID),\n\t\t\tattribute.Int(cqrsotel.AttrAggregateVersion, maxVersion.Int()),\n\t\t),'
    )
    
    # LoadToTimestamp
    content = content.replace(
        'func (s *SQLEventStore) LoadToTimestamp(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tmaxTime time.Time,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, aggregateType, aggregateID, loadParams{',
        'func (s *SQLEventStore) LoadToTimestamp(\n\tctx context.Context,\n\tref event.AggregateRef,\n\tmaxTime time.Time,\n) ([]event.Event, error) {\n\treturn s.loadWithSpan(ctx, ref, loadParams{'
    )
    content = content.replace(
        '\t\tattrs:    cqrsotel.AggregateAttrs(aggregateType, aggregateID),',
        '\t\tattrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),'
    )
    
    # LoadBackwards
    content = content.replace(
        'func (s *SQLEventStore) LoadBackwards(\n\tctx context.Context,\n\tref event.AggregateRef,\n) ([]event.Event, error) {\n\treturn s.loadSimple(\n\t\tctx,\n\t\taggregateType,\n\t\taggregateID,',
        'func (s *SQLEventStore) LoadBackwards(\n\tctx context.Context,\n\tref event.AggregateRef,\n) ([]event.Event, error) {\n\treturn s.loadSimple(\n\t\tctx,\n\t\tref,'
    )
    
    # queryEvents body
    content = content.replace(
        '\targs := make([]any, 0, 2+len(extraArgs))\n\targs = append(args, string(aggregateType), aggregateID)',
        '\targs := make([]any, 0, 2+len(extraArgs))\n\targs = append(args, string(ref.Type), ref.ID)'
    )
    content = content.replace(
        '\t\t"aggregate %s/%s: %w",\n\t\t\taggregateType,\n\t\t\taggregateID,',
        '\t\t"aggregate %s: %w",\n\t\t\tref,'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 12. storage/event_store_scan.go - fix insertEvents body
# ============================================================
def fix_storage_event_store_scan():
    path = "storage/event_store_scan.go"
    content = read_file(path)
    
    content = content.replace(
        '\treturn sharedInsertEvents(\n\t\tctx, tx, aggregateType, aggregateID, events,\n\t\tinsertSQL, s.dialect.FormatTime,\n\t)',
        '\treturn sharedInsertEvents(\n\t\tctx, tx, ref, events,\n\t\tinsertSQL, s.dialect.FormatTime,\n\t)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 13. storage/transactional_store.go - fix SaveWithOutbox body
# ============================================================
def fix_storage_transactional_store():
    path = "storage/transactional_store.go"
    content = read_file(path)
    
    content = content.replace(
        '\tctx, span := startSaveSpan(\n\t\tctx,\n\t\t"event.store.save_with_outbox",\n\t\taggregateType,\n\t\taggregateID,\n\t\texpectedVersion,\n\t\tlen(events),\n\t)',
        '\tctx, span := startSaveSpan(\n\t\tctx,\n\t\t"event.store.save_with_outbox",\n\t\tref,\n\t\texpectedVersion,\n\t\tlen(events),\n\t)'
    )
    content = content.replace(
        '\terr := saveWithOutboxTx(\n\t\tctx,\n\t\ts.db,\n\t\taggregateType,\n\t\taggregateID,\n\t\tevents,\n\t\texpectedVersion,\n\t\ts.checkVersion,\n\t\ts.insertEvents,\n\t\ts.appendOutboxTx,\n\t)',
        '\terr := saveWithOutboxTx(\n\t\tctx,\n\t\ts.db,\n\t\tref,\n\t\tevents,\n\t\texpectedVersion,\n\t\ts.checkVersion,\n\t\ts.insertEvents,\n\t\ts.appendOutboxTx,\n\t)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 14. storage/snapshot.go - fix all methods
# ============================================================
def fix_storage_snapshot():
    path = "storage/snapshot.go"
    content = read_file(path)
    
    # Load method
    content = content.replace(
        '\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "snapshot.load",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(cqrsotel.AggregateAttrs(aggregateType, aggregateID)...),\n\t)\n\tdefer span.End()\n\n\tsnap, err := s.querySnapshot(ctx, aggregateType, aggregateID)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\n\t\treturn nil, event.WrapInfrastructure(err, "storage.load_snapshot",\n\t\t\tfmt.Sprintf("load snapshot for %s %s", aggregateType, aggregateID))\n\t}',
        '\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "snapshot.load",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...),\n\t)\n\tdefer span.End()\n\n\tsnap, err := s.querySnapshot(ctx, ref)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\n\t\treturn nil, event.WrapInfrastructure(err, "storage.load_snapshot",\n\t\t\tfmt.Sprintf("load snapshot for %s", ref))\n\t}'
    )
    
    # LoadAtVersion method
    content = content.replace(
        '\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "snapshot.load_at_version",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(aggregateType, aggregateID),',
        '\tctx, span := cqrsotel.StartSpan(\n\t\tctx, tracer(), "snapshot.load_at_version",\n\t\ttrace.SpanKindClient,\n\t\ttrace.WithAttributes(append(\n\t\t\tcqrsotel.AggregateAttrs(ref.Type, ref.ID),'
    )
    content = content.replace(
        '\tsnap, err := s.querySnapshot(ctx, aggregateType, aggregateID)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"storage.load_snapshot_version",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"load snapshot at version %d for %s %s",\n\t\t\t\tversion,\n\t\t\t\taggregateType,\n\t\t\t\taggregateID,\n\t\t\t),\n\t\t)\n\t}',
        '\tsnap, err := s.querySnapshot(ctx, ref)\n\tif err != nil {\n\t\tcqrsotel.RecordError(span, err)\n\n\t\treturn nil, event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"storage.load_snapshot_version",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"load snapshot at version %d for %s",\n\t\t\t\tversion,\n\t\t\t\tref,\n\t\t\t),\n\t\t)\n\t}'
    )
    content = content.replace(
        '\t\terr := event.WrapRejection(\n\t\t\tevent.ErrSnapshotNotFound,\n\t\t\t"storage.snapshot_version_exceeded",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"load snapshot at version %d for %s %s",\n\t\t\t\tversion,\n\t\t\t\taggregateType,\n\t\t\t\taggregateID,\n\t\t\t),\n\t\t)',
        '\t\terr := event.WrapRejection(\n\t\t\tevent.ErrSnapshotNotFound,\n\t\t\t"storage.snapshot_version_exceeded",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"load snapshot at version %d for %s",\n\t\t\t\tversion,\n\t\t\t\tref,\n\t\t\t),\n\t\t)'
    )
    
    # querySnapshot
    content = content.replace(
        'func (s *SQLSnapshotStore) querySnapshot(\n\tctx context.Context,\n\tref event.AggregateRef,\n) (*event.Snapshot, error) {\n\tp1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)\n\n\tquery := fmt.Sprintf(`SELECT version, state, created_at FROM `+tableSnapshots+`\n\t\tWHERE aggregate_type = %s AND aggregate_id = %s`, p1, p2)\n\n\treturn s.scanSnapshot(\n\t\ts.db.QueryRowContext(ctx, query, string(aggregateType), aggregateID),\n\t\taggregateType,\n\t\taggregateID,\n\t)',
        'func (s *SQLSnapshotStore) querySnapshot(\n\tctx context.Context,\n\tref event.AggregateRef,\n) (*event.Snapshot, error) {\n\tp1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)\n\n\tquery := fmt.Sprintf(`SELECT version, state, created_at FROM `+tableSnapshots+`\n\t\tWHERE aggregate_type = %s AND aggregate_id = %s`, p1, p2)\n\n\treturn s.scanSnapshot(\n\t\ts.db.QueryRowContext(ctx, query, string(ref.Type), ref.ID),\n\t\tref,\n\t)'
    )
    
    # scanSnapshot
    content = content.replace(
        'func (s *SQLSnapshotStore) scanSnapshot(\n\trow *sql.Row,\n\tref event.AggregateRef,\n) (*event.Snapshot, error) {\n\tvar (\n\t\tversion    int\n\t\tstateBytes []byte\n\t)\n\n\ttimeDest := s.dialect.ScanTimeDest()\n\n\terr := row.Scan(&version, &stateBytes, timeDest)\n\tif err != nil {\n\t\tif errors.Is(err, sql.ErrNoRows) {\n\t\t\treturn nil, event.WrapRejection(event.ErrSnapshotNotFound, "storage.snapshot_not_found",\n\t\t\t\tfmt.Sprintf("%s/%s at v%d", aggregateType, aggregateID, event.Version(version)))\n\t\t}\n\n\t\treturn nil, event.WrapInfrastructure(err, "storage.scan_snapshot",\n\t\t\tfmt.Sprintf("scan snapshot for %s/%s", aggregateType, aggregateID))\n\t}',
        'func (s *SQLSnapshotStore) scanSnapshot(\n\trow *sql.Row,\n\tref event.AggregateRef,\n) (*event.Snapshot, error) {\n\tvar (\n\t\tversion    int\n\t\tstateBytes []byte\n\t)\n\n\ttimeDest := s.dialect.ScanTimeDest()\n\n\terr := row.Scan(&version, &stateBytes, timeDest)\n\tif err != nil {\n\t\tif errors.Is(err, sql.ErrNoRows) {\n\t\t\treturn nil, event.WrapRejection(event.ErrSnapshotNotFound, "storage.snapshot_not_found",\n\t\t\t\tfmt.Sprintf("%s at v%d", ref, event.Version(version)))\n\t\t}\n\n\t\treturn nil, event.WrapInfrastructure(err, "storage.scan_snapshot",\n\t\t\tfmt.Sprintf("scan snapshot for %s", ref))\n\t}'
    )
    content = content.replace(
        '\treturn &event.Snapshot{\n\t\tAggregateID:   aggregateID,\n\t\tAggregateType: aggregateType,',
        '\treturn &event.Snapshot{\n\t\tAggregateID:   ref.ID,\n\t\tAggregateType: ref.Type,'
    )
    
    # Delete
    content = content.replace(
        '\treturn deleteByAggregate(\n\t\ts.db, ctx, aggregateType, aggregateID,\n\t\ttableSnapshots, p1, p2, "snapshot",\n\t)',
        '\treturn deleteByAggregate(\n\t\ts.db, ctx, ref,\n\t\ttableSnapshots, p1, p2, "snapshot",\n\t)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 15. storage/stream.go - fix LoadStream body
# ============================================================
def fix_storage_stream():
    path = "storage/stream.go"
    content = read_file(path)
    
    content = content.replace(
        '\ttrace.WithAttributes(cqrsotel.AggregateAttrs(aggregateType, aggregateID)...),',
        '\ttrace.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...),'
    )
    content = content.replace(
        '\trows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID)',
        '\trows, err := s.db.QueryContext(ctx, query, string(ref.Type), ref.ID)'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 16. storage/sql_helpers.go - fix helper functions
# ============================================================
def fix_storage_sql_helpers():
    path = "storage/sql_helpers.go"
    content = read_file(path)
    
    # deleteByAggregate
    content = content.replace(
        '\t_, err := db.ExecContext(ctx, query, string(aggregateType), aggregateID)\n\tif err != nil {\n\t\treturn event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"storage.delete_by_aggregate",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"delete %s from table %s for %s %s",\n\t\t\t\twhat,\n\t\t\t\ttable,\n\t\t\t\taggregateType,\n\t\t\t\taggregateID,\n\t\t\t),\n\t\t)\n\t}',
        '\t_, err := db.ExecContext(ctx, query, string(ref.Type), ref.ID)\n\tif err != nil {\n\t\treturn event.WrapInfrastructure(\n\t\t\terr,\n\t\t\t"storage.delete_by_aggregate",\n\t\t\tfmt.Sprintf(\n\t\t\t\t"delete %s from table %s for %s",\n\t\t\t\twhat,\n\t\t\t\ttable,\n\t\t\t\tref,\n\t\t\t),\n\t\t)\n\t}'
    )
    
    # sharedInsertEvents
    content = content.replace(
        '\t\t\tstring(aggregateType),\n\t\t\taggregateID,',
        '\t\t\tstring(ref.Type),\n\t\t\tref.ID,'
    )
    
    # sharedCheckVersion
    content = content.replace(
        '\terr := tx.QueryRowContext(ctx, query, string(aggregateType), aggregateID).',
        '\terr := tx.QueryRowContext(ctx, query, string(ref.Type), ref.ID).'
    )
    content = content.replace(
        '\t\treturn event.WrapConflict(ErrConcurrencyConflict, "storage.version_mismatch",\n\t\t\tfmt.Sprintf("expected version %d, got %d for %s %s",\n\t\t\t\texpectedVersion.Int(), currentVersion, aggregateType, aggregateID))',
        '\t\treturn event.WrapConflict(ErrConcurrencyConflict, "storage.version_mismatch",\n\t\t\tfmt.Sprintf("expected version %d, got %d for %s",\n\t\t\t\texpectedVersion.Int(), currentVersion, ref))'
    )
    
    # saveWithOutboxTx: fix callback types and body
    content = content.replace(
        '\tcheckVersionFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, event.Version) error,\n\tinsertEventsFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, []event.Event) error,',
        '\tcheckVersionFn func(context.Context, *sql.Tx, event.AggregateRef, event.Version) error,\n\tinsertEventsFn func(context.Context, *sql.Tx, event.AggregateRef, []event.Event) error,'
    )
    content = content.replace(
        '\taggregateType, aggregateID := ref.Type, ref.ID\n\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\ttx, err := db.BeginTx(ctx, nil)',
        '\tif len(events) == 0 {\n\t\treturn nil\n\t}\n\n\ttx, err := db.BeginTx(ctx, nil)'
    )
    content = content.replace(
        '\terr = checkVersionFn(ctx, tx, aggregateType, aggregateID, expectedVersion)',
        '\terr = checkVersionFn(ctx, tx, ref, expectedVersion)'
    )
    content = content.replace(
        '\terr = insertEventsFn(ctx, tx, aggregateType, aggregateID, events)',
        '\terr = insertEventsFn(ctx, tx, ref, events)'
    )
    content = content.replace(
        '\t\treturn event.WrapInfrastructure(err, "storage.check_version",\n\t\t\tfmt.Sprintf("check version for %s %s", aggregateType, aggregateID))',
        '\t\treturn event.WrapInfrastructure(err, "storage.check_version",\n\t\t\tfmt.Sprintf("check version for %s", ref))'
    )
    content = content.replace(
        '\t\treturn event.WrapInfrastructure(err, "storage.insert_events",\n\t\t\tfmt.Sprintf("insert %d events for %s %s", len(events), aggregateType, aggregateID))',
        '\t\treturn event.WrapInfrastructure(err, "storage.insert_events",\n\t\t\tfmt.Sprintf("insert %d events for %s", len(events), ref))'
    )
    
    # Remove unused id import
    content = content.replace('\t"github.com/larsartmann/go-cqrs-lite/core/pkg/id"\n', '')
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 17. storage/event_reconstruction.go - no changes needed (uses string params)
# ============================================================

# ============================================================
# Main: run all fixes
# ============================================================
def main():
    print("=== Fixing production code ===")
    fix_memory_store()
    fix_memory_store_load()
    fix_memory_snapshot()
    fix_memory_stream()
    fix_fake_store()
    fix_fake_snapshot()
    fix_decider()
    fix_decider_load()
    fix_storage_otel()
    fix_storage_event_store()
    fix_storage_event_store_load()
    fix_storage_event_store_scan()
    fix_storage_transactional_store()
    fix_storage_snapshot()
    fix_storage_stream()
    fix_storage_sql_helpers()
    
    print("\n=== Running go build to check ===")
    result = subprocess.run(
        ['go', 'build', './core/...', './memory/...', './storage/...', './testhelpers/...'],
        capture_output=True, text=True
    )
    if result.returncode == 0:
        print("Build succeeded!")
    else:
        print(f"Build errors:\n{result.stderr}")
    
    return 0 if result.returncode == 0 else 1

if __name__ == '__main__':
    sys.exit(main())
