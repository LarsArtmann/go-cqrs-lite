package schema

import "testing"

type stringBrand struct{ value string }

func (stringBrand) MarshalText() ([]byte, error) { return []byte("id"), nil }

type intBrand struct{ value int64 }

func (intBrand) MarshalText() ([]byte, error) { return []byte("1"), nil }

type twoFieldText struct{ left, right string }

func (twoFieldText) MarshalText() ([]byte, error) { return nil, nil }

func TestFromType_TextMarshalerStructsTakeTheirWireType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *Schema
		want Type
	}{
		{name: "string-backed brand", s: FromType[stringBrand](), want: TypeString},
		{name: "int-backed brand", s: FromType[intBrand](), want: TypeInteger},
		{name: "multi-field text type stays an object", s: FromType[twoFieldText](), want: TypeObject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.s.Type != tt.want {
				t.Errorf("schema type = %q, want %q: %+v", tt.s.Type, tt.want, tt.s)
			}
		})
	}
}

func TestFromType_BrandedIDFieldIsAStringProperty(t *testing.T) {
	t.Parallel()

	type payload struct {
		ID    stringBrand
		Count intBrand
		When  twoFieldText
	}

	s := FromType[payload]()

	for field, want := range map[string]Type{"ID": TypeString, "Count": TypeInteger, "When": TypeObject} {
		if got := s.Properties[field].Type; got != want {
			t.Errorf("property %s = %q, want %q", field, got, want)
		}
	}
}
