package eventtest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

// AssertGolden compares got against the golden snapshot at path.
//
// Powered by go-snaps — provides colored diff output, automatic snapshot
// management, and standardized update via UPDATE_SNAPS=true env var.
//
// The update parameter (from each caller's -update flag) is honored when true.
// To update all golden files: UPDATE_SNAPS=true go test ./...
// To clean obsolete snapshots:  UPDATE_SNAPS=clean go test ./...
func AssertGolden(t *testing.T, path string, got []byte, update bool) {
	t.Helper()

	opts := []func(*snaps.Config){
		snaps.Dir(filepath.Dir(path)),
		snaps.Filename(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
	}

	if update {
		opts = append(opts, snaps.Update(true))
	}

	snaps.WithConfig(opts...).MatchSnapshot(t, string(got)) //art-dupl:accept go-snaps golden config; catalog copy in cattest cannot share (separate modules)
}
