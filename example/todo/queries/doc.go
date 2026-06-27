// Package queries implements the read side of the todo example.
//
// It defines read-only query structs (get, list, count), their result DTOs, and
// handlers that read from a domain.TodoReadModel. List results are wrapped in
// query.PaginatedResult and records are mapped to DTOs via FromDomain.
package queries
