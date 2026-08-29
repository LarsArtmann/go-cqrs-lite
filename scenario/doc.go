// Package scenario provides fluent BDD-style test harnesses for deciders,
// projections, and event buses.
//
// Every scenario requires a terminal Then* assertion: a chain that stops
// before any Then* method fails the test at cleanup instead of passing
// vacuously.
//
// # Decider Testing
//
//	scenario.Given(t, apply, initial, evt1, evt2).
//	    When(cmd, decide).
//	    Then("user.created")
//
// # Projection Testing
//
//	scenario.GivenProjection(t, projection, evt1, evt2).
//	    ThenNoError()
package scenario
