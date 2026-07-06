package turso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	tursoclient "turso.tech/database/tursogo"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
)

func TestOpenSyncWithConfig_MemoryWithRemote(t *testing.T) {
	t.Parallel()

	_, err := turso.OpenSyncWithConfig(
		context.Background(),
		turso.DbPath(":memory:"),
		turso.RemoteURL("libsql://example.turso.io"),
		turso.AuthToken("test-token"),
	)
	if err == nil {
		t.Fatal("expected error for in-memory database with remote sync")
	}

	if !errors.Is(err, turso.ErrMemorySync) {
		t.Errorf("error = %v, want ErrMemorySync", err)
	}
}

func TestOpenSync_DelegatesToWithConfig(t *testing.T) {
	t.Parallel()

	// OpenSync and OpenSyncWithConfig must reject identically.
	_, err := turso.OpenSync(
		context.Background(),
		turso.DbPath(":memory:"),
		turso.RemoteURL("libsql://example.turso.io"),
		turso.AuthToken("test-token"),
	)
	if err == nil {
		t.Fatal("expected error for in-memory database with remote sync")
	}

	if !errors.Is(err, turso.ErrMemorySync) {
		t.Errorf("error = %v, want ErrMemorySync", err)
	}
}

func TestSyncOptions_ApplyToConfig(t *testing.T) {
	t.Parallel()

	base := tursoclient.TursoSyncDbConfig{
		Path:      "test.db",
		RemoteUrl: "libsql://example.turso.io",
		AuthToken: "token",
	}

	tests := []struct {
		name  string
		opt   turso.SyncOption
		check func(c tursoclient.TursoSyncDbConfig) bool
	}{
		{
			name:  "WithSyncNamespace",
			opt:   turso.WithSyncNamespace("my-ns"),
			check: func(c tursoclient.TursoSyncDbConfig) bool { return c.Namespace == "my-ns" },
		},
		{
			name:  "WithSyncClientName",
			opt:   turso.WithSyncClientName("my-app"),
			check: func(c tursoclient.TursoSyncDbConfig) bool { return c.ClientName == "my-app" },
		},
		{
			name:  "WithSyncLongPollTimeout",
			opt:   turso.WithSyncLongPollTimeout(30 * time.Second),
			check: func(c tursoclient.TursoSyncDbConfig) bool { return c.LongPollTimeoutMs == 30000 },
		},
		{
			name:  "WithSyncLongPollTimeout_ZeroIgnored",
			opt:   turso.WithSyncLongPollTimeout(0),
			check: func(c tursoclient.TursoSyncDbConfig) bool { return c.LongPollTimeoutMs == 0 },
		},
		{
			name: "WithSyncBootstrapIfEmpty_False",
			opt:  turso.WithSyncBootstrapIfEmpty(false),
			check: func(c tursoclient.TursoSyncDbConfig) bool {
				return c.BootstrapIfEmpty != nil && *c.BootstrapIfEmpty == false
			},
		},
		{
			name: "WithSyncBootstrapIfEmpty_True",
			opt:  turso.WithSyncBootstrapIfEmpty(true),
			check: func(c tursoclient.TursoSyncDbConfig) bool {
				return c.BootstrapIfEmpty != nil && *c.BootstrapIfEmpty == true
			},
		},
		{
			name:  "WithSyncBusyTimeout",
			opt:   turso.WithSyncBusyTimeout(10 * time.Second),
			check: func(c tursoclient.TursoSyncDbConfig) bool { return c.BusyTimeout == 10000 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := base
			tt.opt(&cfg)
			if !tt.check(cfg) {
				t.Errorf("option did not apply config correctly: %+v", cfg)
			}
		})
	}
}

func TestSyncOptions_Combined(t *testing.T) {
	t.Parallel()

	cfg := tursoclient.TursoSyncDbConfig{Path: "test.db"}

	turso.WithSyncNamespace("ns")(&cfg)
	turso.WithSyncClientName("app")(&cfg)
	turso.WithSyncLongPollTimeout(15 * time.Second)(&cfg)
	turso.WithSyncBusyTimeout(8 * time.Second)(&cfg)
	turso.WithSyncBootstrapIfEmpty(false)(&cfg)

	if cfg.Namespace != "ns" {
		t.Errorf("Namespace = %q, want ns", cfg.Namespace)
	}

	if cfg.ClientName != "app" {
		t.Errorf("ClientName = %q, want app", cfg.ClientName)
	}

	if cfg.LongPollTimeoutMs != 15000 {
		t.Errorf("LongPollTimeoutMs = %d, want 15000", cfg.LongPollTimeoutMs)
	}

	if cfg.BusyTimeout != 8000 {
		t.Errorf("BusyTimeout = %d, want 8000", cfg.BusyTimeout)
	}

	if cfg.BootstrapIfEmpty == nil || *cfg.BootstrapIfEmpty {
		t.Error("BootstrapIfEmpty should be set to false")
	}
}

func TestOpenSyncWithConfig_ErrorClassification(t *testing.T) {
	t.Parallel()

	_, err := turso.OpenSyncWithConfig(
		context.Background(),
		turso.DbPath(":memory:"),
		turso.RemoteURL("libsql://example.turso.io"),
		turso.AuthToken("token"),
		turso.WithSyncClientName("test"),
	)

	classification := errorfamily.Classify(err)
	if classification != errorfamily.Rejection {
		t.Errorf("Classify = %s, want Rejection", classification)
	}
}
