package metaengine

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestMaterializedViewSpec_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    MaterializedViewSpec
		wantErr bool
	}{
		{
			name: "valid scalar SUM",
			spec: MaterializedViewSpec{Collection: "orders", Fn: MatViewSum, Column: "amount"},
		},
		{
			name: "valid grouped AVG",
			spec: MaterializedViewSpec{Collection: "orders", Fn: MatViewAvg, Column: "amount", GroupBy: "customer"},
		},
		{
			name: "valid scalar COUNT (no column)",
			spec: MaterializedViewSpec{Collection: "orders", Fn: MatViewCount},
		},
		{
			name:    "COUNT must not take a column",
			spec:    MaterializedViewSpec{Collection: "orders", Fn: MatViewCount, Column: "amount"},
			wantErr: true,
		},
		{
			name:    "SUM requires a column",
			spec:    MaterializedViewSpec{Collection: "orders", Fn: MatViewSum},
			wantErr: true,
		},
		{
			name:    "empty collection",
			spec:    MaterializedViewSpec{Fn: MatViewSum, Column: "amount"},
			wantErr: true,
		},
		{
			name:    "unknown fn",
			spec:    MaterializedViewSpec{Collection: "orders", Fn: "MEDIAN", Column: "amount"},
			wantErr: true,
		},
		{
			name:    "quote injection in collection",
			spec:    MaterializedViewSpec{Collection: "orders'; DROP TABLE meta_map; --", Fn: MatViewSum, Column: "amount"},
			wantErr: true,
		},
		{
			name:    "semicolon in groupBy",
			spec:    MaterializedViewSpec{Collection: "orders", Fn: MatViewSum, Column: "amount", GroupBy: "a;b"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaterializedViewSpec_ViewName(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	spec := MaterializedViewSpec{Collection: "orders", Fn: MatViewSum, Column: "amount", GroupBy: "customer"}
	name := spec.ViewName()

	g.Expect(name).To(gomega.HavePrefix("cqrs_mv_orders_sum_amount_by_customer"))
	g.Expect(name).To(gomega.Equal(spec.ViewName()), "deterministic across calls")

	other := spec
	other.Fn = MatViewAvg
	g.Expect(other.ViewName()).To(gomega.Not(gomega.Equal(name)), "different shapes never collide")

	long := MaterializedViewSpec{
		Collection: "extremely_long_collection_name_that_goes_on_and_on_forever",
		Fn:         MatViewSum,
		Column:     "another_extremely_long_column_name_for_good_measure",
	}
	g.Expect(len(long.ViewName())).To(gomega.BeNumerically("<=", 64))

	weird := MaterializedViewSpec{Collection: "my collection", Fn: MatViewMin, Column: "x y"}
	g.Expect(weird.ViewName()).To(gomega.Not(gomega.ContainSubstring(" ")))
	g.Expect(strings.ContainsAny(weird.ViewName(), `";'`)).To(gomega.BeFalse())
}
