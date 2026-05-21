package catalog

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	_                 fmt.Stringer = ServiceID("")
	_                 fmt.Stringer = DomainID("")
	_                 fmt.Stringer = MessageID("")
	_                 fmt.Stringer = ChannelID("")
	ErrEmptyServiceID              = errorfamily.NewRejection(
		"catalog.empty_service_id",
		"service ID cannot be empty",
	)
	ErrEmptyDomainID = errorfamily.NewRejection(
		"catalog.empty_domain_id",
		"domain ID cannot be empty",
	)
	ErrEmptyMessageID = errorfamily.NewRejection(
		"catalog.empty_message_id",
		"message ID cannot be empty",
	)
	ErrEmptyChannelID = errorfamily.NewRejection(
		"catalog.empty_channel_id",
		"channel ID cannot be empty",
	)
)

// IsZero returns true if the ServiceID is empty.
func (id ServiceID) IsZero() bool { return id == "" }

// ParseServiceID validates and creates a ServiceID from a string.
// Returns an error if the string is empty.
func ParseServiceID(s string) (ServiceID, error) {
	if s == "" {
		return "", errorfamily.WrapRejection(
			ErrEmptyServiceID,
			"catalog.parse_service_id",
			"parse service ID",
		)
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
		return "", errorfamily.WrapRejection(
			ErrEmptyDomainID,
			"catalog.parse_domain_id",
			"parse domain ID",
		)
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
		return "", errorfamily.WrapRejection(
			ErrEmptyMessageID,
			"catalog.parse_message_id",
			"parse message ID",
		)
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
		return "", errorfamily.WrapRejection(
			ErrEmptyChannelID,
			"catalog.parse_channel_id",
			"parse channel ID",
		)
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
