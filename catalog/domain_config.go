package catalog

// DomainOption configures domain-level metadata.
type DomainOption func(*Domain)

// DomainSends declares messages this domain sends.
func DomainSends(msgs ...Ref) DomainOption {
	return func(d *Domain) {
		d.Sends = msgs
	}
}

// DomainReceives declares messages this domain receives.
func DomainReceives(msgs ...Ref) DomainOption {
	return func(d *Domain) {
		d.Receives = msgs
	}
}

// DomainEntities declares entities owned by this domain.
func DomainEntities(entities ...string) DomainOption {
	return func(d *Domain) {
		d.Entities = entities
	}
}

// DomainBadges sets visual badges on the domain.
func DomainBadges(badges ...Badge) DomainOption {
	return func(d *Domain) {
		d.Badges = badges
	}
}

// DomainOwners sets the list of owners for the domain.
func DomainOwners(owners ...string) DomainOption {
	return func(d *Domain) {
		d.Owners = owners
	}
}

// DomainAttachments links external resources to the domain.
func DomainAttachments(attachments ...Attachment) DomainOption {
	return func(d *Domain) {
		d.Attachments = attachments
	}
}

// DomainUbiquitousLanguage sets the DDD ubiquitous language glossary for the domain.
func DomainUbiquitousLanguage(terms ...UbiquitousLanguageTerm) DomainOption {
	return func(d *Domain) {
		d.UbiquitousLanguage = terms
	}
}

// DomainSubDomains declares sub-domains owned by this domain.
func DomainSubDomains(ids ...DomainID) DomainOption {
	return func(d *Domain) {
		d.SubDomains = ids
	}
}

// DomainDataProducts associates data products with this domain.
func DomainDataProducts(ids ...DataProductID) DomainOption {
	return func(d *Domain) {
		d.DataProducts = ids
	}
}
