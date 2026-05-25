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

func copyDataStore(ds *DataStore) DataStore {
	return DataStore{
		ID:             ds.ID,
		Name:           ds.Name,
		Version:        ds.Version,
		Summary:        ds.Summary,
		ContainerType:  ds.ContainerType,
		Technology:     ds.Technology,
		Classification: ds.Classification,
		Retention:      ds.Retention,
		Residency:      ds.Residency,
		Owners:         copySlice(ds.Owners),
		Badges:         copyBadges(ds.Badges),
	}
}

func copyFlow(f *Flow) Flow {
	return Flow{
		ID:      f.ID,
		Name:    f.Name,
		Version: f.Version,
		Summary: f.Summary,
		Steps:   copyFlowSteps(f.Steps),
		Badges:  copyBadges(f.Badges),
	}
}

func copyFlowSteps(steps []FlowStep) []FlowStep {
	if steps == nil {
		return nil
	}

	cp := make([]FlowStep, len(steps))

	for i, s := range steps {
		cp[i] = copyFlowStep(s)
	}

	return cp
}

func copyFlowStep(s FlowStep) FlowStep {
	return FlowStep{
		ID:        s.ID,
		Title:     s.Title,
		Summary:   s.Summary,
		Service:   copyFlowStepRef(s.Service),
		Message:   copyFlowStepRef(s.Message),
		Channel:   copyFlowStepRef(s.Channel),
		Actor:     copyFlowActor(s.Actor),
		External:  copyFlowActor(s.External),
		Custom:    copyFlowCustom(s.Custom),
		NextStep:  copyFlowEdge(s.NextStep),
		NextSteps: copySlice(s.NextSteps),
	}
}

func copyFlowStepRef(r *FlowStepRef) *FlowStepRef {
	if r == nil {
		return nil
	}

	return &FlowStepRef{ID: r.ID, Version: r.Version}
}

func copyFlowActor(a *FlowActor) *FlowActor {
	if a == nil {
		return nil
	}

	return &FlowActor{Name: a.Name, Summary: a.Summary, URL: a.URL}
}

func copyFlowCustom(c *FlowCustomNode) *FlowCustomNode {
	if c == nil {
		return nil
	}

	return &FlowCustomNode{
		Title: c.Title, Icon: c.Icon, Type: c.Type,
		Summary: c.Summary, URL: c.URL, Color: c.Color,
	}
}

func copyFlowEdge(e *FlowEdge) *FlowEdge {
	if e == nil {
		return nil
	}

	return &FlowEdge{ID: e.ID, Label: e.Label}
}

func copyTeam(t *Team) Team {
	return Team{
		ID:                    t.ID,
		Name:                  t.Name,
		Summary:               t.Summary,
		Members:               copySlice(t.Members),
		Email:                 t.Email,
		SlackDirectMessageURL: t.SlackDirectMessageURL,
	}
}

func copyUser(u *User) User {
	return User{
		ID:                    u.ID,
		Name:                  u.Name,
		AvatarURL:             u.AvatarURL,
		Role:                  u.Role,
		Email:                 u.Email,
		SlackDirectMessageURL: u.SlackDirectMessageURL,
	}
}
