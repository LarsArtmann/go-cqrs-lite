package catalog

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
		ID:          s.ID,
		Title:       s.Title,
		Summary:     s.Summary,
		Service:     copyFlowStepRef(s.Service),
		Message:     copyFlowStepRef(s.Message),
		Channel:     copyFlowStepRef(s.Channel),
		Actor:       copyFlowActor(s.Actor),
		External:    copyFlowActor(s.External),
		Custom:      copyFlowCustom(s.Custom),
		Agent:       copyFlowStepRef(s.Agent),
		DataStore:   copyFlowStepRef(s.DataStore),
		DataProduct: copyFlowStepRef(s.DataProduct),
		SubFlow:     copyFlowStepRef(s.SubFlow),
		NextStep:    copyFlowEdge(s.NextStep),
		NextSteps:   copySlice(s.NextSteps),
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

func copyEntity(e *Entity) Entity {
	return Entity{
		ID:      e.ID,
		Name:    e.Name,
		Version: e.Version,
		Summary: e.Summary,
		Schema:  e.Schema,
		Owners:  copySlice(e.Owners),
		Badges:  copyBadges(e.Badges),
	}
}

func copyDataProduct(dp *DataProduct) DataProduct {
	return DataProduct{
		ID:      dp.ID,
		Name:    dp.Name,
		Version: dp.Version,
		Summary: dp.Summary,
		Inputs:  copySlice(dp.Inputs),
		Outputs: copySlice(dp.Outputs),
		Owners:  copySlice(dp.Owners),
		Badges:  copyBadges(dp.Badges),
	}
}

func copyAgent(a *Agent) Agent {
	return Agent{
		ID:        a.ID,
		Name:      a.Name,
		Version:   a.Version,
		Summary:   a.Summary,
		Sends:     copySlice(a.Sends),
		Receives:  copySlice(a.Receives),
		ReadsFrom: copySlice(a.ReadsFrom),
		WritesTo:  copySlice(a.WritesTo),
		Model:     copyAgentModel(a.Model),
		Tools:     copyAgentTools(a.Tools),
		Flows:     copySlice(a.Flows),
		Owners:    copySlice(a.Owners),
		Badges:    copyBadges(a.Badges),
	}
}

func copyAgentModel(m *AgentModel) *AgentModel {
	if m == nil {
		return nil
	}

	return &AgentModel{Provider: m.Provider, Name: m.Name, Version: m.Version}
}

func copyAgentTools(tools []AgentTool) []AgentTool {
	if tools == nil {
		return nil
	}

	cp := make([]AgentTool, len(tools))
	for i, t := range tools {
		cp[i] = AgentTool{
			Name: t.Name, Type: t.Type, URL: t.URL,
			Description: t.Description, Icon: t.Icon,
		}
	}

	return cp
}
