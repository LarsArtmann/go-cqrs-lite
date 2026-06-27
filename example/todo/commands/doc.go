// Package commands implements the write side of the todo example.
//
// For each todo operation (create, update, delete, change status) it defines a
// command struct with a JSON type discriminator, a constructor, and a handler
// that wraps an event-sourced decider.Repository and delegates to the matching
// aggregate.Decide* function. Handlers register against the Todo aggregate.
package commands
