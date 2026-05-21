package catalog

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	_ = fmt.Stringer(ServiceID(""))
	_ = fmt.Stringer(DomainID(""))
	_ = fmt.Stringer(MessageID(""))
	_ = fmt.Stringer(ChannelID(""))

	ErrEmptyServiceID = errorfamily.NewRejection(
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

type idType interface {
	~string
}

func parseID[T idType](s string, err error) (T, error) {
	if s == "" {
		return "", err
	}

	return T(s), nil
}

// IsZero returns true if the ServiceID is empty.
func (id ServiceID) IsZero() bool { return id == "" }

// ParseServiceID validates and creates a ServiceID from a string.
func ParseServiceID(s string) (ServiceID, error) {
	return parseID[ServiceID](s, ErrEmptyServiceID)
}

// IsZero returns true if the DomainID is empty.
func (id DomainID) IsZero() bool { return id == "" }

// ParseDomainID validates and creates a DomainID from a string.
func ParseDomainID(s string) (DomainID, error) {
	return parseID[DomainID](s, ErrEmptyDomainID)
}

// IsZero returns true if the MessageID is empty.
func (id MessageID) IsZero() bool { return id == "" }

// ParseMessageID validates and creates a MessageID from a string.
func ParseMessageID(s string) (MessageID, error) {
	return parseID[MessageID](s, ErrEmptyMessageID)
}

// IsZero returns true if the ChannelID is empty.
func (id ChannelID) IsZero() bool { return id == "" }

// ParseChannelID validates and creates a ChannelID from a string.
func ParseChannelID(s string) (ChannelID, error) {
	return parseID[ChannelID](s, ErrEmptyChannelID)
}
