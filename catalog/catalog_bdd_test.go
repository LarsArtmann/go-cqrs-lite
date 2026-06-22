package catalog_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

// Fixture types for generic message constructors.
// Names are chosen to test auto-derivation: CreateUserCmd → "Create User".

type CreateUserCmd struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserCreatedEvent struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

type GetUserQuery struct {
	ID string `json:"id"`
}

var _ = Describe("Catalog Builder", func() {
	Describe("As a developer documenting my CQRS system", func() {
		Describe("building a catalog from Go structs", func() {
			var cat *catalog.Catalog

			BeforeEach(func() {
				b := catalog.NewBuilder("E-Commerce Platform", "1.0.0")
				b.AddService(
					"user-svc", "User Service", "2.0.0", "Manages users",
					catalog.Command[CreateUserCmd](
						"createUser",
						catalog.WithSummary("Create a new user account"),
						catalog.Owners("team-identity"),
					),
					catalog.Event[UserCreatedEvent]("userCreated", catalog.Sends),
					catalog.Query[GetUserQuery]("getUser"),
				)
				cat = b.Build()
			})

			It("should include the service in the catalog", func() {
				Expect(cat.Services).To(HaveLen(1))
				Expect(string(cat.Services[0].ID)).To(Equal("user-svc"))
			})

			It("should auto-derive the command name from the Go type", func() {
				cmds := cat.Services[0].Commands
				Expect(cmds).To(HaveLen(1))
				Expect(string(cmds[0].Name)).To(Equal("Create User"))
			})

			It("should auto-derive the event name from the Go type", func() {
				events := cat.Services[0].Events
				Expect(events).To(HaveLen(1))
				Expect(string(events[0].Name)).To(Equal("User Created"))
			})

			It("should auto-derive the query name from the Go type", func() {
				queries := cat.Services[0].Queries
				Expect(queries).To(HaveLen(1))
				Expect(string(queries[0].Name)).To(Equal("Get User"))
			})

			It("should derive JSON schema from struct tags", func() {
				cmds := cat.Services[0].Commands
				Expect(cmds[0].Schema).NotTo(BeNil())
				Expect(string(cmds[0].Schema.Type)).To(Equal("object"))
			})

			It("should apply message options", func() {
				cmds := cat.Services[0].Commands
				Expect(string(cmds[0].Summary)).To(Equal("Create a new user account"))
				Expect(cmds[0].Owners).To(ContainElement("team-identity"))
			})

			It("should set default direction for commands", func() {
				cmds := cat.Services[0].Commands
				Expect(cmds[0].Direction).To(Equal(catalog.Receives))
			})

			It("should respect explicit direction for events", func() {
				events := cat.Services[0].Events
				Expect(events[0].Direction).To(Equal(catalog.Sends))
			})
		})

		Describe("auto-naming edge cases", func() {
			It("should handle single-word type names", func() {
				type Activate struct{ Active bool }

				b := catalog.NewBuilder("Test", "1.0.0")
				b.AddService(
					"svc", "Svc", "1.0.0", "",
					catalog.Command[Activate]("activate"),
				)
				cat := b.Build()

				Expect(string(cat.Services[0].Commands[0].Name)).To(Equal("Activate"))
			})

			It("should preserve type names that are already short", func() {
				type X struct{}

				b := catalog.NewBuilder("Test", "1.0.0")
				b.AddService(
					"svc", "Svc", "1.0.0", "",
					catalog.Command[X]("x"),
				)
				cat := b.Build()

				Expect(string(cat.Services[0].Commands[0].Name)).To(Equal("X"))
			})
		})

		Describe("merging messages from multiple AddService calls", func() {
			It("should accumulate messages when the same service is added twice", func() {
				b := catalog.NewBuilder("Platform", "1.0.0")
				b.AddService(
					"order-svc", "Order Service", "1.0.0", "",
					catalog.Command[CreateUserCmd]("cmd1"),
				)
				b.AddService(
					"order-svc", "Order Service", "1.0.0", "",
					catalog.Event[UserCreatedEvent]("evt1", catalog.Sends),
				)

				cat := b.Build()

				Expect(cat.Services).To(HaveLen(1))
				Expect(cat.Services[0].Commands).To(HaveLen(1))
				Expect(cat.Services[0].Events).To(HaveLen(1))
			})
		})
	})
})

var _ = Describe("Registry", func() {
	Describe("as a developer using the low-level Registry API", func() {
		Describe("adding messages incrementally", func() {
			It("should store commands, events, and queries under the right service", func() {
				reg := catalog.NewRegistry("Platform", "1.0.0")
				reg.SetServiceMeta("svc", "My Service", "1.0.0", "Desc")

				reg.AddCommand("svc", catalog.Message{
					Kind: catalog.CommandMessage, ID: "cmd1", Name: "Do Thing",
				})
				reg.AddEvent("svc", catalog.Message{
					Kind: catalog.EventMessage, ID: "evt1", Name: "Thing Done",
				})
				reg.AddQuery("svc", catalog.Message{
					Kind: catalog.QueryMessage, ID: "qry1", Name: "Get Thing",
				})

				cat := reg.Build()

				Expect(cat.Services).To(HaveLen(1))
				Expect(cat.Services[0].Commands).To(HaveLen(1))
				Expect(cat.Services[0].Events).To(HaveLen(1))
				Expect(cat.Services[0].Queries).To(HaveLen(1))
			})

			It("should return an immutable snapshot from Build", func() {
				reg := catalog.NewRegistry("Platform", "1.0.0")
				reg.SetServiceMeta("svc", "Svc", "1.0.0", "")
				reg.AddCommand("svc", catalog.Message{
					Kind: catalog.CommandMessage, ID: "cmd1", Name: "Cmd",
				})

				cat1 := reg.Build()
				originalCmdCount := len(cat1.Services[0].Commands)

				reg.AddCommand("svc", catalog.Message{
					Kind: catalog.CommandMessage, ID: "cmd2", Name: "Cmd2",
				})
				cat2 := reg.Build()

				Expect(cat1.Services[0].Commands).To(HaveLen(originalCmdCount),
					"first snapshot should be unaffected by later additions")
				Expect(cat2.Services[0].Commands).To(HaveLen(originalCmdCount+1),
					"second snapshot should reflect the addition")
			})
		})
	})
})

var _ = Describe("Catalog Validation", func() {
	Describe("as a developer validating my catalog", func() {
		It("should pass for a well-formed catalog", func() {
			b := catalog.NewBuilder("Platform", "1.0.0")
			b.AddService(
				"svc", "Svc", "1.0.0", "",
				catalog.Command[CreateUserCmd]("cmd1"),
			)
			cat := b.Build()

			violations := cat.Validate()
			Expect(violations).To(BeEmpty())
		})

		It("should flag duplicate message IDs within a service", func() {
			reg := catalog.NewRegistry("Platform", "1.0.0")
			reg.SetServiceMeta("svc", "Svc", "1.0.0", "")
			reg.AddCommand("svc", catalog.Message{
				Kind: catalog.CommandMessage, ID: "dup", Name: "First",
			})
			reg.AddCommand("svc", catalog.Message{
				Kind: catalog.CommandMessage, ID: "dup", Name: "Second",
			})

			cat := reg.Build()
			violations := cat.Validate()
			Expect(violations).NotTo(BeEmpty())
		})
	})
})

var _ = Describe("Catalog JSON Serialization", func() {
	Describe("as a developer serializing the catalog to JSON", func() {
		It("should round-trip through marshal/unmarshal", func() {
			b := catalog.NewBuilder("Platform", "1.0.0")
			b.AddService(
				"svc", "Svc", "1.0.0", "",
				catalog.Command[CreateUserCmd]("cmd1"),
				catalog.Event[UserCreatedEvent]("evt1", catalog.Sends),
			)
			original := b.Build()

			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var restored catalog.Catalog
			err = json.Unmarshal(data, &restored)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(restored.Title)).To(Equal("Platform"))
			Expect(restored.Services).To(HaveLen(1))
			Expect(restored.Services[0].Commands).To(HaveLen(1))
			Expect(restored.Services[0].Events).To(HaveLen(1))
		})
	})
})
