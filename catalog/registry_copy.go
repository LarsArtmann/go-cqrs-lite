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
