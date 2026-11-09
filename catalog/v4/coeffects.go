package catalog

import "fmt"

// DeriveProducersConsumers returns a copy of the catalog with
// producer/consumer relationships derived from service sends/receives:
// a service that send-declares an event is its producer, one that
// receive-declares it is a consumer. Explicit Producers/Consumers options
// take precedence and are never overwritten. The EventCatalog exporter runs
// this before rendering; it is exported so CI gates and custom renderers
// can reason about the same enriched graph.
func (c *Catalog) DeriveProducersConsumers() *Catalog {
	producerMap := map[MessageID][]ServiceID{}
	consumerMap := map[MessageID][]ServiceID{}

	for _, svc := range c.Services {
		for _, evt := range svc.Events {
			messageID := Key(evt)
			if evt.IsSend() {
				producerMap[messageID] = append(producerMap[messageID], svc.ID)
			} else {
				consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
			}
		}

		for _, cmd := range svc.Commands {
			messageID := Key(cmd)
			consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
		}

		for _, q := range svc.Queries {
			messageID := Key(q)
			consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
		}
	}

	enriched := *c
	enriched.Services = make([]Service, len(c.Services))

	for i, svc := range c.Services {
		svcCopy := svc
		svcCopy.Events = make([]Message, len(svc.Events))
		svcCopy.Commands = make([]Message, len(svc.Commands))
		svcCopy.Queries = make([]Message, len(svc.Queries))

		for j, evt := range svc.Events {
			evtCopy := evt
			messageID := Key(evt)

			if p, ok := producerMap[messageID]; ok && len(evtCopy.Producers) == 0 {
				evtCopy.Producers = p
			}

			if cons, ok := consumerMap[messageID]; ok && len(evtCopy.Consumers) == 0 {
				evtCopy.Consumers = cons
			}

			svcCopy.Events[j] = evtCopy
		}

		for j, cmd := range svc.Commands {
			cmdCopy := cmd
			messageID := Key(cmd)

			if cons, ok := consumerMap[messageID]; ok && len(cmdCopy.Consumers) == 0 {
				cmdCopy.Consumers = cons
			}

			svcCopy.Commands[j] = cmdCopy
		}

		for j, q := range svc.Queries {
			qCopy := q
			messageID := Key(q)

			if cons, ok := consumerMap[messageID]; ok && len(qCopy.Consumers) == 0 {
				qCopy.Consumers = cons
			}

			svcCopy.Queries[j] = qCopy
		}

		enriched.Services[i] = svcCopy
	}

	return &enriched
}

// ValidateCoeffects checks the coeffect graph after derivation: an event
// that services consume but nothing produces is a dangling subscription —
// the documentation-side twin of system.New's coeffect gate and cqrs-lint's
// E018 (the "user.creted" typo class). Explicitly declared external
// producers (the Producers option) are honored, so events imported from
// systems outside this catalog do not violate. Producer-without-consumer
// events are NOT violations (legitimate dead-event visibility lives in the
// export summary, mirroring the runtime advisory tier).
func (c *Catalog) ValidateCoeffects() []Violation {
	enriched := c.DeriveProducersConsumers()

	var violations []Violation

	seen := make(map[MessageID]bool)

	for _, svc := range enriched.Services {
		for _, evt := range svc.Events {
			messageID := Key(evt)
			if seen[messageID] {
				continue
			}

			seen[messageID] = true

			if len(evt.Consumers) == 0 || len(evt.Producers) > 0 {
				continue
			}

			violations = append(violations, Violation{
				Path: fmt.Sprintf("services[%s].event[%s]", svc.ID, messageID),
				Message: fmt.Sprintf(
					"coeffect dangling: event %q is consumed by %v but no service produces it — "+
						"declare a producer (Sends/Producers) or fix the event type",
					messageID,
					evt.Consumers,
				),
			})
		}
	}

	return violations
}
