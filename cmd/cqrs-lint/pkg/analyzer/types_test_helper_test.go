package analyzer

import (
	"testing"
)

func TestBuildContextWithTypes(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

import "fmt"

type MyStore struct{}

func (s *MyStore) Close() error { return nil }

func main() {
	s := &MyStore{}
	_ = s.Close()
	fmt.Println("ok")
}
`,
	}

	ctx, cleanup := BuildContextWithTypes(t, "1.26", sources)
	defer cleanup()

	if len(ctx.GoFiles) == 0 {
		t.Fatal("expected at least 1 GoFile")
	}

	gf := ctx.GoFiles[0]
	if gf.Pkg == nil {
		t.Fatal("Pkg is nil")
	}
	if gf.Pkg.TypesInfo == nil {
		t.Fatal("TypesInfo is nil")
	}
	if gf.Pkg.TypesInfo.Types == nil {
		t.Fatal("Types map is nil — type-checking did not run")
	}

	t.Logf("TypesInfo.Types has %d entries", len(gf.Pkg.TypesInfo.Types))
	if len(gf.Pkg.TypesInfo.Types) == 0 {
		t.Error("Types map is empty — expected populated entries")
	}
}
