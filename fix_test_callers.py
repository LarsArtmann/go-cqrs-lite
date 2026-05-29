#!/usr/bin/env python3
"""Fix test callers to use AggregateRef instead of separate (aggType, aggID) args."""
import re
import subprocess
import sys

def read_file(path):
    with open(path) as f:
        return f.read()

def write_file(path, content):
    with open(path, 'w') as f:
        f.write(content)

def ref_wrap(match):
    """Wrap separate aggType, aggID args into AggregateRef."""
    return match.group(0)  # placeholder

# Fix all test files: replace store.Save(ctx, aggType, aggID, ...) with store.Save(ctx, event.NewAggregateRef(aggType, aggID), ...)
# and similar patterns for Load, LoadFromVersion, AppendBatch, etc.

def fix_test_callers(content, patterns):
    """Apply patterns to wrap separate args into AggregateRef."""
    for old, new in patterns:
        content = content.replace(old, new)
    return content

# ============================================================
# 1. core/decider/decider_helpers_test.go
# ============================================================
def fix_decider_helpers_test():
    path = "core/decider/decider_helpers_test.go"
    content = read_file(path)
    
    # mustAppendBatch
    content = content.replace(
        'err := store.AppendBatch(t.Context(), aggType, aggID, events)',
        'err := store.AppendBatch(t.Context(), event.NewAggregateRef(aggType, aggID), events)'
    )
    
    # ctxCheckStore.Save
    content = content.replace(
        '''func (c *ctxCheckStore) Save(
\tctx context.Context,
\taggType event.AggregateType,
\taggID id.AggregateID,
\tevents []event.Event,
\texpectedVersion event.Version,
) error {
\tif err := c.checkCtx(ctx); err != nil {
\t\treturn err
\t}

\treturn c.Store.Save(ctx, aggType, aggID, events, expectedVersion)
}''',
        '''func (c *ctxCheckStore) Save(
\tctx context.Context,
\tref event.AggregateRef,
\tevents []event.Event,
\texpectedVersion event.Version,
) error {
\tif err := c.checkCtx(ctx); err != nil {
\t\treturn err
\t}

\treturn c.Store.Save(ctx, ref, events, expectedVersion)
}'''
    )
    
    # ctxCheckStore.Load
    content = content.replace(
        '''func (c *ctxCheckStore) Load(
\tctx context.Context,
\taggType event.AggregateType,
\taggID id.AggregateID,
) ([]event.Event, error) {
\tif err := c.checkCtx(ctx); err != nil {
\t\treturn nil, err
\t}

\treturn c.Store.Load(ctx, aggType, aggID)
}''',
        '''func (c *ctxCheckStore) Load(
\tctx context.Context,
\tref event.AggregateRef,
) ([]event.Event, error) {
\tif err := c.checkCtx(ctx); err != nil {
\t\treturn nil, err
\t}

\treturn c.Store.Load(ctx, ref)
}'''
    )
    
    # errStore.LoadToVersion
    content = content.replace(
        '''func (e *errStore) LoadToVersion(
\t_ context.Context,
\t_ event.AggregateType,
\t_ id.AggregateID,
\t_ event.Version,
) ([]event.Event, error) {''',
        '''func (e *errStore) LoadToVersion(
\t_ context.Context,
\t_ event.AggregateRef,
\t_ event.Version,
) ([]event.Event, error) {'''
    )
    
    # errStore.LoadToTimestamp
    content = content.replace(
        '''func (e *errStore) LoadToTimestamp(
\t_ context.Context,
\t_ event.AggregateType,
\t_ id.AggregateID,
\t_ time.Time,
) ([]event.Event, error) {''',
        '''func (e *errStore) LoadToTimestamp(
\t_ context.Context,
\t_ event.AggregateRef,
\t_ time.Time,
) ([]event.Event, error) {'''
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 2. core/decider/decider_execute_test.go
# ============================================================
def fix_decider_execute_test():
    path = "core/decider/decider_execute_test.go"
    content = read_file(path)
    
    # SaveFn callback
    content = content.replace(
        'func(_ context.Context, _ event.AggregateType, _ id.AggregateID, _ []event.Event, _ event.Version) error {\n\t\t\treturn errors.New("db connection lost")\n\t\t}',
        'func(_ context.Context, _ event.AggregateRef, _ []event.Event, _ event.Version) error {\n\t\t\treturn errors.New("db connection lost")\n\t\t}'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 3. core/decider/decider_load_test.go
# ============================================================
def fix_decider_load_test():
    path = "core/decider/decider_load_test.go"
    content = read_file(path)
    
    # LoadFn callback
    content = content.replace(
        'func(_ event.AggregateType, _ id.AggregateID) ([]event.Event, error) {\n\t\t\treturn nil, errors.New("db unavailable")\n\t\t}',
        'func(_ event.AggregateRef) ([]event.Event, error) {\n\t\t\treturn nil, errors.New("db unavailable")\n\t\t}'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 4. core/decider/decider_coverage_test.go
# ============================================================
def fix_decider_coverage_test():
    path = "core/decider/decider_coverage_test.go"
    content = read_file(path)
    
    # fakeTransactionalStore.SaveWithOutbox
    content = content.replace(
        '''func (f *fakeTransactionalStore) SaveWithOutbox(
\t_ context.Context,
\t_ event.AggregateType,
\t_ id.AggregateID,
\t_ []event.Event,
\t_ event.Version,
) error {''',
        '''func (f *fakeTransactionalStore) SaveWithOutbox(
\t_ context.Context,
\t_ event.AggregateRef,
\t_ []event.Event,
\t_ event.Version,
) error {'''
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 5. core/decider/decider_snapshot_test.go - check if exists
# ============================================================
def fix_decider_snapshot_test():
    import os
    path = "core/decider/decider_snapshot_test.go"
    if not os.path.exists(path):
        print(f"Skip {path} (not found)")
        return
    content = read_file(path)
    
    # LoadFromVersionFn callback
    content = content.replace(
        'func(_ event.AggregateType, _ id.AggregateID, _ event.Version) ([]event.Event, error) {\n\t\t\treturn nil, nil\n\t\t}',
        'func(_ event.AggregateRef, _ event.Version) ([]event.Event, error) {\n\t\t\treturn nil, nil\n\t\t}'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 6. core/event/event_bdd_test.go - fix store.Save/Load/AppendBatch callers
# ============================================================
def fix_event_bdd_test():
    path = "core/event/event_bdd_test.go"
    content = read_file(path)
    
    # Wrap all store method calls with aggType, aggID -> AggregateRef
    # store.Save(ctx, aggType, aggID, events, version) -> store.Save(ctx, event.NewAggregateRef(aggType, aggID), events, version)
    # store.Load(ctx, aggType, aggID) -> store.Load(ctx, event.NewAggregateRef(aggType, aggID))
    # store.AppendBatch(ctx, aggType, aggID, events) -> store.AppendBatch(ctx, event.NewAggregateRef(aggType, aggID), events)
    
    # store.Save patterns
    content = content.replace(
        'store.Save(ctx, aggType, aggID, []event.Event{',
        'store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{'
    )
    content = content.replace(
        'store.Load(ctx, aggType, aggID)',
        'store.Load(ctx, event.NewAggregateRef(aggType, aggID))'
    )
    content = content.replace(
        'store.AppendBatch(ctx, aggType, aggID, ',
        'store.AppendBatch(ctx, event.NewAggregateRef(aggType, aggID), '
    )
    # versioned.Load/AppendBatch
    content = content.replace(
        'versioned.Load(ctx, aggType, aggID)',
        'versioned.Load(ctx, event.NewAggregateRef(aggType, aggID))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 7. memory/store_test.go and store_extra_test.go
# ============================================================
def fix_memory_store_test():
    for path in ["memory/store_test.go", "memory/store_extra_test.go"]:
        content = read_file(path)
        
        # All the patterns: store.Save(ctx, "User", aggID, ...) -> store.Save(ctx, event.NewAggregateRef("User", aggID), ...)
        # Use regex for flexibility
        content = re.sub(
            r'store\.Save\(ctx,\s+"User",\s+aggID,\s+',
            'store.Save(ctx, event.NewAggregateRef("User", aggID), '
        )
        content = re.sub(
            r'store\.Save\(ctx,\s+"TestAggregate",\s+aggID,\s+',
            'store.Save(ctx, event.NewAggregateRef("TestAggregate", aggID), '
        )
        content = re.sub(
            r'store\.Load\(ctx,\s+"User",\s+aggID\)',
            'store.Load(ctx, event.NewAggregateRef("User", aggID))'
        )
        content = re.sub(
            r'store\.Load\(ctx,\s+"TestAggregate",\s+aggID\)',
            'store.Load(ctx, event.NewAggregateRef("TestAggregate", aggID))'
        )
        content = re.sub(
            r'store\.LoadFromVersion\(ctx,\s+"User",\s+aggID,\s+',
            'store.LoadFromVersion(ctx, event.NewAggregateRef("User", aggID), '
        )
        content = re.sub(
            r'store\.LoadFromVersion\(ctx,\s+"TestAggregate",\s+aggID,\s+',
            'store.LoadFromVersion(ctx, event.NewAggregateRef("TestAggregate", aggID), '
        )
        content = re.sub(
            r'store\.AppendBatch\(ctx,\s+"User",\s+aggID,\s+',
            'store.AppendBatch(ctx, event.NewAggregateRef("User", aggID), '
        )
        content = re.sub(
            r'store\.AppendBatch\(ctx,\s+"TestAggregate",\s+aggID,\s+',
            'store.AppendBatch(ctx, event.NewAggregateRef("TestAggregate", aggID), '
        )
        content = re.sub(
            r'store\.LoadToVersion\(ctx,\s+"User",\s+aggID,\s+',
            'store.LoadToVersion(ctx, event.NewAggregateRef("User", aggID), '
        )
        content = re.sub(
            r'store\.LoadToTimestamp\(ctx,\s+"User",\s+aggID,\s+',
            'store.LoadToTimestamp(ctx, event.NewAggregateRef("User", aggID), '
        )
        content = re.sub(
            r'store\.LoadBackwards\(ctx,\s+"User",\s+aggID\)',
            'store.LoadBackwards(ctx, event.NewAggregateRef("User", aggID))'
        )
        content = re.sub(
            r'backwardsLoader\.LoadBackwards\(ctx,\s+"User",\s+aggID\)',
            'backwardsLoader.LoadBackwards(ctx, event.NewAggregateRef("User", aggID))'
        )
        content = re.sub(
            r'backwardsLoader\.LoadBackwards\(context\.Background\(\),\s+"User",\s+id\.AggregateID\{\}\)',
            'backwardsLoader.LoadBackwards(context.Background(), event.AggregateRef{})'
        )
        content = re.sub(
            r'store\.Save\(ctx,\s+event\.AggregateType\("User"\),\s+agg1,\s+',
            'store.Save(ctx, event.NewAggregateRef(event.AggregateType("User"), agg1), '
        )
        content = re.sub(
            r'store\.Save\(ctx,\s+event\.AggregateType\("Order"\),\s+agg2,\s+',
            'store.Save(ctx, event.NewAggregateRef(event.AggregateType("Order"), agg2), '
        )
        
        write_file(path, content)
        print(f"Fixed {path}")

# ============================================================
# 8. memory/memory_bdd_test.go
# ============================================================
def fix_memory_bdd_test():
    path = "memory/memory_bdd_test.go"
    content = read_file(path)
    
    content = content.replace(
        'store.Save(ctx, "TestAggregate", aggID, events, 0)',
        'store.Save(ctx, event.NewAggregateRef("TestAggregate", aggID), events, 0)'
    )
    content = content.replace(
        'store.Save(ctx, "TestAggregate", aggID, more, 0)',
        'store.Save(ctx, event.NewAggregateRef("TestAggregate", aggID), more, 0)'
    )
    content = content.replace(
        'store.Load(ctx, "TestAggregate", aggID)',
        'store.Load(ctx, event.NewAggregateRef("TestAggregate", aggID))'
    )
    content = content.replace(
        'store.AppendBatch(ctx, "TestAggregate", aggID, batch)',
        'store.AppendBatch(ctx, event.NewAggregateRef("TestAggregate", aggID), batch)'
    )
    content = content.replace(
        'store.LoadFromVersion(ctx, "TestAggregate", aggID, 2)',
        'store.LoadFromVersion(ctx, event.NewAggregateRef("TestAggregate", aggID), 2)'
    )
    content = content.replace(
        'store.LoadFromVersion(ctx, "TestAggregate", aggID, 1)',
        'store.LoadFromVersion(ctx, event.NewAggregateRef("TestAggregate", aggID), 1)'
    )
    content = content.replace(
        'store.Save(ctx, "TestAggregate", aggID,\n\t\t\t\t[]event.Event{makeMemEvent("Created", aggID, 1)}, 0)',
        'store.Save(ctx, event.NewAggregateRef("TestAggregate", aggID),\n\t\t\t\t[]event.Event{makeMemEvent("Created", aggID, 1)}, 0)'
    )
    content = content.replace(
        'snapStore.Load(ctx, aggType, aggID)',
        'snapStore.Load(ctx, event.NewAggregateRef(aggType, aggID))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

# ============================================================
# 9. integration tests
# ============================================================
def fix_integration_tests():
    # event_sourcing_bdd_test.go
    path = "integration/event/event_sourcing_bdd_test.go"
    content = read_file(path)
    
    content = content.replace(
        'store.Save(ctx, aggType, aggID, events, event.Version(0))',
        'store.Save(ctx, event.NewAggregateRef(aggType, aggID), events, event.Version(0))'
    )
    content = content.replace(
        'store.Save(ctx, aggType, aggID, second, event.Version(1))',
        'store.Save(ctx, event.NewAggregateRef(aggType, aggID), second, event.Version(1))'
    )
    content = content.replace(
        'store.Save(ctx, aggType, aggID, conflicting, event.Version(0))',
        'store.Save(ctx, event.NewAggregateRef(aggType, aggID), conflicting, event.Version(0))'
    )
    content = content.replace(
        'store.Load(ctx, aggType, aggID)',
        'store.Load(ctx, event.NewAggregateRef(aggType, aggID))'
    )
    content = content.replace(
        'store.Load(ctx, aggType, id.NewAggregateID())',
        'store.Load(ctx, event.NewAggregateRef(aggType, id.NewAggregateID()))'
    )
    content = content.replace(
        'store.LoadFromVersion(ctx, aggType, aggID, event.Version(2))',
        'store.LoadFromVersion(ctx, event.NewAggregateRef(aggType, aggID), event.Version(2))'
    )
    content = content.replace(
        'store.LoadFromVersion(ctx, aggType, aggID, event.Version(99))',
        'store.LoadFromVersion(ctx, event.NewAggregateRef(aggType, aggID), event.Version(99))'
    )
    content = content.replace(
        'store.AppendBatch(ctx, aggType, aggID, events)',
        'store.AppendBatch(ctx, event.NewAggregateRef(aggType, aggID), events)'
    )
    content = content.replace(
        'store.Save(ctx, aggType, id.NewAggregateID(), nil, event.Version(0))',
        'store.Save(ctx, event.NewAggregateRef(aggType, id.NewAggregateID()), nil, event.Version(0))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")
    
    # metadata_roundtrip_test.go
    path = "integration/event/metadata_roundtrip_test.go"
    content = read_file(path)
    
    content = content.replace(
        '''err = store.Save(
\t\t\tctx,
\t\t\tevent.AggregateType("User"),
\t\t\taggID,
\t\t\t[]event.Event{evt},
\t\t\tevent.Version(0),
\t\t)''',
        '''err = store.Save(
\t\t\tctx,
\t\t\tevent.NewAggregateRef(event.AggregateType("User"), aggID),
\t\t\t[]event.Event{evt},
\t\t\tevent.Version(0),
\t\t)'''
    )
    content = content.replace(
        '''err = store.Save(
\t\t\tctx,
\t\t\tevent.AggregateType("Test"),
\t\t\taggID,
\t\t\t[]event.Event{evt},
\t\t\tevent.Version(0),
\t\t)''',
        '''err = store.Save(
\t\t\tctx,
\t\t\tevent.NewAggregateRef(event.AggregateType("Test"), aggID),
\t\t\t[]event.Event{evt},
\t\t\tevent.Version(0),
\t\t)'''
    )
    content = content.replace(
        'store.Load(ctx, event.AggregateType("User"), aggID)',
        'store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))'
    )
    content = content.replace(
        'store.Load(ctx, event.AggregateType("Test"), aggID)',
        'store.Load(ctx, event.NewAggregateRef(event.AggregateType("Test"), aggID))'
    )
    content = content.replace(
        'store.Save(ctx, aggType, aggID, events, event.Version(0))',
        'store.Save(ctx, event.NewAggregateRef(aggType, aggID), events, event.Version(0))'
    )
    content = content.replace(
        'store.LoadFromVersion(ctx, aggType, aggID, event.Version(1))',
        'store.LoadFromVersion(ctx, event.NewAggregateRef(aggType, aggID), event.Version(1))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")
    
    # benchmark_test.go
    path = "integration/event/benchmark_test.go"
    content = read_file(path)
    content = content.replace(
        '_ = store.Save(ctx, "Bench", aggregateID, []event.Event{evt}, 1)',
        '_ = store.Save(ctx, event.NewAggregateRef("Bench", aggregateID), []event.Event{evt}, 1)'
    )
    content = content.replace(
        '_ = store.Save(ctx, "Bench", aggregateID, []event.Event{evt}, event.Version(i+1))',
        '_ = store.Save(ctx, event.NewAggregateRef("Bench", aggregateID), []event.Event{evt}, event.Version(i+1))'
    )
    content = content.replace(
        '_, _ = store.Load(ctx, "Bench", aggregateID)',
        '_, _ = store.Load(ctx, event.NewAggregateRef("Bench", aggregateID))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")
    
    # timetravel_test.go
    path = "integration/event/timetravel_test.go"
    content = read_file(path)
    content = content.replace(
        'store.AppendBatch(ctx, "Counter", aggID, []event.Event{evt1, evt2, evt3})',
        'store.AppendBatch(ctx, event.NewAggregateRef("Counter", aggID), []event.Event{evt1, evt2, evt3})'
    )
    content = content.replace(
        'store.LoadToVersion(ctx, "Counter", aggID, 2)',
        'store.LoadToVersion(ctx, event.NewAggregateRef("Counter", aggID), 2)'
    )
    content = content.replace(
        'store.LoadToTimestamp(ctx, "Counter", aggID, now.Add(-15*time.Minute))',
        'store.LoadToTimestamp(ctx, event.NewAggregateRef("Counter", aggID), now.Add(-15*time.Minute))'
    )
    content = content.replace(
        'store.AppendBatch(ctx, "Counter", aggID, []event.Event{evt1, evt3})',
        'store.AppendBatch(ctx, event.NewAggregateRef("Counter", aggID), []event.Event{evt1, evt3})'
    )
    content = content.replace(
        'store.AppendBatch(ctx, "Counter", aggID, []event.Event{evt2})',
        'store.AppendBatch(ctx, event.NewAggregateRef("Counter", aggID), []event.Event{evt2})'
    )
    content = content.replace(
        'store.AppendBatch(ctx, "Issue", aggID1, []event.Event{evt1, evt3})',
        'store.AppendBatch(ctx, event.NewAggregateRef("Issue", aggID1), []event.Event{evt1, evt3})'
    )
    content = content.replace(
        'store.AppendBatch(ctx, "Issue", aggID2, []event.Event{evt2})',
        'store.AppendBatch(ctx, event.NewAggregateRef("Issue", aggID2), []event.Event{evt2})'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")
    
    # full_flow_test.go
    path = "integration/full_flow_test.go"
    content = read_file(path)
    content = content.replace(
        'store.Load(ctx, "User", gu.AggregateID)',
        'store.Load(ctx, event.NewAggregateRef("User", gu.AggregateID))'
    )
    content = content.replace(
        'store.Load(ctx, "User", aggID)',
        'store.Load(ctx, event.NewAggregateRef("User", aggID))'
    )
    content = content.replace(
        'store.LoadStream(ctx, "User", aggID)',
        'store.LoadStream(ctx, event.NewAggregateRef("User", aggID))'
    )
    
    write_file(path, content)
    print(f"Fixed {path}")
    
    # stream/listbuilder_bdd_test.go
    path = "stream/listbuilder_bdd_test.go"
    content = read_file(path)
    content = content.replace(
        '''store.Save(ctx, "User", activeID, []event.Event{activeEvt}, event.Version(0))''',
        '''store.Save(ctx, event.NewAggregateRef("User", activeID), []event.Event{activeEvt}, event.Version(0))'''
    )
    content = content.replace(
        '''store.Save(ctx, "User", deletedID, []event.Event{deletedEvt}, event.Version(0))''',
        '''store.Save(ctx, event.NewAggregateRef("User", deletedID), []event.Event{deletedEvt}, event.Version(0))'''
    )
    content = content.replace(
        '''store.Save(ctx, "Order", orderID, []event.Event{orderEvt}, event.Version(0))''',
        '''store.Save(ctx, event.NewAggregateRef("Order", orderID), []event.Event{orderEvt}, event.Version(0))'''
    )
    
    write_file(path, content)
    print(f"Fixed {path}")

def main():
    fix_decider_helpers_test()
    fix_decider_execute_test()
    fix_decider_load_test()
    fix_decider_coverage_test()
    fix_decider_snapshot_test()
    fix_event_bdd_test()
    fix_memory_store_test()
    fix_memory_bdd_test()
    fix_integration_tests()
    
    print("\n=== Running tests to check ===")
    result = subprocess.run(
        ['go', 'test', '-count=1', '-run', '^$', './core/...', './memory/...', './storage/...', 
         './testhelpers/...', './integration/...', './stream/...'],
        capture_output=True, text=True
    )
    if result.returncode == 0:
        print("Test compilation succeeded!")
    else:
        print(f"Test compilation errors:\n{result.stderr}")
    return 0 if result.returncode == 0 else 1

if __name__ == '__main__':
    sys.exit(main())
