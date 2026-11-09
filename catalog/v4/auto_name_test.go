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
		{"ExternalIDSetPayload", "External ID Set Payload"},
		{"APIKeyPayload", "API Key Payload"},
		{"PhoneNumberUpdatedPayload", "Phone Number Updated Payload"},
		{"Base64Encoder", "Base64 Encoder"},
		{"ParseHTTP2Server", "Parse HTTP2 Server"},
		{"UserCreated", "User Created"},
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

func TestDisplayNameFromID(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"crm.count", "Crm Count"},
		{"contact_created", "Contact Created"},
		{"commands-wire-hooks", "Commands Wire Hooks"},
		{"externalIDSet", "External ID Set"},
		{"single", "Single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if got := displayNameFromID(MessageID(tt.input)); got != tt.want {
				t.Errorf("displayNameFromID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
