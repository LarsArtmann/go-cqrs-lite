package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServiceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ServiceID
		wantErr bool
	}{
		{"valid ID", "users", ServiceID("users"), false},
		{"hyphenated", "user-service", ServiceID("user-service"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseServiceID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "service ID cannot be empty")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustParseServiceID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseServiceID("orders")
		assert.Equal(t, ServiceID("orders"), id)
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MustParseServiceID("")
		})
	})
}

func TestServiceID_IsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, ServiceID("").IsZero())
	assert.False(t, ServiceID("users").IsZero())
}

func TestParseDomainID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    DomainID
		wantErr bool
	}{
		{"valid ID", "ecommerce", DomainID("ecommerce"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDomainID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "domain ID cannot be empty")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustParseDomainID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseDomainID("billing")
		assert.Equal(t, DomainID("billing"), id)
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MustParseDomainID("")
		})
	})
}

func TestDomainID_IsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, DomainID("").IsZero())
	assert.False(t, DomainID("orders").IsZero())
}

func TestParseMessageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    MessageID
		wantErr bool
	}{
		{"valid ID", "CreateUser", MessageID("CreateUser"), false},
		{"event style", "user.created", MessageID("user.created"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMessageID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "message ID cannot be empty")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustParseMessageID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseMessageID("CreateUser")
		assert.Equal(t, MessageID("CreateUser"), id)
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MustParseMessageID("")
		})
	})
}

func TestMessageID_IsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, MessageID("").IsZero())
	assert.False(t, MessageID("CreateUser").IsZero())
}

func TestParseChannelID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ChannelID
		wantErr bool
	}{
		{"valid ID", "user.commands", ChannelID("user.commands"), false},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseChannelID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "channel ID cannot be empty")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustParseChannelID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseChannelID("user.events")
		assert.Equal(t, ChannelID("user.events"), id)
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MustParseChannelID("")
		})
	})
}

func TestChannelID_IsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, ChannelID("").IsZero())
	assert.False(t, ChannelID("user.commands").IsZero())
}
