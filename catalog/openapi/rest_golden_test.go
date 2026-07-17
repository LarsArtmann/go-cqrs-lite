package openapi

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// TestExporter_RestGolden verifies the full REST feature set renders correctly:
// MsgOperation (method/path/statusCodes), typed Responses, Parameters, Security.
func TestExporter_RestGolden(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("REST API", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "user-svc",
		Name:    "User Service",
		Version: "1.0.0",
		Summary: "Manages users",
		Commands: []catalog.Message{
			{
				ID:      "user.create",
				Name:    "CreateUser",
				Version: "1.0.0",
				Kind:    catalog.CommandMessage,
				Operation: &catalog.Operation{
					Method:      "POST",
					Path:        "/api/v2/users",
					StatusCodes: []string{"201", "400"},
				},
				Security: []string{"bearerAuth"},
				Responses: []catalog.ResponseSpec{
					{
						StatusCode:  "201",
						Description: "User created",
						Schema: &catalog.Schema{
							Type: catalog.TypeObject,
							Properties: map[string]catalog.Property{
								"id":    {Type: catalog.TypeString},
								"email": {Type: catalog.TypeString},
							},
						},
					},
				},
			},
		},
		Queries: []catalog.Message{
			{
				ID:      "user.list",
				Name:    "ListUsers",
				Version: "1.0.0",
				Kind:    catalog.QueryMessage,
				Operation: &catalog.Operation{
					Method: "GET",
					Path:   "/api/v2/users",
				},
				Security: []string{"bearerAuth"},
				Schema: &catalog.Schema{
					Type: catalog.TypeObject,
					Parameters: []catalog.Parameter{
						{Name: "limit", In: "query", Description: "Max results per page"},
						{Name: "offset", In: "query", Description: "Pagination offset"},
					},
				},
			},
		},
	})
	reg.AddSecurityScheme(catalog.SecurityScheme{
		ID:           "bearerAuth",
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT Bearer token authentication",
	})

	cat := reg.Build()
	doc := NewExporter("REST API", "1.0.0").Export(cat)

	// Verify security schemes in components
	if doc.Components.SecuritySchemes == nil {
		t.Fatal("expected securitySchemes in components")
	}

	ss, ok := doc.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("expected bearerAuth security scheme")
	}

	if ss.Type != "http" || ss.Scheme != "bearer" {
		t.Errorf("expected http/bearer, got %s/%s", ss.Type, ss.Scheme)
	}

	// Verify POST path
	postItem, ok := doc.Paths["/api/v2/users"]
	if !ok {
		t.Fatalf("expected path /api/v2/users, got paths: %v", pathKeys(doc))
	}

	if postItem.Post == nil {
		t.Fatal("expected POST operation")
	}

	if len(postItem.Post.Security) == 0 {
		t.Error("expected security on POST operation")
	}

	if _, ok := postItem.Post.Responses["201"]; !ok {
		t.Error("expected 201 response")
	}

	if _, ok := doc.Components.Schemas["command.user.create.response.201"]; !ok {
		t.Error("expected response schema in components")
	}

	// Verify GET path (should merge into same PathItem)
	if postItem.Get == nil {
		t.Fatal("expected GET operation to coexist with POST on same path")
	}

	if len(postItem.Get.Parameters) != 2 {
		t.Errorf("expected 2 query parameters, got %d", len(postItem.Get.Parameters))
	}

	// Verify JSON serializes correctly
	raw, err := json.Marshal(doc, json.Deterministic(true))
	if err != nil {
		t.Fatalf("failed to marshal document: %v", err)
	}

	jsonStr := string(raw)

	if !strings.Contains(jsonStr, "bearerAuth") {
		t.Error("expected bearerAuth in JSON output")
	}

	if !strings.Contains(jsonStr, "/api/v2/users") {
		t.Error("expected /api/v2/users path in JSON output")
	}

	if !strings.Contains(jsonStr, "command.user.create.response.201") {
		t.Error("expected response schema ref in JSON output")
	}
}

func pathKeys(doc *Document) []string {
	keys := make([]string, 0, len(doc.Paths))
	for k := range doc.Paths {
		keys = append(keys, k)
	}
	return keys
}
