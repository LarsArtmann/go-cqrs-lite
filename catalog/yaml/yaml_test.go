package yaml

import (
	"strings"
	"testing"
)

func TestMarshal_String(t *testing.T) {
	t.Parallel()
	b, err := Marshal("hello")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestMarshal_EmptyString(t *testing.T) {
	t.Parallel()
	b, err := Marshal("")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != `""` {
		t.Errorf("got %q, want %q", got, `""`)
	}
}

func TestMarshal_Int(t *testing.T) {
	t.Parallel()
	b, err := Marshal(42)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestMarshal_Float(t *testing.T) {
	t.Parallel()
	b, err := Marshal(3.14)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "3.14" {
		t.Errorf("got %q, want %q", got, "3.14")
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
	for _, tt := range tests {
		b, err := Marshal(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(b)); got != tt.want {
			t.Errorf("Marshal(%v) = %q, want %q", tt.input, got, tt.want)
		}
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
	for _, tt := range tests {
		b, err := Marshal(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(b)); got != tt.want {
			t.Errorf("Marshal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarshal_Nil(t *testing.T) {
	t.Parallel()
	var ptr *string
	b, err := Marshal(ptr)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "null" {
		t.Errorf("got %q, want %q", got, "null")
	}
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
	if got := strings.TrimSpace(string(b)); got != "[]" {
		t.Errorf("got %q, want %q", got, "[]")
	}
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
	if got := strings.TrimSpace(string(b)); got != "{}" {
		t.Errorf("got %q, want %q", got, "{}")
	}
}

func TestMarshal_SortedMap(t *testing.T) {
	t.Parallel()
	m := map[string]int{"z": 1, "a": 2, "m": 3}
	b, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "a: 2") || !strings.Contains(got, "m: 3") || !strings.Contains(got, "z: 1") {
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
