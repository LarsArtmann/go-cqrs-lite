// Package systemscenario is the system-level BDD testing harness for
// go-cqrs-lite (ADR-0153).
//
// It boots a REAL system via [system.New] with the same DomainConfig and
// DeploymentConfig a production binary uses, then drives Axon-style
// Given/When/Then chains against it:
//
//	sc := systemscenario.System(t, ctx, domainCfg, deployCfg)
//
//	sc.Given(
//		sc.Event("task.created", ref, TaskCreated{Title: "ship"}),
//	).When(cmdComplete).Then("task.completed")
//
// The fixture-from-production-config property (the load-bearing Axon 5
// lesson) means harness tests cannot drift from the wiring the production
// binary boots: there is nothing to re-declare.
//
// # Determinism contract
//
// Command dispatch is synchronous through the journal: event assertions
// (Then, ThenEvents, ThenNoEvents, ThenCommands) diff the journal against a
// baseline captured when the first When act began. Projection folding is
// asynchronous (bus delivery), so query assertions (ThenQuery, ThenQueryFunc)
// poll until the result matches or [WithAwaitTimeout] (default 5s) expires.
// TimeAdvances flips the scenario into await mode (timer firing is
// scheduler-tick asynchronous).
//
// # Phase mapping to Axon
//
//	Given(events)          ~ fixture.given().events(...)
//	Given().Command(...)   ~ fixture.given().commands(...)
//	When(cmd)              ~ fixture.when().command(...)
//	When().Event(...)      ~ fixture.when().event(...)
//	WhenQuery(q)           ~ (no Axon equivalent — queries are first-class here)
//	When().TimeAdvances(d) ~ Axon 4 whenTimeElapses (dropped in Axon 5)
//	Then("t.done")         ~ fixture.then().events(...)
//	ThenCommands(...)      ~ fixture.then().commands(...)
//	ThenError(...)         ~ fixture.then().exception(...)
//	ThenSuccess()          ~ fixture.then().success()
package systemscenario
