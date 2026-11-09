package dgraphengine_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dropAll connects to Dgraph, drops all data (nodes + schema), and returns
// whether the operation succeeded. Returns false when Dgraph is unreachable
// so that individual tests can skip gracefully.
func dropAll() bool {
	client, err := dgo.NewClient(dgraphAddr(),
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Alter(ctx, &api.Operation{DropAll: true}); err != nil {
		log.Printf("dgraphengine TestMain: DropAll failed: %v", err)
		return false
	}

	return true
}

// TestMain ensures a clean Dgraph state before and after the test suite.
// DropAll before: clean slate so prior test runs don't interfere.
// DropAll after: prevent stale data accumulation on persistent instances.
// Each engine.New() call re-applies the schema via init(), so DropAll
// (which also wipes schema) is safe — tests create fresh engines.
func TestMain(m *testing.M) {
	available := dropAll() // clean slate before tests
	if !available {
		log.Println("dgraphengine: Dgraph not available, tests will skip")
	}

	code := m.Run()

	if available {
		dropAll() // cleanup after tests
	}

	os.Exit(code)
}
