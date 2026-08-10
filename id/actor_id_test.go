package id

import (
	"encoding/json/v2"
	"fmt"
	"testing"
)

func TestNewUserActor(t *testing.T) {
	t.Parallel()

	uid, err := ParseUserID("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	if err != nil {
		t.Fatal(err)
	}

	a := NewUserActor(uid)

	if a.Kind() != ActorUser {
		t.Errorf("Kind() = %v, want ActorUser", a.Kind())
	}

	if a.Raw() != uid.String() {
		t.Errorf("Raw() = %q, want %q", a.Raw(), uid.String())
	}

	if a.IsZero() {
		t.Error("IsZero() = true, want false")
	}
}

func TestNewBotActor(t *testing.T) {
	t.Parallel()

	a := NewBotActor("github-actions")

	if a.Kind() != ActorBot {
		t.Errorf("Kind() = %v, want ActorBot", a.Kind())
	}

	if a.Raw() != "github-actions" {
		t.Errorf("Raw() = %q, want %q", a.Raw(), "github-actions")
	}
}

func TestNewSystemActor(t *testing.T) {
	t.Parallel()

	a := NewSystemActor("scheduler")

	if a.Kind() != ActorSystem {
		t.Errorf("Kind() = %v, want ActorSystem", a.Kind())
	}

	if a.Raw() != "scheduler" {
		t.Errorf("Raw() = %q, want %q", a.Raw(), "scheduler")
	}
}

func TestNewServiceActor(t *testing.T) {
	t.Parallel()

	a := NewServiceActor("api-gateway")

	if a.Kind() != ActorService {
		t.Errorf("Kind() = %v, want ActorService", a.Kind())
	}

	if a.Raw() != "api-gateway" {
		t.Errorf("Raw() = %q, want %q", a.Raw(), "api-gateway")
	}
}

func TestParseActorID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantKind  ActorKind
		wantRaw   string
		wantError bool
	}{
		{name: "user", input: "user:01H4S2Z4QX8N1P5K3M7R9T0V2W", wantKind: ActorUser, wantRaw: "01H4S2Z4QX8N1P5K3M7R9T0V2W"},
		{name: "bot", input: "bot:ci-runner", wantKind: ActorBot, wantRaw: "ci-runner"},
		{name: "system", input: "system:gc", wantKind: ActorSystem, wantRaw: "gc"},
		{name: "service", input: "service:notification-worker", wantKind: ActorService, wantRaw: "notification-worker"},
		{name: "empty", input: "", wantKind: ActorUnknown, wantRaw: ""},
		{name: "missing_colon", input: "userraw", wantError: true},
		{name: "unknown_kind", input: "alien:raw", wantError: true},
		{name: "empty_raw", input: "user:", wantKind: ActorUser, wantRaw: ""},
		{name: "colon_in_raw", input: "system:job:42", wantKind: ActorSystem, wantRaw: "job:42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseActorID(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Kind() != tt.wantKind {
				t.Errorf("Kind() = %v, want %v", got.Kind(), tt.wantKind)
			}

			if got.Raw() != tt.wantRaw {
				t.Errorf("Raw() = %q, want %q", got.Raw(), tt.wantRaw)
			}
		})
	}
}

func TestActorID_PrefixedString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    ActorID
		want string
	}{
		{"user", NewUserActor(mustParseUserID(t, "01H4S2Z4QX8N1P5K3M7R9T0V2W")), "user:01H4S2Z4QX8N1P5K3M7R9T0V2W"},
		{"system", NewSystemActor("scheduler"), "system:scheduler"},
		{"zero", ActorID{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.PrefixedString(); got != tt.want {
				t.Errorf("PrefixedString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActorID_IsZero(t *testing.T) {
	t.Parallel()

	if !(ActorID{}).IsZero() {
		t.Error("zero value should be zero")
	}

	if NewUserActor(NewUserID()).IsZero() {
		t.Error("constructed actor should not be zero")
	}

	zeroRaw := ActorID{kind: ActorUser, raw: ""}
	if !zeroRaw.IsZero() {
		t.Error("ActorID with empty raw should be zero regardless of kind")
	}
}

func TestActorID_Equal(t *testing.T) {
	t.Parallel()

	uid := mustParseUserID(t, "01H4S2Z4QX8N1P5K3M7R9T0V2W")
	a := NewUserActor(uid)
	b := NewUserActor(uid)

	if !a.Equal(b) {
		t.Error("same construction should be equal")
	}

	c := NewSystemActor("scheduler")
	if a.Equal(c) {
		t.Error("different kinds should not be equal")
	}

	d := NewUserActor(mustParseUserID(t, "01HK1540X0841Y0A6BSX1VKR95"))
	if a.Equal(d) {
		t.Error("different raw values should not be equal")
	}

	if !(ActorID{}).Equal(ActorID{}) {
		t.Error("zero values should be equal")
	}
}

func TestActorID_String(t *testing.T) {
	t.Parallel()

	a := NewSystemActor("scheduler")
	if a.String() != "scheduler" {
		t.Errorf("String() = %q, want %q", a.String(), "scheduler")
	}
}

func TestActorKind_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind ActorKind
		want string
	}{
		{ActorUnknown, "unknown"},
		{ActorUser, "user"},
		{ActorBot, "bot"},
		{ActorSystem, "system"},
		{ActorService, "service"},
		{ActorKind(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("ActorKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestActorID_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    ActorID
	}{
		{"user", NewUserActor(mustParseUserID(t, "01H4S2Z4QX8N1P5K3M7R9T0V2W"))},
		{"bot", NewBotActor("ci-runner")},
		{"system", NewSystemActor("gc")},
		{"service", NewServiceActor("api-gateway")},
		{"zero", ActorID{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.a)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got ActorID
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if !got.Equal(tt.a) {
				t.Errorf("roundtrip mismatch: got %v (prefixed=%q), want %v (prefixed=%q)",
					got, got.PrefixedString(), tt.a, tt.a.PrefixedString())
			}
		})
	}
}

func TestActorID_UnmarshalJSON_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
	}{
		{"missing_colon", `"userraw"`},
		{"unknown_kind", `"alien:raw"`},
		{"not_a_string", `42`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got ActorID
			if err := json.Unmarshal([]byte(tt.data), &got); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestActorID_ParseRoundtrip(t *testing.T) {
	t.Parallel()

	original := NewBotActor("deploy-bot")
	prefixed := original.PrefixedString()

	parsed, err := ParseActorID(prefixed)
	if err != nil {
		t.Fatalf("ParseActorID: %v", err)
	}

	if !parsed.Equal(original) {
		t.Errorf("ParseActorID(%q) = %v, want %v", prefixed, parsed, original)
	}
}

func TestActorID_Format(t *testing.T) {
	t.Parallel()

	a := NewSystemActor("scheduler")

	if got := fmt.Sprintf("%v", a); got != a.PrefixedString() {
		t.Errorf("Sprintf%%v = %q, want %q", got, a.PrefixedString())
	}

	if got := fmt.Sprintf("%s", a); got != a.PrefixedString() {
		t.Errorf("Sprintf%%s = %q, want %q", got, a.PrefixedString())
	}
}

func mustParseUserID(t *testing.T, s string) UserID {
	t.Helper()

	uid, err := ParseUserID(s)
	if err != nil {
		t.Fatal(err)
	}

	return uid
}
