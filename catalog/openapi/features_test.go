package openapi

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestExporter_WithOpenAPIVersion(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Svc", Version: "1.0.0",
		Commands: []catalog.Message{
			{ID: "do-thing", Name: "DoThing", Version: "1.0.0", Kind: catalog.CommandMessage},
		},
	})

	cat := reg.Build()
	doc := NewExporter("Svc", "1.0.0", WithOpenAPIVersion("3.1.0")).Export(cat)

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want 3.1.0", doc.OpenAPI)
	}

	if def := NewExporter("Svc", "1.0.0").Export(cat).OpenAPI; def != "3.0.3" {
		t.Errorf("default OpenAPI version = %q, want 3.0.3", def)
	}
}

func TestExporter_ResponseHeaders(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Svc", Version: "1.0.0",
		Commands: []catalog.Message{{
			ID: "send-message", Name: "SendMessage", Version: "1.0.0",
			Kind: catalog.CommandMessage,
			Responses: []catalog.ResponseSpec{
				{
					StatusCode:  "429",
					Description: "Rate limited",
					Headers: []catalog.ResponseHeader{
						{Name: "Retry-After", Description: "Seconds until retry"},
					},
				},
			},
		}},
	})

	doc := NewExporter("Svc", "1.0.0").Export(reg.Build())

	var rateLimited *Response

	for _, pi := range doc.Paths {
		if pi.Post == nil {
			continue
		}

		if resp, ok := pi.Post.Responses["429"]; ok {
			rateLimited = resp
		}
	}

	if rateLimited == nil {
		t.Fatal("429 response missing from exported operations")
	}

	hdr, ok := rateLimited.Headers["Retry-After"]
	if !ok {
		t.Fatalf("Retry-After header missing, headers = %v", rateLimited.Headers)
	}

	if hdr.Description != "Seconds until retry" {
		t.Errorf("Retry-After description = %q", hdr.Description)
	}

	schema, ok := hdr.Schema.(map[string]any)
	if !ok || schema["type"] != "string" {
		t.Errorf("Retry-After schema = %v, want default untyped string", hdr.Schema)
	}
}

func TestExporter_MultipartRequestBody(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Svc", Version: "1.0.0",
		Commands: []catalog.Message{{
			ID: "upload", Name: "Upload", Version: "1.0.0",
			Kind:                 catalog.CommandMessage,
			RequestBodyMediaType: "multipart/form-data",
			RequestBodySchema: &catalog.Schema{
				Type: catalog.TypeObject,
				Properties: map[string]catalog.Property{
					"file": {Type: "string", Format: "binary"},
				},
			},
		}},
	})

	doc := NewExporter("Svc", "1.0.0").Export(reg.Build())

	var body *RequestBody

	for _, pi := range doc.Paths {
		if pi.Post != nil {
			body = pi.Post.RequestBody
		}
	}

	if body == nil {
		t.Fatal("request body missing")
	}

	mt, ok := body.Content["multipart/form-data"]
	if !ok {
		t.Fatalf("multipart content missing, content = %v", body.Content)
	}

	if _, ok := body.Content["application/json"]; ok {
		t.Error("JSON content must not be emitted alongside the declared media type")
	}

	if mt.Schema == nil {
		t.Fatal("multipart schema missing")
	}
}

func TestExporter_TypedParamSchema(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Svc", Version: "1.0.0",
		Queries: []catalog.Message{{
			ID: "list", Name: "List", Version: "1.0.0",
			Kind: catalog.QueryMessage,
			Schema: &catalog.Schema{
				Type: catalog.TypeObject,
				Parameters: []catalog.Parameter{{
					Name:     "limit",
					In:       "query",
					Required: false,
					Schema: &catalog.Property{
						Type: catalog.TypeInteger,
						Enum: []string{"10", "50", "100"},
					},
				}},
			},
		}},
	})

	doc := NewExporter("Svc", "1.0.0").Export(reg.Build())

	for _, pi := range doc.Paths {
		if pi.Get == nil {
			continue
		}

		for _, p := range pi.Get.Parameters {
			if p.Name != "limit" {
				continue
			}

			schema, ok := p.Schema.(map[string]any)
			if !ok {
				t.Fatalf("limit schema = %T, want map", p.Schema)
			}

			if schema["type"] != "integer" {
				t.Errorf("limit type = %v, want integer", schema["type"])
			}

			if _, hasEnum := schema["enum"]; !hasEnum {
				t.Errorf("limit enum missing, schema = %v", schema)
			}

			return
		}
	}

	t.Fatal("limit parameter missing from exported operation")
}
