package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	tmp := t.TempDir()

	base := `{"database": {"host": "localhost", "port": 5432}}`
	overlay := `{"database": {"host": "production.db.example.com"}}`

	if err := os.WriteFile(filepath.Join(tmp, "app.json"), []byte(base), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(tmp, "app.production.json"),
		[]byte(overlay),
		0o600,
	); err != nil {
		t.Fatalf("write overlay: %v", err)
	}

	t.Setenv("GO_ENV", "production")

	loader := NewLoader(tmp)
	var cfg struct {
		Database struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"database"`
	}

	if err := loader.Load("app", &cfg); err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Database.Host != "production.db.example.com" {
		t.Fatalf("expected production host, got %q", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Fatalf("expected port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoader_Load_NoOverlay(t *testing.T) {
	tmp := t.TempDir()

	base := `{"app": {"name": "test-app"}}`
	if err := os.WriteFile(filepath.Join(tmp, "app.json"), []byte(base), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}

	loader := NewLoader(tmp)
	var cfg struct {
		App struct {
			Name string `json:"name"`
		} `json:"app"`
	}

	if err := loader.Load("app", &cfg); err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.App.Name != "test-app" {
		t.Fatalf("expected test-app, got %q", cfg.App.Name)
	}
}

func TestLoader_Load_MissingFile(t *testing.T) {
	loader := NewLoader(t.TempDir())
	var cfg struct{}

	err := loader.Load("missing", &cfg)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
