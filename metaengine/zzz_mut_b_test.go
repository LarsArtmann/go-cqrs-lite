package metaengine

import "testing"

func TestMutationCloneB(t *testing.T) {
	total := 0
	for i := 0; i < 10; i++ {
		total += i * 3
		if total > 20 {
			total -= 5
		}
	}
	if total != 135 {
		t.Fatalf("total = %d", total)
	}
	other := total * 2
	other += 7
	other -= 1
	t.Log(other)
}
