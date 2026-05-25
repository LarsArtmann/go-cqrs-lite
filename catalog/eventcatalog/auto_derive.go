package eventcatalog

import "github.com/larsartmann/go-cqrs-lite/catalog"

// autoDeriveProducersConsumers returns a copy of the catalog with
// producer/consumer relationships auto-derived from service sends/receives.
func autoDeriveProducersConsumers(cat *catalog.Catalog) *catalog.Catalog {
	producerMap := map[string][]catalog.ServiceID{}
	consumerMap := map[string][]catalog.ServiceID{}

	for _, svc := range cat.Services {
		for _, evt := range svc.Events {
			msgID := string(catalog.GetID(evt))
			if evt.IsSend() {
				producerMap[msgID] = append(producerMap[msgID], svc.ID)
			} else {
				consumerMap[msgID] = append(consumerMap[msgID], svc.ID)
			}
		}

		for _, cmd := range svc.Commands {
			msgID := string(catalog.GetID(cmd))
			consumerMap[msgID] = append(consumerMap[msgID], svc.ID)
		}

		for _, q := range svc.Queries {
			msgID := string(catalog.GetID(q))
			consumerMap[msgID] = append(consumerMap[msgID], svc.ID)
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
			msgID := string(catalog.GetID(evt))
			if p, ok := producerMap[msgID]; ok && len(evtCopy.Producers) == 0 {
				evtCopy.Producers = p
			}

			if c, ok := consumerMap[msgID]; ok && len(evtCopy.Consumers) == 0 {
				evtCopy.Consumers = c
			}

			svcCopy.Events[j] = evtCopy
		}

		for j, cmd := range svc.Commands {
			cmdCopy := cmd
			msgID := string(catalog.GetID(cmd))
			if c, ok := consumerMap[msgID]; ok && len(cmdCopy.Consumers) == 0 {
				cmdCopy.Consumers = c
			}

			svcCopy.Commands[j] = cmdCopy
		}

		for j, q := range svc.Queries {
			qCopy := q
			msgID := string(catalog.GetID(q))
			if c, ok := consumerMap[msgID]; ok && len(qCopy.Consumers) == 0 {
				qCopy.Consumers = c
			}

			svcCopy.Queries[j] = qCopy
		}

		enriched.Services[i] = svcCopy
	}

	return &enriched
}
