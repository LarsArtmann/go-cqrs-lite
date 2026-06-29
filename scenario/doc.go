// Package scenario provides fluent BDD-style test harnesses for deciders,
// projections, and event buses.
//
// # Decider Testing
//
//	scenario.Given(apply, initial, evt1, evt2).
//	    When(cmd, decide).
//	    Then(expectedEventTypes)
//
// # Projection Testing
//
//	scenario.GivenProjection(projection, evt1, evt2).
//	    ThenNoError()
package scenario
