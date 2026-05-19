package catalog

import "fmt"

var (
	_ fmt.Stringer = ServiceID("")
	_ fmt.Stringer = DomainID("")
	_ fmt.Stringer = MessageID("")
	_ fmt.Stringer = ChannelID("")
)

// IsZero returns true if the ServiceID is empty.
func (id ServiceID) IsZero() bool { return id == "" }

// ParseServiceID validates and creates a ServiceID from a string.
// Returns an error if the string is empty.
func ParseServiceID(s string) (ServiceID, error) {
	if s == "" {
		return "", fmt.Errorf("service ID cannot be empty") //nolint:err113 // specific value required
	}

	return ServiceID(s), nil
}

// MustParseServiceID parses a ServiceID or panics.
func MustParseServiceID(s string) ServiceID {
	id, err := ParseServiceID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// IsZero returns true if the DomainID is empty.
func (id DomainID) IsZero() bool { return id == "" }

// ParseDomainID validates and creates a DomainID from a string.
// Returns an error if the string is empty.
func ParseDomainID(s string) (DomainID, error) {
	if s == "" {
		return "", fmt.Errorf("domain ID cannot be empty") //nolint:err113 // specific value required
	}

	return DomainID(s), nil
}

// MustParseDomainID parses a DomainID or panics.
func MustParseDomainID(s string) DomainID {
	id, err := ParseDomainID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// IsZero returns true if the MessageID is empty.
func (id MessageID) IsZero() bool { return id == "" }

// ParseMessageID validates and creates a MessageID from a string.
// Returns an error if the string is empty.
func ParseMessageID(s string) (MessageID, error) {
	if s == "" {
		return "", fmt.Errorf("message ID cannot be empty") //nolint:err113 // specific value required
	}

	return MessageID(s), nil
}

// MustParseMessageID parses a MessageID or panics.
func MustParseMessageID(s string) MessageID {
	id, err := ParseMessageID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// IsZero returns true if the ChannelID is empty.
func (id ChannelID) IsZero() bool { return id == "" }

// ParseChannelID validates and creates a ChannelID from a string.
// Returns an error if the string is empty.
func ParseChannelID(s string) (ChannelID, error) {
	if s == "" {
		return "", fmt.Errorf("channel ID cannot be empty") //nolint:err113 // specific value required
	}

	return ChannelID(s), nil
}

// MustParseChannelID parses a ChannelID or panics.
func MustParseChannelID(s string) ChannelID {
	id, err := ParseChannelID(s)
	if err != nil {
		panic(err)
	}

	return id
}
