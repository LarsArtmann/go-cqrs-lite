#21
Fixed and released in watermill/v4.6.3: eventToMessage writes correlation_id/causation_id through the shared writeTracing, typed causation included. Round-trip pinned by TestEventToMessage_TypedCausationRoundtrip. Closing as fixed: v4.6.2 was the last affected tag.

#26
stack/postgres v4.4.2 is released. Retract directive for v4.2.0 is on master (stack/postgres/go.mod) and reaches the proxy with the next tagged release. Keeping this open until the retract is published.

#35
Shipped in event/v4.13.0: event.RequestScope + WithRequestScope + RequestScopeEnricher, recipe in the skill references (recipes 2.42). Closing as fixed.
