package catalog

func copyServicePtr(s Service) *Service {
	cp := copyService(&s)

	return &cp
}

func copyDomainPtr(d Domain) *Domain {
	cp := copyDomain(&d)

	return &cp
}

func copyChannelPtr(ch Channel) *Channel {
	cp := copyChannel(&ch)

	return &cp
}

func copyDataStorePtr(ds DataStore) *DataStore {
	cp := copyDataStore(&ds)

	return &cp
}

func copyFlowPtr(f Flow) *Flow {
	cp := copyFlow(&f)

	return &cp
}

func copyTeamPtr(t Team) *Team {
	cp := copyTeam(&t)

	return &cp
}

func copyUserPtr(u User) *User {
	cp := copyUser(&u)

	return &cp
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
