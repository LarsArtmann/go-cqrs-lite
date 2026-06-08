package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/pkg/config"
)

// AppConfig demonstrates using pkg/config for CQRS application configuration.
type AppConfig struct {
	StoreType string `json:"storeType"` // "memory" or "sqlite"
	Database  string `json:"database"`  // SQLite path (only used when storeType="sqlite")
	Port      int    `json:"port"`
}

// demonstrateConfig shows how to use pkg/config with environment overlays.
func demonstrateConfig() {
	tmpDir, err := os.MkdirTemp("", "cqrs-config-example")
	if err != nil {
		log.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	baseConfig := AppConfig{StoreType: "memory", Database: "", Port: 8080}

	baseData, err := json.MarshalIndent(baseConfig, "", "  ")
	if err != nil {
		log.Fatalf("marshal config: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(tmpDir, "app.json"),
		append(baseData, '\n'),
		0o644,
	); err != nil {
		log.Fatalf("write base config: %v", err)
	}

	prodConfig := AppConfig{StoreType: "sqlite", Database: "/data/events.db", Port: 3000}

	prodData, err := json.MarshalIndent(prodConfig, "", "  ")
	if err != nil {
		log.Fatalf("marshal prod config: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(tmpDir, "app.production.json"),
		append(prodData, '\n'),
		0o644,
	); err != nil {
		log.Fatalf("write prod config: %v", err)
	}

	fmt.Println("--- Config Demo (development) ---")

	os.Setenv("GO_ENV", "development")

	var devCfg AppConfig

	loader := config.NewLoader(tmpDir)

	if err := loader.Load("app", &devCfg); err != nil {
		log.Fatalf("load dev config: %v", err)
	}

	fmt.Printf("  storeType: %s\n", devCfg.StoreType)
	fmt.Printf("  port: %d\n", devCfg.Port)

	fmt.Println("--- Config Demo (production) ---")

	os.Setenv("GO_ENV", "production")

	var prodCfg AppConfig

	if err := loader.Load("app", &prodCfg); err != nil {
		log.Fatalf("load prod config: %v", err)
	}

	fmt.Printf("  storeType: %s\n", prodCfg.StoreType)
	fmt.Printf("  database: %s\n", prodCfg.Database)
	fmt.Printf("  port: %d\n", prodCfg.Port)
}
