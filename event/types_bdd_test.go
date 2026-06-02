package event_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

var _ = Describe("Version", func() {
	Describe("As a developer working with event versions", func() {
		Context("when I parse a valid version", func() {
			It("should accept it so I can use it for optimistic concurrency checks", func() {
				v, err := event.ParseVersion(5)
				Expect(err).ToNot(HaveOccurred())
				Expect(v.Int()).To(Equal(5))
			})
		})

		Context("when I parse a negative version", func() {
			It(
				"should reject it because negative versions are meaningless in event sourcing",
				func() {
					_, err := event.ParseVersion(-1)
					Expect(err).To(HaveOccurred())
				},
			)
		})

		Context("when I parse zero", func() {
			It(
				"should accept it as 'no events yet', letting me distinguish new aggregates",
				func() {
					v, err := event.ParseVersion(0)
					Expect(err).ToNot(HaveOccurred())
					Expect(v.IsZero()).To(BeTrue())
				},
			)
		})

		Context("when I increment a version", func() {
			It(
				"should return a new version without mutating the original, so I don't corrupt my state",
				func() {
					v := event.Version(3)
					v2 := v.Increment()
					Expect(v2.Int()).To(Equal(4))
					Expect(v.Int()).To(Equal(3))
				},
			)
		})

		Context("when I decrement a version", func() {
			It("should return the previous version", func() {
				v := event.Version(3)
				Expect(v.Decrement().Int()).To(Equal(2))
			})

			It("should not mutate the original", func() {
				v := event.Version(3)
				_ = v.Decrement()
				Expect(v.Int()).To(Equal(3))
			})
		})

		Context("when I call IsPositive", func() {
			It("should be true for positive versions, telling me the aggregate has events", func() {
				Expect(event.Version(1).IsPositive()).To(BeTrue())
			})

			It("should be false for zero, telling me the aggregate is new", func() {
				Expect(event.Version(0).IsPositive()).To(BeFalse())
			})
		})

		Context("when I use arithmetic methods", func() {
			It("should add correctly so I can project ahead in my snapshot strategy", func() {
				Expect(event.Version(5).Add(3).Int()).To(Equal(8))
			})

			It("should subtract correctly so I can compute how many events to replay", func() {
				Expect(event.Version(5).Sub(2).Int()).To(Equal(3))
			})

			It(
				"should compute modulo correctly so I can implement every-N-events snapshotting",
				func() {
					Expect(event.Version(7).Mod(3)).To(Equal(1))
				},
			)
		})

		Context("when I compare versions with Cmp", func() {
			DescribeTable(
				"comparisons",
				func(v1, v2 event.Version, expected int) {
					Expect(v1.Cmp(v2)).To(Equal(expected))
				},
				Entry(
					"should return -1 when less than other, helping me detect version drift",
					event.Version(1),
					event.Version(3),
					-1,
				),
				Entry(
					"should return 0 when equal, confirming my optimistic concurrency check passes",
					event.Version(3),
					event.Version(3),
					0,
				),
				Entry(
					"should return +1 when greater than other, telling me I'm ahead of the store",
					event.Version(5),
					event.Version(3),
					1,
				),
			)
		})

		Context("when I convert to string", func() {
			It("should return the decimal representation for logging and debugging", func() {
				Expect(event.Version(42).String()).To(Equal("42"))
			})
		})
	})
})

var _ = Describe("SchemaVersion", func() {
	Describe("As a developer managing event schema evolution", func() {
		Context("when I create a SchemaVersion", func() {
			It(
				"should hold the version so I can track which schema migration each event uses",
				func() {
					sv := event.SchemaVersion(2)
					Expect(sv.Int()).To(Equal(2))
				},
			)
		})

		Context("when I convert to string", func() {
			It("should return the decimal representation for my migration logs", func() {
				Expect(event.SchemaVersion(3).String()).To(Equal("3"))
			})
		})

		Context("when I check IsZero", func() {
			It(
				"should be true for zero, meaning no schema version was specified (legacy events)",
				func() {
					Expect(event.SchemaVersion(0).IsZero()).To(BeTrue())
				},
			)

			It(
				"should be false for non-zero, meaning the event has explicit schema tracking",
				func() {
					Expect(event.SchemaVersion(1).IsZero()).To(BeFalse())
				},
			)
		})

		Context("when I parse a schema version", func() {
			It("should accept valid positive values", func() {
				sv, err := event.ParseSchemaVersion(2)
				Expect(err).ToNot(HaveOccurred())
				Expect(sv.Int()).To(Equal(2))
			})

			It("should reject negative values", func() {
				_, err := event.ParseSchemaVersion(-1)
				Expect(err).To(HaveOccurred())
			})

			It("should reject zero — schema versioning starts at 1", func() {
				_, err := event.ParseSchemaVersion(0)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I decrement a schema version", func() {
			It("should return the previous version for rollback handling", func() {
				sv := event.SchemaVersion(3)
				Expect(sv.Decrement().Int()).To(Equal(2))
			})

			It("should not mutate the original value", func() {
				sv := event.SchemaVersion(3)
				_ = sv.Decrement()
				Expect(sv.Int()).To(Equal(3))
			})
		})
	})
})

var _ = Describe("CheckVersionConflict", func() {
	Describe("As a developer implementing optimistic concurrency", func() {
		DescribeTable(
			"version matches succeed without error",
			func(existing int, expected event.Version) {
				err := event.CheckVersionConflict(existing, expected)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("existing length matches expected version", 3, event.Version(3)),
			Entry("starting fresh with zero events", 0, event.Version(0)),
		)

		Context("when existing length does not match expected version", func() {
			It("should detect the version conflict and explain the mismatch", func() {
				err := event.CheckVersionConflict(2, event.Version(3))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version conflict"))
			})
		})
	})
})
