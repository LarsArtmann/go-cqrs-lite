package event

import "testing"

func TestParseSchemaVersion(t *testing.T) {
	t.Parallel()

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		sv, err := parseSchemaVersion(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sv.Int() != 2 {
			t.Errorf("expected 2, got %d", sv.Int())
		}
	})

	t.Run("zero errors", func(t *testing.T) {
		t.Parallel()

		_, err := parseSchemaVersion(0)
		if err == nil {
			t.Error("expected error for zero")
		}
	})

	t.Run("negative errors", func(t *testing.T) {
		t.Parallel()

		_, err := parseSchemaVersion(-1)
		if err == nil {
			t.Error("expected error for negative")
		}
	})
}
