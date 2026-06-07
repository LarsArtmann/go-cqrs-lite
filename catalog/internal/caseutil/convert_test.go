package caseutil

import "testing"

func TestToSeparated(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"HelloWorld", "hello.world"},
		{"GetUserID", "get.user.id"},
		{"XMLParser", "xml.parser"},
		{"simple", "simple"},
		{"", ""},
		{"GetUser2", "get.user.2"},
		{"V2API", "v.2api"},
		{"hello world", "hello.world"},
		{"hello_world", "hello.world"},
		{"ABCD", "abcd"},
		{"Réservé", "rserv"},
		{"A", "a"},
		{"already-lower", "already-lower"},
	}

	for _, tt := range tests {
		if got := ToSeparated(tt.input, '.'); got != tt.want {
			t.Errorf("ToSeparated(%q, '.') = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDotSeparated(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"CreateUser", "create.user"},
		{"UserCreated", "user.created"},
		{"GetUserByID", "get.user.by.id"},
		{"HTTPServer", "http.server"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		if got := DotSeparated(tt.input); got != tt.want {
			t.Errorf("DotSeparated(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToKebab(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"CreateUser", "create-user"},
		{"GetUser", "get-user"},
		{"XMLParser", "xml-parser"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		if got := ToKebab(tt.input); got != tt.want {
			t.Errorf("ToKebab(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToPascal(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"create_user", "CreateUser"},
		{"get-user-by-id", "GetUserById"},
		{"simple", "Simple"},
		{"", ""},
		{"a b", "AB"},
		{"x", "X"},
		{"CreateOrder", "Createorder"},
		{"AlreadyPascal", "Alreadypascal"},
	}

	for _, tt := range tests {
		if got := ToPascal(tt.input); got != tt.want {
			t.Errorf("ToPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
