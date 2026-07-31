package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC030(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
	}{
		{"DetectsInfiniteLoopWithoutCancel", `package main

import "context"

func worker(ctx context.Context) {
	for {
		doWork()
	}
}
`, 1},
		{"NoFindingWhenCtxDoneInSelect", `package main

import "context"

func worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			doWork()
		}
	}
}
`, 0},
		{"NoFindingForBoundedLoop", `package main

func worker() {
	for i := 0; i < 10; i++ {
		doWork()
	}
}
`, 0},
		{"NoFindingWhenDoneOnNonCtxReceiver", `package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-r.Context().Done():
			return
		}
	}
}
`, 0},
		{"NoFindingWhenCtxErrCheck", `package main

import "context"

func poll(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		doWork()
	}
}
`, 0},
		{"NoFindingWhenLoopHasBreak", `package main

func reconstruct(parent map[int]int, end int) []int {
	var path []int
	for k := end; ; k = parent[k] {
		path = append(path, k)
		if k == parent[k] {
			break
		}
	}
	return path
}
`, 0},
		{"NoFindingWhenCustomStopChannel", `package main

import "time"

func sampler(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			doWork()
		}
	}
}
`, 0},
		{"StillFlagsLoopWithOnlyReturnInGoroutine", `package main

func worker() {
	for {
		go func() {
			return
		}()
		doWork()
	}
}
`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				"worker.go": tt.source,
			})
			findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
			ruletest.AssertRule(t, findings, "C030", tt.wantCount)
		})
	}
}
