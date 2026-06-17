package asyncapi_test

import (
	"testing"

	"github.com/go-faster/yaml"
	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi"
)

func TestDocumentYAMLRoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)

	original := asyncapi.Document{
		AsyncAPI: "2.6.0",
		Info: catalog.DocumentInfo{
			Title:       "Test Event Catalog",
			Version:     "1.0.0",
			Description: "Round-trip test",
		},
		Channels: map[string]asyncapi.Channel{
			"user.created": {
				Address: "user.created",
				Title:   "User Created",
				Messages: map[string]asyncapi.Ref{
					"message": {Ref: "#/components/messages/user.created"},
				},
			},
		},
		Components: asyncapi.Components{
			Messages: map[string]asyncapi.Message{
				"user.created": {
					Name:        "user.created",
					Title:       "User Created",
					ContentType: "application/json",
					Payload:     asyncapi.Ref{Ref: "#/components/schemas/UserCreated"},
				},
			},
		},
	}

	data, err := original.MarshalYAML()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(data).NotTo(gomega.BeEmpty())

	var decoded asyncapi.Document
	err = yaml.Unmarshal(data, &decoded)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(decoded.AsyncAPI).To(gomega.Equal(original.AsyncAPI))
	g.Expect(decoded.Info.Title).To(gomega.Equal(original.Info.Title))
	g.Expect(decoded.Info.Version).To(gomega.Equal(original.Info.Version))
	g.Expect(decoded.Channels).To(gomega.HaveKey("user.created"))
}
