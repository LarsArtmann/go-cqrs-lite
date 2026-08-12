package catalog

import (
	"testing"
)

func TestCamelCaseToHuman(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"CreateUserCmd", "Create User"},
		{"CreateUserCommand", "Create User"},
		{"UserCreatedEvent", "User Created"},
		{"UserCreatedEvt", "User Created"},
		{"GetUserQuery", "Get User"},
		{"GetUserQry", "Get User"},
		{"ChangeUserName", "Change User Name"},
		{"Activate", "Activate"},
		{"X", "X"},
		{"Cmd", "Cmd"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := camelCaseToHuman(tt.input)
			if got != tt.want {
				t.Errorf("camelCaseToHuman(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
