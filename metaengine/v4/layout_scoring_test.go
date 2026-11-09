package metaengine_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Layout scoring (ADR-0124 Layer 4)", func() {
	Describe("SelectLayout", func() {
		It("selects Embed for KV engine with ReadSpeed", func() {
			kvProfile := metaengine.EngineProfile{
				Name: "pebble",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}

			option, cost := metaengine.SelectLayout(kvProfile, metaengine.PriorityReadSpeed)
			Expect(option).To(Equal(metaengine.LayoutEmbed))
			Expect(cost.ReadCost).To(BeNumerically("<", 1.0))
		})

		It("selects Normalize for KV engine with WriteSpeed", func() {
			kvProfile := metaengine.EngineProfile{
				Name: "pebble",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}

			option, _ := metaengine.SelectLayout(kvProfile, metaengine.PriorityWriteSpeed)
			Expect(option).To(Equal(metaengine.LayoutNormalize))
		})

		It("selects Normalize for SQL engine with ReadSpeed", func() {
			sqlProfile := metaengine.EngineProfile{
				Name: "postgres",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutRow,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityOLogN,
				},
			}

			option, _ := metaengine.SelectLayout(sqlProfile, metaengine.PriorityReadSpeed)
			Expect(option).To(Equal(metaengine.LayoutNormalize))
		})

		It(
			"selects Embed for KV engine with StorageSpace is NOT selected (KV embed is cheaper on storage for small aggregates)",
			func() {
				// StorageSpace penalizes duplication. Embed has higher storage cost.
				// So Normalize should be selected for StorageSpace.
				kvProfile := metaengine.EngineProfile{
					Name: "pebble",
					Layouts: map[metaengine.ADT]metaengine.StorageLayout{
						metaengine.ADTMap: metaengine.LayoutKV,
					},
					Supports: map[metaengine.ADT]metaengine.Complexity{
						metaengine.ADTMap: metaengine.ComplexityO1,
					},
				}

				option, _ := metaengine.SelectLayout(kvProfile, metaengine.PriorityStorageSpace)
				Expect(option).To(Equal(metaengine.LayoutNormalize))
			},
		)
	})

	Describe("LayoutCost.ScoreWeighted", func() {
		It("applies priority weights correctly", func() {
			cost := metaengine.LayoutCost{
				ReadCost:    2.0,
				WriteCost:   1.0,
				StorageCost: 1.0,
			}

			readScore := cost.ScoreWeighted(metaengine.PriorityReadSpeed.Weights())
			balancedScore := cost.ScoreWeighted(metaengine.PriorityBalanced.Weights())

			// ReadSpeed should penalize the high read cost more
			Expect(readScore).To(BeNumerically(">", balancedScore))
		})
	})

	Describe("ScoreLayouts", func() {
		It("returns both embed and normalize options", func() {
			profile := metaengine.EngineProfile{
				Name: "test",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}

			costs := metaengine.ScoreLayouts(profile)
			Expect(costs).To(HaveLen(2))
			options := []metaengine.LayoutOption{costs[0].Option, costs[1].Option}
			Expect(options).To(ContainElement(metaengine.LayoutEmbed))
			Expect(options).To(ContainElement(metaengine.LayoutNormalize))
		})
	})
})
