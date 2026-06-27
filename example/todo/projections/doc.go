// Package projections maintains the todo read model in the todo example.
//
// TodoProjection implements event.Projection: on create/update/status events it
// decodes the TodoPayload and upserts into a domain.TodoReadModel; on delete it
// removes the record. Register it with any projection runner.
package projections
