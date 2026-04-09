package testhelpers

import "testing"

func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}
