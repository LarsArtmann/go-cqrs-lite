package systemscenario

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// await polls probe every 10ms until it passes or the await timeout
// expires, then fails the test with what and the probe's last detail.
func (s *Scenario) await(what string, probe func() (bool, string)) {
	s.t.Helper()

	deadline := time.Now().Add(s.cfg.awaitTimeout)

	for {
		ok, detail := probe()
		if ok {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("%s: condition not met within %s: %s", what, s.cfg.awaitTimeout, detail)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// ThenQuery polls fn until its result deep-equals want, awaiting the
// asynchronous projection pipeline (bus delivery + fold) — the read-model
// analog of scenario/v4's ThenQueryResult, with the await built in. Query
// errors keep polling (a projection may not have the row yet); the last
// error surfaces on timeout.
func (p *WhenPhase) ThenQuery(fn func() (any, error), want any) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenQuery")

	s.await("ThenQuery", func() (bool, string) {
		got, err := fn()
		if err != nil {
			return false, "query returned error: " + err.Error()
		}

		if reflect.DeepEqual(got, want) {
			return true, ""
		}

		return false, fmt.Sprintf("result mismatch\nwant: %#v\ngot:  %#v", want, got)
	})

	return p
}

// ThenQueryFunc polls fn until check accepts the result (returns nil),
// awaiting the asynchronous projection pipeline. Use it when the wanted
// result is easier to describe than to construct (field subsets, orderings).
func (p *WhenPhase) ThenQueryFunc(fn func() (any, error), check func(got any) error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenQueryFunc")

	s.await("ThenQueryFunc", func() (bool, string) {
		got, err := fn()
		if err != nil {
			return false, "query returned error: " + err.Error()
		}

		if err := check(got); err != nil {
			return false, err.Error()
		}

		return true, ""
	})

	return p
}

// ThenQueryFails asserts the query fn returns an error matching target
// (errors.Is) IMMEDIATELY - the negative twin of ThenQuery. Use it for
// lookups of missing entities: unlike ThenQuery, which keeps polling
// (assuming eventual projection consistency), a failed query here fails the
// test right away with the query's error.
func (p *WhenPhase) ThenQueryFails(fn func() (any, error), target error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenQueryFails")

	got, err := fn()
	if err == nil {
		s.t.Fatalf("ThenQueryFails: expected error %v, got result %#v", target, got)
	}

	if !errors.Is(err, target) {
		s.t.Fatalf("ThenQueryFails: error mismatch\nwant: %v\ngot:  %v", target, err)
	}

	return p
}

// ThenQueryTyped is the generic twin of [WhenPhase.ThenQueryFunc]: fn returns
// a typed result, check receives the typed value — no any-assertion dance.
// It is package-level because Go methods cannot take type parameters.
//
//	systemscenario.ThenQueryTyped(phase,
//		func() (UserView, error) { return FindUserByID(ctx, sys, id) },
//		func(user UserView) error {
//			if user.Email != want { return fmt.Errorf("Email: want %s, got %s", want, user.Email) }
//			return nil
//		})
func ThenQueryTyped[T any](p *WhenPhase, fn func() (T, error), check func(got T) error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenQueryTyped")

	s.await("ThenQueryTyped", func() (bool, string) {
		got, err := fn()
		if err != nil {
			return false, "query returned error: " + err.Error()
		}

		if err := check(got); err != nil {
			return false, err.Error()
		}

		return true, ""
	})

	return p
}

// ThenResult asserts the captured result of the WhenQuery act deep-equals
// want. Query dispatch is synchronous, so no polling.
func (p *WhenPhase) ThenResult(want any) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenResult")

	if !reflect.DeepEqual(s.lastQueryResult, want) {
		s.t.Fatalf(
			"ThenResult: query result mismatch\nwant: %#v\ngot:  %#v",
			want,
			s.lastQueryResult,
		)
	}

	return p
}

// ThenSuccess asserts the most recent act produced no error (Axon
// then().success analog) — the positive counterpart of ThenError.
func (p *WhenPhase) ThenSuccess() *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenSuccess")

	if s.lastErr != nil {
		s.t.Fatalf("ThenSuccess: last act returned error: %v", s.lastErr)
	}

	return p
}
