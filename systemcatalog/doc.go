// Package systemcatalog derives architecture catalogs from declared truth:
// a system.DomainConfig plus the runtime registrations it produces.
//
// Fleet consumers hand-copied their message inventories into
// catalog.Builder calls; the copies drift the moment a command, event, or
// query is added. This package replaces the copy with derivation:
//
//   - Commands and queries come from the booted System's registration
//     records (runtime truth), with payload schemas reflected from the Go
//     types the handlers were registered with.
//   - Events come from DomainConfig.Events union DomainConfig.Schema, with
//     the Schema declaration's current version stamped on each message.
//
// The derived catalog documents ONE service per deployable (the fleet
// convention); curated resources (flows, owners, data stores) are added via
// [WithCurator] or [DeriveInto] on a shared builder and survive derivation
// by construction — derivation adds truth, it never culls.
package systemcatalog
