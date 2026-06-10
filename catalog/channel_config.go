package catalog

// ChannelOption configures channel-level metadata.
type ChannelOption func(*Channel)

// ChannelAddress sets the channel address (e.g., "user.orders").
func ChannelAddress(addr string) ChannelOption {
	return func(ch *Channel) {
		ch.Address = Address(addr)
	}
}

// ChannelProtocols sets the supported protocols (e.g., "http", "kafka").
func ChannelProtocols(protocols ...string) ChannelOption {
	return func(ch *Channel) {
		ch.Protocols = make([]Protocol, len(protocols))
		for i, p := range protocols {
			ch.Protocols[i] = Protocol(p)
		}
	}
}

// ChannelMessages associates message IDs with this channel.
func ChannelMessages(msgIDs ...MessageID) ChannelOption {
	return func(ch *Channel) {
		ch.Messages = msgIDs
	}
}

// ChannelDeliveryGuarantee sets the delivery guarantee (e.g., "at-least-once").
func ChannelDeliveryGuarantee(guarantee string) ChannelOption {
	return func(ch *Channel) {
		ch.DeliveryGuarantee = DeliveryGuarantee(guarantee)
	}
}

// ChannelParameters sets dynamic addressing parameters.
func ChannelParameters(params map[string]ChannelParam) ChannelOption {
	return func(ch *Channel) {
		ch.Parameters = params
	}
}

// ChannelRoutes sets routing rules for this channel.
func ChannelRoutes(routes ...ChannelRoute) ChannelOption {
	return func(ch *Channel) {
		ch.Routes = routes
	}
}

// ChannelOwners sets the list of owners for the channel.
func ChannelOwners(owners ...string) ChannelOption {
	return func(ch *Channel) {
		ch.Owners = owners
	}
}

// ChannelBadges sets visual badges on the channel.
func ChannelBadges(badges ...Badge) ChannelOption {
	return func(ch *Channel) {
		ch.Badges = badges
	}
}
