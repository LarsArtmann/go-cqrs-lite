package yaml

import (
	"strconv"
	"strings"
	"testing"
)

func TestMarshal_Primitives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"empty string", "", `""`},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := Marshal(tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if got := strings.TrimSpace(string(b)); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMarshal_Bool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input bool
		want  string
	}{
		{true, "true"},
		{false, "false"},
	}
	for _, tc := range tests {
		t.Run(strconv.FormatBool(tc.input), marshalTestFunc(tc.input, tc.want, "%v"))
	}
}

func TestMarshal_SpecialStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"true", `"true"`},
		{"false", `"false"`},
		{"null", `"null"`},
		{"hello: world", `"hello: world"`},
		{"has spaces", `"has spaces"`},
		{"a & b", `"a \u0026 b"`},
		{"#comment", `"#comment"`},
	}
	for _, tc := range tests {
		t.Run(tc.input, marshalTestFunc(tc.input, tc.want, "%q"))
	}
}

// marshalTestFunc returns a test function that marshals an input and checks the result.
func marshalTestFunc(input any, want, format string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		b, err := Marshal(input)
		if err != nil {
			t.Fatal(err)
		}

		if got := strings.TrimSpace(string(b)); got != want {
			t.Errorf("Marshal("+format+") = %q, want %q", input, got, want)
		}
	}
}

func assertTrimmedEq(t *testing.T, b []byte, want string) {
	t.Helper()

	if got := strings.TrimSpace(string(b)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarshal_Nil(t *testing.T) {
	t.Parallel()

	var ptr *string

	b, err := Marshal(ptr)
	if err != nil {
		t.Fatal(err)
	}

	assertTrimmedEq(t, b, "null")
}

func TestMarshal_Slice(t *testing.T) {
	t.Parallel()

	b, err := Marshal([]string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}

	want := "- a\n- b\n- c\n"
	if got := string(b); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestMarshal_EmptySlice(t *testing.T) {
	t.Parallel()

	b, err := Marshal([]string{})
	if err != nil {
		t.Fatal(err)
	}

	assertTrimmedEq(t, b, "[]")
}

func TestMarshal_Map(t *testing.T) {
	t.Parallel()

	b, err := Marshal(map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))

	want := "key: value"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarshal_EmptyMap(t *testing.T) {
	t.Parallel()

	b, err := Marshal(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	assertTrimmedEq(t, b, "{}")
}

func TestMarshal_SortedMap(t *testing.T) {
	t.Parallel()

	m := map[string]int{"z": 1, "a": 2, "m": 3}

	b, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	if !strings.Contains(got, "a: 2") || !strings.Contains(got, "m: 3") ||
		!strings.Contains(got, "z: 1") {
		t.Errorf("map keys not sorted:\n%s", got)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if lines[0] != "a: 2" {
		t.Errorf("first key should be 'a', got %q", lines[0])
	}
}

type testStruct struct {
	Name  string `yaml:"name"`
	Age   int    `yaml:"age"`
	Email string `json:"email"`
}

func TestMarshal_Struct(t *testing.T) {
	t.Parallel()

	s := testStruct{Name: "Alice", Age: 30, Email: "alice@example.com"}

	b, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	if !strings.Contains(got, "name: Alice") {
		t.Errorf("missing name field in:\n%s", got)
	}

	if !strings.Contains(got, "age: 30") {
		t.Errorf("missing age field in:\n%s", got)
	}

	if !strings.Contains(got, "email:") {
		t.Errorf("missing email field (json tag fallback) in:\n%s", got)
	}
}

func TestMarshal_StructYamlTag(t *testing.T) {
	t.Parallel()

	s := testStruct{Name: "Bob", Age: 25}

	b, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	if !strings.Contains(got, "name: Bob") {
		t.Errorf("yaml tag not used:\n%s", got)
	}
}

func TestMarshal_NestedStruct(t *testing.T) {
	t.Parallel()

	type inner struct {
		Value string `yaml:"value"`
	}

	type outer struct {
		Name  string `yaml:"name"`
		Inner inner  `yaml:"inner"`
	}

	o := outer{Name: "test", Inner: inner{Value: "nested"}}

	b, err := Marshal(o)
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	if !strings.Contains(got, "name: test") {
		t.Errorf("missing name in:\n%s", got)
	}

	if !strings.Contains(got, "inner:") {
		t.Errorf("missing inner in:\n%s", got)
	}

	if !strings.Contains(got, "value: nested") {
		t.Errorf("missing nested value in:\n%s", got)
	}
}

func TestMarshal_SliceOfStructs(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `yaml:"name"`
	}

	items := []item{{Name: "first"}, {Name: "second"}}

	b, err := Marshal(items)
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	if !strings.Contains(got, "name: first") || !strings.Contains(got, "name: second") {
		t.Errorf("missing items in:\n%s", got)
	}
}
