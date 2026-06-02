package eventcatalog

import "github.com/larsartmann/go-cqrs-lite/catalog/v2"

// autoDeriveProducersConsumers returns a copy of the catalog with
// producer/consumer relationships auto-derived from service sends/receives.
func autoDeriveProducersConsumers(cat *catalog.Catalog) *catalog.Catalog {
	producerMap := map[catalog.MessageID][]catalog.ServiceID{}
	consumerMap := map[catalog.MessageID][]catalog.ServiceID{}

	for _, svc := range cat.Services {
		for _, evt := range svc.Events {
			messageID := catalog.Key(evt)
			if evt.IsSend() {
				producerMap[messageID] = append(producerMap[messageID], svc.ID)
			} else {
				consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
			}
		}

		for _, cmd := range svc.Commands {
			messageID := catalog.Key(cmd)
			consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
		}

		for _, q := range svc.Queries {
			messageID := catalog.Key(q)
			consumerMap[messageID] = append(consumerMap[messageID], svc.ID)
		}
	}

	enriched := *cat
	enriched.Services = make([]catalog.Service, len(cat.Services))

	for i, svc := range cat.Services {
		svcCopy := svc
		svcCopy.Events = make([]catalog.Message, len(svc.Events))
		svcCopy.Commands = make([]catalog.Message, len(svc.Commands))
		svcCopy.Queries = make([]catalog.Message, len(svc.Queries))

		for j, evt := range svc.Events {
			evtCopy := evt
			messageID := catalog.Key(evt)
			if p, ok := producerMap[messageID]; ok && len(evtCopy.Producers) == 0 {
				evtCopy.Producers = p
			}

			if c, ok := consumerMap[messageID]; ok && len(evtCopy.Consumers) == 0 {
				evtCopy.Consumers = c
			}

			svcCopy.Events[j] = evtCopy
		}

		for j, cmd := range svc.Commands {
			cmdCopy := cmd
			messageID := catalog.Key(cmd)
			if c, ok := consumerMap[messageID]; ok && len(cmdCopy.Consumers) == 0 {
				cmdCopy.Consumers = c
			}

			svcCopy.Commands[j] = cmdCopy
		}

		for j, q := range svc.Queries {
			qCopy := q
			messageID := catalog.Key(q)
			if c, ok := consumerMap[messageID]; ok && len(qCopy.Consumers) == 0 {
				qCopy.Consumers = c
			}

			svcCopy.Queries[j] = qCopy
		}

		enriched.Services[i] = svcCopy
	}

	return &enriched
}
