// Package testing provides fluent BDD-style test harnesses for deciders,
// projections, and event buses.
//
// # Decider Testing
//
//	dqtesting.Given(decider, evt1, evt2).
//	    When(cmd).
//	    ThenDecider(expectedEvents)
//
// # Projection Testing
//
//	dqtesting.GivenProjection(projection, evt1, evt2).
//	    ThenProjectionNoError()
package cqrs_testing
