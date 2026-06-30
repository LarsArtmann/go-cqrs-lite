package catalog

// ServiceOption configures service-level metadata (badges, repository, etc.).
type ServiceOption func(*Service)

// ServiceBadges sets visual badges on the service.
func ServiceBadges(badges ...Badge) ServiceOption {
	return func(s *Service) {
		s.Badges = badges
	}
}

// ServiceRepository attaches code repository metadata to the service.
func ServiceRepository(language, url string) ServiceOption {
	return func(s *Service) {
		s.Repository = &Repository{Language: Language(language), URL: URL(url)}
	}
}

// ServiceWritesTo declares which data stores this service writes to.
func ServiceWritesTo(storeIDs ...DataStoreID) ServiceOption {
	return func(s *Service) {
		s.WritesTo = storeIDs
	}
}

// ServiceReadsFrom declares which data stores this service reads from.
func ServiceReadsFrom(storeIDs ...DataStoreID) ServiceOption {
	return func(s *Service) {
		s.ReadsFrom = storeIDs
	}
}

// ServiceEntities declares the entities owned by this service.
func ServiceEntities(entities ...string) ServiceOption {
	return func(s *Service) {
		s.Entities = entities
	}
}

// ServiceSpecifications attaches API specifications to the service.
func ServiceSpecifications(specs ...Specification) ServiceOption {
	return func(s *Service) {
		s.Specifications = specs
	}
}

// ServiceAttachments links external resources (ADRs, runbooks, etc.) to the service.
func ServiceAttachments(attachments ...Attachment) ServiceOption {
	return func(s *Service) {
		s.Attachments = attachments
	}
}

// ServiceOwners sets the list of owners (teams or individuals) for the service.
func ServiceOwners(owners ...string) ServiceOption {
	return func(s *Service) {
		s.Owners = owners
	}
}

// ServiceExternalSystem marks a service as an external (third-party) system.
func ServiceExternalSystem() ServiceOption {
	return func(s *Service) {
		s.ExternalSystem = true
	}
}
