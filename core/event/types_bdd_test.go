package event_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

var _ = Describe("Version", func() {
	Describe("As a developer working with event versions", func() {
		Context("when I parse a valid version", func() {
			It("should return the version without error", func() {
				v, err := event.ParseVersion(5)
				Expect(err).ToNot(HaveOccurred())
				Expect(v.Int()).To(Equal(5))
			})
		})

		Context("when I parse a negative version", func() {
			It("should return an error", func() {
				_, err := event.ParseVersion(-1)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I parse zero", func() {
			It("should succeed and report IsZero", func() {
				v, err := event.ParseVersion(0)
				Expect(err).ToNot(HaveOccurred())
				Expect(v.IsZero()).To(BeTrue())
			})
		})

		Context("when I increment a version", func() {
			It("should return a new version without mutating the original", func() {
				v := event.Version(3)
				v2 := v.Increment()
				Expect(v2.Int()).To(Equal(4))
				Expect(v.Int()).To(Equal(3))
			})
		})

		Context("when I call IsPositive", func() {
			It("should be true for positive versions", func() {
				Expect(event.Version(1).IsPositive()).To(BeTrue())
			})

			It("should be false for zero", func() {
				Expect(event.Version(0).IsPositive()).To(BeFalse())
			})
		})

		Context("when I use arithmetic methods", func() {
			It("should add correctly", func() {
				Expect(event.Version(5).Add(3).Int()).To(Equal(8))
			})

			It("should subtract correctly", func() {
				Expect(event.Version(5).Sub(2).Int()).To(Equal(3))
			})

			It("should compute modulo correctly", func() {
				Expect(event.Version(7).Mod(3)).To(Equal(1))
			})
		})

		Context("when I compare versions with Cmp", func() {
			It("should return -1 when less than other", func() {
				Expect(event.Version(1).Cmp(event.Version(3))).To(Equal(-1))
			})

			It("should return 0 when equal", func() {
				Expect(event.Version(3).Cmp(event.Version(3))).To(Equal(0))
			})

			It("should return +1 when greater than other", func() {
				Expect(event.Version(5).Cmp(event.Version(3))).To(Equal(1))
			})
		})

		Context("when I convert to string", func() {
			It("should return the decimal representation", func() {
				Expect(event.Version(42).String()).To(Equal("42"))
			})
		})
	})
})

var _ = Describe("SchemaVersion", func() {
	Describe("As a developer managing event schema evolution", func() {
		Context("when I create a SchemaVersion", func() {
			It("should return the correct Int value", func() {
				sv := event.SchemaVersion(2)
				Expect(sv.Int()).To(Equal(2))
			})
		})

		Context("when I convert to string", func() {
			It("should return the decimal representation", func() {
				Expect(event.SchemaVersion(3).String()).To(Equal("3"))
			})
		})

		Context("when I check IsZero", func() {
			It("should be true for zero schema version", func() {
				Expect(event.SchemaVersion(0).IsZero()).To(BeTrue())
			})

			It("should be false for non-zero schema version", func() {
				Expect(event.SchemaVersion(1).IsZero()).To(BeFalse())
			})
		})
	})
})

var _ = Describe("CheckVersionConflict", func() {
	Describe("As a developer implementing optimistic concurrency", func() {
		Context("when existing length matches expected version", func() {
			It("should return nil", func() {
				err := event.CheckVersionConflict(3, event.Version(3))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when existing length does not match expected version", func() {
			It("should detect the version conflict and explain the mismatch", func() {
				err := event.CheckVersionConflict(2, event.Version(3))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version conflict"))
			})
		})

		Context("when starting fresh with zero events", func() {
			It("should succeed when expected version is zero", func() {
				err := event.CheckVersionConflict(0, event.Version(0))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
