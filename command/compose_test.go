package command_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
)

func TestCompose_AllNil(t *testing.T) {
	t.Parallel()

	err := command.Compose(nil, nil)
	if err != nil {
		t.Fatalf("Compose(nil, nil) = %v, want nil", err)
	}
}

func TestCompose_SingleError(t *testing.T) {
	t.Parallel()

	base := errors.New("single error")
	err := command.Compose(base)

	if err == nil {
		t.Fatal("Compose(single) = nil, want error")
	}

	if !errors.Is(err, base) {
		t.Fatalf("Compose(single) does not contain base error")
	}
}

func TestCompose_MultipleErrors(t *testing.T) {
	t.Parallel()

	err1 := errors.New("first")
	err2 := errors.New("second")
	err3 := errors.New("third")

	combined := command.Compose(err1, err2, err3)

	if combined == nil {
		t.Fatal("Compose(multiple) = nil, want error")
	}

	for _, e := range []error{err1, err2, err3} {
		if !errors.Is(combined, e) {
			t.Fatalf("Compose result missing error: %v", e)
		}
	}
}

func TestCompose_WithClassifiedErrors(t *testing.T) {
	t.Parallel()

	rejection := command.NewRejection("cmd.rejected", "rejected")
	infrastructure := command.NewInfrastructure("cmd.infra", "infra failed")

	combined := command.Compose(rejection, infrastructure)
	if combined == nil {
		t.Fatal("Compose(classified) = nil, want error")
	}

	if !errors.Is(combined, rejection) {
		t.Fatal("missing rejection in composed error")
	}

	if !errors.Is(combined, infrastructure) {
		t.Fatal("missing infrastructure in composed error")
	}
}

func TestCompose_MixNilAndErrors(t *testing.T) {
	t.Parallel()

	err1 := errors.New("real error")
	combined := command.Compose(nil, err1, nil)

	if combined == nil {
		t.Fatal("Compose(nil, err, nil) = nil, want error")
	}

	if !errors.Is(combined, err1) {
		t.Fatal("missing real error in composed result")
	}
}
