package catalog

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseServiceID(t *testing.T) {
	t.Parallel()

	testParseID(t, "ServiceID", ParseServiceID, []idTestCase[ServiceID]{
		{"valid ID", "users", ServiceID("users"), false, ErrEmptyServiceID},
		{"hyphenated", "user-service", ServiceID("user-service"), false, ErrEmptyServiceID},
		{"empty string", "", "", true, ErrEmptyServiceID},
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

	testParseID(t, "DomainID", ParseDomainID, []idTestCase[DomainID]{
		{"valid ID", "ecommerce", DomainID("ecommerce"), false, ErrEmptyDomainID},
		{"empty string", "", "", true, ErrEmptyDomainID},
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

	testParseID(t, "MessageID", ParseMessageID, []idTestCase[MessageID]{
		{"valid ID", "CreateUser", MessageID("CreateUser"), false, ErrEmptyMessageID},
		{"event style", "user.created", MessageID("user.created"), false, ErrEmptyMessageID},
		{"empty string", "", "", true, ErrEmptyMessageID},
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

	testParseID(t, "ChannelID", ParseChannelID, []idTestCase[ChannelID]{
		{"valid ID", "user.commands", ChannelID("user.commands"), false, ErrEmptyChannelID},
		{"empty string", "", "", true, ErrEmptyChannelID},
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

type idTestCase[ID comparable] struct {
	name    string
	input   string
	want    ID
	wantErr bool
	errSent error
}

func testParseID[ID comparable](
	t *testing.T,
	label string,
	parse func(string) (ID, error),
	cases []idTestCase[ID],
) {
	t.Helper()

	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s/%s", label, tt.name), func(t *testing.T) {
			t.Parallel()

			got, err := parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if !errors.Is(err, tt.errSent) {
					t.Fatalf("expected %v, got %v", tt.errSent, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
