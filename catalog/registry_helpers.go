package catalog

import (
	"maps"
	"slices"
)

func copySlice[T any](s []T) []T {
	return slices.Clone(s)
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
		Producers:  copySlice(m.Producers),
		Consumers:  copySlice(m.Consumers),
		Operation:  copyOperation(m.Operation),
		Badges:     copyBadges(m.Badges),
		Repository: copyRepository(m.Repository),
	}
}

func copyOperation(op *Operation) *Operation {
	if op == nil {
		return nil
	}

	return &Operation{
		Method:      op.Method,
		Path:        op.Path,
		StatusCodes: copySlice(op.StatusCodes),
	}
}

func copyRepository(r *Repository) *Repository {
	if r == nil {
		return nil
	}

	return &Repository{
		Language: r.Language,
		URL:      r.URL,
	}
}

func copyBadges(badges []Badge) []Badge {
	return copySlice(badges)
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	return maps.Clone(m)
}

func copyService(s *Service) Service {
	return Service{
		ID:             s.ID,
		Name:           s.Name,
		Version:        s.Version,
		Summary:        s.Summary,
		Owners:         copySlice(s.Owners),
		Commands:       copyMessages(s.Commands),
		Events:         copyMessages(s.Events),
		Queries:        copyMessages(s.Queries),
		WritesTo:       copySlice(s.WritesTo),
		ReadsFrom:      copySlice(s.ReadsFrom),
		Entities:       copySlice(s.Entities),
		Flows:          copySlice(s.Flows),
		Repository:     copyRepository(s.Repository),
		Badges:         copyBadges(s.Badges),
		Specifications: copySlice(s.Specifications),
		Attachments:    copySlice(s.Attachments),
	}
}

func copyDomain(d *Domain) Domain {
	return Domain{
		ID:          d.ID,
		Name:        d.Name,
		Version:     d.Version,
		Summary:     d.Summary,
		Owners:      copySlice(d.Owners),
		Services:    copySlice(d.Services),
		Sends:       copySlice(d.Sends),
		Receives:    copySlice(d.Receives),
		Entities:    copySlice(d.Entities),
		Flows:       copySlice(d.Flows),
		Badges:      copyBadges(d.Badges),
		Attachments: copySlice(d.Attachments),
	}
}

func copyChannel(ch *Channel) Channel {
	return Channel{
		ID:                ch.ID,
		Name:              ch.Name,
		Version:           ch.Version,
		Summary:           ch.Summary,
		Address:           ch.Address,
		Protocols:         copySlice(ch.Protocols),
		Messages:          copySlice(ch.Messages),
		DeliveryGuarantee: ch.DeliveryGuarantee,
		Parameters:        copyChannelParams(ch.Parameters),
		Routes:            copySlice(ch.Routes),
		Owners:            copySlice(ch.Owners),
		Badges:            copyBadges(ch.Badges),
	}
}

func copyChannelParams(params map[string]ChannelParam) map[string]ChannelParam {
	if params == nil {
		return nil
	}

	cp := make(map[string]ChannelParam, len(params))

	for k, v := range params {
		cp[k] = ChannelParam{
			Enum:        copySlice(v.Enum),
			Default:     v.Default,
			Description: v.Description,
		}
	}

	return cp
}
