package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestParseSource(t *testing.T) {
	t.Parallel()

	t.Run("valid source", func(t *testing.T) {
		t.Parallel()

		s, err := event.ParseSource("api")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.String() != "api" {
			t.Errorf("expected api, got %s", s)
		}
		if s.IsEmpty() {
			t.Error("source should not be empty")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()

		s, err := event.ParseSource("  scheduler  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.String() != "scheduler" {
			t.Errorf("expected scheduler, got %s", s)
		}
	})

	t.Run("empty source errors", func(t *testing.T) {
		t.Parallel()

		_, err := event.ParseSource("")
		if err == nil {
			t.Error("expected error for empty source")
		}
	})

	t.Run("whitespace-only source errors", func(t *testing.T) {
		t.Parallel()

		_, err := event.ParseSource("   ")
		if err == nil {
			t.Error("expected error for whitespace-only source")
		}
	})
}

func TestSourceIsEmpty(t *testing.T) {
	t.Parallel()

	var s event.Source
	if !s.IsEmpty() {
		t.Error("zero-value Source should be empty")
	}
}

func TestParseIPAddress(t *testing.T) {
	t.Parallel()

	t.Run("valid IPv4", func(t *testing.T) {
		t.Parallel()

		ip, err := event.ParseIPAddress("192.168.1.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip.String() != "192.168.1.1" {
			t.Errorf("expected 192.168.1.1, got %s", ip)
		}
		if ip.IsEmpty() {
			t.Error("IP should not be empty")
		}
	})

	t.Run("valid IPv6", func(t *testing.T) {
		t.Parallel()

		ip, err := event.ParseIPAddress("::1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip.IsEmpty() {
			t.Error("IP should not be empty")
		}
	})

	t.Run("empty returns empty without error", func(t *testing.T) {
		t.Parallel()

		ip, err := event.ParseIPAddress("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ip.IsEmpty() {
			t.Error("empty IP should be empty")
		}
	})

	t.Run("invalid IP errors", func(t *testing.T) {
		t.Parallel()

		_, err := event.ParseIPAddress("not-an-ip")
		if err == nil {
			t.Error("expected error for invalid IP")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()

		ip, err := event.ParseIPAddress("  10.0.0.1  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip.String() != "10.0.0.1" {
			t.Errorf("expected 10.0.0.1, got %s", ip)
		}
	})
}

func TestIPAddressIsEmpty(t *testing.T) {
	t.Parallel()

	var ip event.IPAddress
	if !ip.IsEmpty() {
		t.Error("zero-value IPAddress should be empty")
	}
}

func TestParseUserAgent(t *testing.T) {
	t.Parallel()

	t.Run("parses valid user agent", func(t *testing.T) {
		t.Parallel()

		ua := event.ParseUserAgent("Mozilla/5.0")
		if ua.String() != "Mozilla/5.0" {
			t.Errorf("expected Mozilla/5.0, got %s", ua)
		}
		if ua.IsEmpty() {
			t.Error("user agent should not be empty")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()

		ua := event.ParseUserAgent("  test-agent  ")
		if ua.String() != "test-agent" {
			t.Errorf("expected test-agent, got %s", ua)
		}
	})

	t.Run("empty returns empty", func(t *testing.T) {
		t.Parallel()

		ua := event.ParseUserAgent("")
		if !ua.IsEmpty() {
			t.Error("empty user agent should be empty")
		}
	})

	t.Run("whitespace-only returns empty", func(t *testing.T) {
		t.Parallel()

		ua := event.ParseUserAgent("   ")
		if !ua.IsEmpty() {
			t.Error("whitespace-only user agent should be empty")
		}
	})
}

func TestUserAgentIsEmpty(t *testing.T) {
	t.Parallel()

	var ua event.UserAgent
	if !ua.IsEmpty() {
		t.Error("zero-value UserAgent should be empty")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("ParseVersion valid", func(t *testing.T) {
		t.Parallel()

		v, err := event.ParseVersion(5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Int() != 5 {
			t.Errorf("expected 5, got %d", v.Int())
		}
		if v.IsZero() {
			t.Error("version 5 should not be zero")
		}
	})

	t.Run("ParseVersion negative errors", func(t *testing.T) {
		t.Parallel()

		_, err := event.ParseVersion(-1)
		if err == nil {
			t.Error("expected error for negative version")
		}
	})

	t.Run("IsZero", func(t *testing.T) {
		t.Parallel()

		var v event.Version
		if !v.IsZero() {
			t.Error("zero-value Version should be zero")
		}
	})

	t.Run("Increment", func(t *testing.T) {
		t.Parallel()

		v := event.Version(3)
		v2 := v.Increment()
		if v2.Int() != 4 {
			t.Errorf("expected 4, got %d", v2.Int())
		}
		if v.Int() != 3 {
			t.Errorf("original should be unchanged, got %d", v.Int())
		}
	})

	t.Run("ParseVersion zero", func(t *testing.T) {
		t.Parallel()

		v, err := event.ParseVersion(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.IsZero() {
			t.Error("version 0 should be zero")
		}
	})
}
