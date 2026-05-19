package catalog

import (
	"errors"
	"testing"
)

func TestParseServiceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ServiceID
		wantErr bool
	}{
		{"valid ID", "users", ServiceID("users"), false},
		{"hyphenated", "user-service", ServiceID("user-service"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseServiceID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if !errors.Is(err, ErrEmptyServiceID) {
					t.Fatalf("expected ErrEmptyServiceID, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestMustParseServiceID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseServiceID("orders")
		if id != ServiceID("orders") {
			t.Fatalf("expected %q, got %q", "orders", id)
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()

		MustParseServiceID("")
	})
}

func TestServiceID_IsZero(t *testing.T) {
	t.Parallel()

	if !ServiceID("").IsZero() {
		t.Fatal("empty ServiceID should be zero")
	}

	if ServiceID("users").IsZero() {
		t.Fatal(`"users" ServiceID should not be zero`)
	}
}

func TestParseDomainID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    DomainID
		wantErr bool
	}{
		{"valid ID", "ecommerce", DomainID("ecommerce"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDomainID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if !errors.Is(err, ErrEmptyDomainID) {
					t.Fatalf("expected ErrEmptyDomainID, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestMustParseDomainID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseDomainID("billing")
		if id != DomainID("billing") {
			t.Fatalf("expected %q, got %q", "billing", id)
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()

		MustParseDomainID("")
	})
}

func TestDomainID_IsZero(t *testing.T) {
	t.Parallel()

	if !DomainID("").IsZero() {
		t.Fatal("empty DomainID should be zero")
	}

	if DomainID("orders").IsZero() {
		t.Fatal(`"orders" DomainID should not be zero`)
	}
}

func TestParseMessageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    MessageID
		wantErr bool
	}{
		{"valid ID", "CreateUser", MessageID("CreateUser"), false},
		{"event style", "user.created", MessageID("user.created"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMessageID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if !errors.Is(err, ErrEmptyMessageID) {
					t.Fatalf("expected ErrEmptyMessageID, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestMustParseMessageID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseMessageID("CreateUser")
		if id != MessageID("CreateUser") {
			t.Fatalf("expected %q, got %q", "CreateUser", id)
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()

		MustParseMessageID("")
	})
}

func TestMessageID_IsZero(t *testing.T) {
	t.Parallel()

	if !MessageID("").IsZero() {
		t.Fatal("empty MessageID should be zero")
	}

	if MessageID("CreateUser").IsZero() {
		t.Fatal(`"CreateUser" MessageID should not be zero`)
	}
}

func TestParseChannelID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ChannelID
		wantErr bool
	}{
		{"valid ID", "user.commands", ChannelID("user.commands"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseChannelID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if !errors.Is(err, ErrEmptyChannelID) {
					t.Fatalf("expected ErrEmptyChannelID, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestMustParseChannelID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseChannelID("user.events")
		if id != ChannelID("user.events") {
			t.Fatalf("expected %q, got %q", "user.events", id)
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()

		MustParseChannelID("")
	})
}

func TestChannelID_IsZero(t *testing.T) {
	t.Parallel()

	if !ChannelID("").IsZero() {
		t.Fatal("empty ChannelID should be zero")
	}

	if ChannelID("user.commands").IsZero() {
		t.Fatal(`"user.commands" ChannelID should not be zero`)
	}
}
