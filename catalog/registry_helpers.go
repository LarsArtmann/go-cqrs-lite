package catalog

import "maps"

func copySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	cp := make([]T, len(s))
	copy(cp, s)

	return cp
}

func copyMessages(msgs []Message) []Message {
	if msgs == nil {
		return nil
	}

	copies := make([]Message, len(msgs))

	for i, m := range msgs {
		copies[i] = copyMessage(m)
	}

	return copies
}

func copyMessage(m Message) Message {
	return Message{
		Kind:       m.Kind,
		ID:         m.ID,
		Name:       m.Name,
		Version:    m.Version,
		Summary:    m.Summary,
		Schema:     m.Schema,
		Direction:  m.Direction,
		Examples:   copySlice(m.Examples),
		Owners:     copySlice(m.Owners),
		Labels:     copyMap(m.Labels),
		Deprecated: m.Deprecated,
		Changelog:  copySlice(m.Changelog),
	}
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	return maps.Clone(m)
}

func copyService(s *Service) Service {
	return Service{
		ID:       s.ID,
		Name:     s.Name,
		Version:  s.Version,
		Summary:  s.Summary,
		Owners:   copySlice(s.Owners),
		Commands: copyMessages(s.Commands),
		Events:   copyMessages(s.Events),
		Queries:  copyMessages(s.Queries),
	}
}

func copyDomain(d *Domain) Domain {
	return Domain{
		ID:       d.ID,
		Name:     d.Name,
		Version:  d.Version,
		Summary:  d.Summary,
		Owners:   copySlice(d.Owners),
		Services: copySlice(d.Services),
	}
}

func copyChannel(ch *Channel) Channel {
	return Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Version:   ch.Version,
		Summary:   ch.Summary,
		Address:   ch.Address,
		Protocols: copySlice(ch.Protocols),
		Messages:  copySlice(ch.Messages),
	}
}
