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
