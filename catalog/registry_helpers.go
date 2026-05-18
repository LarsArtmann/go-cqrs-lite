package catalog

func copySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	cp := make([]T, len(s))
	copy(cp, s)

	return cp
}

func copyService(s *Service) Service {
	return Service{
		ID:       s.ID,
		Name:     s.Name,
		Version:  s.Version,
		Summary:  s.Summary,
		Owners:   copySlice(s.Owners),
		Commands: copySlice(s.Commands),
		Events:   copySlice(s.Events),
		Queries:  copySlice(s.Queries),
	}
}

func copyDomain(d *Domain) Domain {
	return Domain{
		ID:       d.ID,
		Name:     d.Name,
		Version:  d.Version,
		Summary:  d.Summary,
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
