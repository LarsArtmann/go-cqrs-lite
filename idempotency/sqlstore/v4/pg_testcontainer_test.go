//go:build integration

package sqlstore_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

func pgTestDSN(t *testing.T) string { return pgtestcontainer.DSN(t) }
