package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func TestExporter_Export_Channel(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddChannel(catalog.Channel{
		ID:        "order-events",
		Name:      "Order Events",
		Version:   "1.0.0",
		Summary:   "All order-related events",
		Address:   "orders.{env}.events",
		Protocols: []string{"kafka"},
		Owners:    []string{"platform-team"},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "channels", "order-events", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read channel file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "id: order-events")
	assertContains(t, content, "name: Order Events")
	assertContains(t, content, "address: \"orders.{env}.events\"")
	assertContains(t, content, "- kafka")
	assertContains(t, content, "- platform-team")
	assertContains(t, content, "<NodeGraph />")
}

func TestExporter_Export_ChannelWithParams(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddChannel(catalog.Channel{
		ID:      "order-events",
		Name:    "Order Events",
		Version: "1.0.0",
		Parameters: map[string]catalog.ChannelParam{
			"env": {Enum: []string{"dev", "prod"}, Default: "dev"},
		},
		DeliveryGuarantee: "at-least-once",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "channels", "order-events", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read channel file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "deliveryGuarantee: at-least-once")
	assertContains(t, content, "parameters:")
	assertContains(t, content, "env:")
	assertContains(t, content, "- dev")
	assertContains(t, content, "- prod")
	assertContains(t, content, "default: dev")
}

func TestExporter_Export_DataStore(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDataStore(catalog.DataStore{
		ID:             "orders-db",
		Name:           "Orders Database",
		Version:        "1.0.0",
		ContainerType:  "database",
		Technology:     "postgres@16",
		Classification: "internal",
		Summary:        "Primary order data store",
		Owners:         []string{"order-team"},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "data", "orders-db", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read data store file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "id: orders-db")
	assertContains(t, content, "container_type: database")
	assertContains(t, content, "technology: postgres@16")
	assertContains(t, content, "classification: internal")
	assertContains(t, content, "- order-team")
	assertContains(t, content, "<NodeGraph />")
}

func TestExporter_Export_Flow(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddFlow(catalog.Flow{
		ID: "create-order", Name: "Create Order Flow", Version: "1.0.0",
		Summary: "Complete order creation flow",
		Steps: []catalog.FlowStep{
			{
				ID: "1", Title: "Create Order",
				Message:  &catalog.FlowStepRef{ID: "CreateOrder", Version: "1.0.0"},
				NextStep: &catalog.FlowEdge{ID: "2", Label: "submit"},
			},
			{
				ID: "2", Title: "Order Service",
				Service: &catalog.FlowStepRef{ID: "order-svc", Version: "1.0.0"},
			},
		},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "flows", "create-order", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read flow file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "id: create-order")
	assertContains(t, content, "steps:")
	assertContains(t, content, "message:")
	assertContains(t, content, "id: CreateOrder")
	assertContains(t, content, "service:")
	assertContains(t, content, "id: order-svc")
	assertContains(t, content, "next_step:")
	assertContains(t, content, "label: \"submit\"")
}

func TestExporter_Export_Team(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddTeam(catalog.Team{
		ID:      "order-team",
		Name:    "Order Team",
		Summary: "Team responsible for orders",
		Members: []string{"alice", "bob"},
		Email:   "orders@example.com",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "teams", "order-team.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read team file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "id: order-team")
	assertContains(t, content, "name: Order Team")
	assertContains(t, content, "- alice")
	assertContains(t, content, "- bob")
	assertContains(t, content, "email: \"orders@example.com\"")
}

func TestExporter_Export_User(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddUser(catalog.User{
		ID:    "alice",
		Name:  "Alice Smith",
		Role:  "Senior Engineer",
		Email: "alice@example.com",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "users", "alice.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read user file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "id: alice")
	assertContains(t, content, "name: Alice Smith")
	assertContains(t, content, "role: \"Senior Engineer\"")
}

func TestExporter_Export_ServiceWithBadges(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
		Badges: []catalog.Badge{
			{Content: "Production", BackgroundColor: "green", TextColor: "green"},
		},
		Repository: &catalog.Repository{Language: "Go", URL: "https://github.com/example/orders"},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "services", "order-svc", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "badges:")
	assertContains(t, content, "content: \"Production\"")
	assertContains(t, content, "backgroundColor: green")
	assertContains(t, content, "repository:")
	assertContains(t, content, "language: \"Go\"")
	assertContains(t, content, "<NodeGraph />")
}

func TestAutoDerive_ProducersConsumers(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddEvent("order-svc", newEvent("OrderCreated", "Order Created", catalog.Sends))
	reg.AddService(catalog.Service{ID: "notif-svc", Name: "Notification Service", Version: "1.0.0"})
	reg.AddEvent("notif-svc", newEvent("OrderCreated", "Order Created", catalog.Receives))

	cat := reg.Build()

	enriched := autoDeriveProducersConsumers(cat)

	if len(enriched.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(enriched.Services))
	}

	orderSvc := enriched.Services[0]
	if len(orderSvc.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(orderSvc.Events))
	}

	evt := orderSvc.Events[0]
	if len(evt.Producers) != 1 || evt.Producers[0] != "order-svc" {
		t.Errorf("expected producers [order-svc], got %v", evt.Producers)
	}

	notifSvc := enriched.Services[1]
	recvEvt := notifSvc.Events[0]
	if len(recvEvt.Consumers) != 1 || recvEvt.Consumers[0] != "notif-svc" {
		t.Errorf("expected consumers [notif-svc], got %v", recvEvt.Consumers)
	}
}

func TestExporter_Export_MessageWithProducersConsumers(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddEvent("order-svc", newEvent("OrderCreated", "Order Created", catalog.Sends))

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpDir, "services", "order-svc", "events", "OrderCreated", "index.mdx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event file: %v", err)
	}

	content := string(data)
	assertContains(t, content, "producers:")
	assertContains(t, content, "id: order-svc")
}

func TestExporter_Export_FullIntegration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("E-Commerce", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
		Summary:  "Manages orders",
		WritesTo: []catalog.DataStoreID{"orders-db"},
		Badges:   []catalog.Badge{{Content: "Production", BackgroundColor: "green"}},
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order",
		Version: "1.0.0", Summary: "Create a new order",
	})
	reg.AddEvent("order-svc", newEvent("OrderCreated", "Order Created", catalog.Sends))
	reg.AddDomain(catalog.Domain{
		ID: "orders", Name: "Orders", Version: "1.0.0",
		Services: []catalog.ServiceID{"order-svc"},
	})
	reg.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: "1.0.0",
		Protocols: []string{"kafka"},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders DB", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
	})
	reg.AddFlow(catalog.Flow{
		ID: "create-order", Name: "Create Order", Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{ID: "1", Title: "Create Order", Message: &catalog.FlowStepRef{ID: "CreateOrder"}},
		},
	})
	reg.AddTeam(catalog.Team{ID: "order-team", Name: "Order Team", Members: []string{"alice"}})
	reg.AddUser(catalog.User{ID: "alice", Name: "Alice", Role: "Engineer"})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{
		"services/order-svc/index.mdx",
		"services/order-svc/commands/CreateOrder/index.mdx",
		"services/order-svc/events/OrderCreated/index.mdx",
		"domains/orders/index.mdx",
		"channels/order-events/index.mdx",
		"data/orders-db/index.mdx",
		"flows/create-order/index.mdx",
		"teams/order-team.mdx",
		"users/alice.mdx",
		"eventcatalog.config.js",
		"package.json",
		"llms.txt",
	}

	for _, f := range expectedFiles {
		p := filepath.Join(tmpDir, f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", f)
		}
	}
}

func TestLLMsTxt_AllResourceTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
		Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order",
		Version: "1.0.0", Summary: "Create a new order",
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created",
		Version: "1.0.0", Direction: catalog.Sends, Summary: "Order was created",
	})
	reg.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: "1.0.0",
		Summary: "All order events", Protocols: []string{"kafka"},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders Database", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
		Summary: "Primary order store",
	})
	reg.AddFlow(catalog.Flow{
		ID: "create-order", Name: "Create Order Flow", Version: "1.0.0",
		Summary: "Order creation", Steps: []catalog.FlowStep{
			{ID: "1", Title: "Submit"},
			{ID: "2", Title: "Process"},
		},
	})
	reg.AddTeam(catalog.Team{
		ID: "order-team", Name: "Order Team", Summary: "Owns order domain",
		Members: []string{"alice", "bob"},
	})
	reg.AddUser(catalog.User{
		ID: "alice", Name: "Alice Smith", Role: "Senior Engineer",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	content := string(data)

	assertContains(t, content, "# TestCatalog")
	assertContains(t, content, "## Order Service (order-svc)")
	assertContains(t, content, "Manages orders")
	assertContains(t, content, "### Commands")
	assertContains(t, content, "Create Order")
	assertContains(t, content, "### Events")
	assertContains(t, content, "Order Created")
	assertContains(t, content, "## Channel: Order Events (order-events)")
	assertContains(t, content, "Protocols: kafka")
	assertContains(t, content, "## Data Store: Orders Database (orders-db)")
	assertContains(t, content, "Type: database")
	assertContains(t, content, "Technology: postgres@16")
	assertContains(t, content, "## Flow: Create Order Flow (create-order)")
	assertContains(t, content, "Steps: 2")
	assertContains(t, content, "## Team: Order Team (order-team)")
	assertContains(t, content, "Members: alice, bob")
	assertContains(t, content, "## User: Alice Smith (alice)")
	assertContains(t, content, "Role: Senior Engineer")
}

func assertContains(t *testing.T, content, substr string) {
	t.Helper()

	if !strings.Contains(content, substr) {
		t.Errorf("expected content to contain %q\nactual:\n%s", substr, content)
	}
}
