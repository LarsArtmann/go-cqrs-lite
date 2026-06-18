package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Validator validates event payloads against registered Go types.
// It works by attempting to unmarshal the event payload into the registered
// type for that event type. If unmarshaling fails, the event is rejected.
//
// This is the runtime validation layer complementing the schema evolution
// provided by Upcaster and VersionedStore. Together they ensure:
//   - Validator: rejects structurally invalid events before they enter the system
//   - Upcaster/VersionedStore: transforms old events to current schema on read
//
// Usage:
//
//	v := schema.NewValidator()
//	schema.RegisterType[UserCreated](v, "user.created")
//	schema.RegisterTypeWithValidator(v, "user.updated",
//	    func(u UserUpdated) error {
//	        if u.Name == "" { return errors.New("name required") }
//	        return nil
//	    })
//	bus.Use(middleware.EventValidation(v.Validate))
type Validator struct {
	mu         sync.RWMutex
	types      map[event.Type]reflect.Type
	validators map[event.Type]func(any) error
	decode     func([]byte, any) error
	strict     bool
}

// ValidatorOption configures a Validator.
type ValidatorOption func(*Validator)

// WithCodec sets a custom decode function for payload decoding.
// Defaults to encoding/json.Unmarshal. Use this for CBOR or other codecs:
//
//	v := schema.NewValidator(schema.WithCodec(cborCodec.Decode))
func WithCodec(decode func([]byte, any) error) ValidatorOption {
	return func(v *Validator) {
		v.decode = decode
	}
}

// WithStrictMode rejects events whose type is not registered with the validator.
// By default, unregistered event types pass validation (lenient mode).
func WithStrictMode() ValidatorOption {
	return func(v *Validator) {
		v.strict = true
	}
}

// NewValidator creates a new schema Validator with the given options.
func NewValidator(opts ...ValidatorOption) *Validator {
	v := &Validator{
		types:      make(map[event.Type]reflect.Type),
		validators: make(map[event.Type]func(any) error),
		decode:     json.Unmarshal,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// RegisterType registers a Go type for an event type. Events of this type
// will be validated by unmarshaling their payload into a new instance of T.
func RegisterType[T any](v *Validator, eventType event.Type) {
	v.RegisterType(eventType, reflect.TypeFor[T]())
}

// RegisterTypeWithValidator registers a Go type and a custom validation function.
// After successful unmarshaling, the function is called with the decoded value
// for additional business-rule validation.
func RegisterTypeWithValidator[T any](
	v *Validator,
	eventType event.Type,
	validate func(T) error,
) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.types[eventType] = reflect.TypeFor[T]()

	v.validators[eventType] = func(val any) error {
		typed, ok := val.(T)
		if !ok {
			return fmt.Errorf("type assertion failed: expected %T", typed)
		}

		return validate(typed)
	}
}

// RegisterType registers a reflect.Type for an event type.
func (v *Validator) RegisterType(eventType event.Type, t reflect.Type) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.types[eventType] = t
}

// Validate validates an event against its registered schema.
// Returns nil if valid, a Rejection error if invalid.
// Unregistered event types pass validation unless strict mode is enabled.
func (v *Validator) Validate(evt event.Event) error {
	v.mu.RLock()
	t, ok := v.types[evt.Type()]
	customValidate := v.validators[evt.Type()]
	v.mu.RUnlock()

	if !ok {
		if v.strict {
			return event.WrapRejection(
				fmt.Errorf("no schema registered for event type %q", evt.Type()),
				"schema.unregistered_type",
				"strict mode: unregistered event type",
			)
		}

		return nil
	}

	payload := evt.Payload()
	if len(payload) == 0 {
		return nil
	}

	instance := reflect.New(t).Interface()

	err := v.decode(payload, instance)
	if err != nil {
		return event.WrapRejection(
			err,
			"schema.decode_failed",
			fmt.Sprintf("payload does not conform to schema for %q", evt.Type()),
		)
	}

	if customValidate != nil {
		val := reflect.ValueOf(instance)
		if val.Kind() == reflect.Pointer && !val.IsNil() {
			val = val.Elem()
		}

		err := customValidate(val.Interface())
		if err != nil {
			return event.WrapRejection(
				err,
				"schema.validation_failed",
				fmt.Sprintf("validation failed for %q: %s", evt.Type(), err),
			)
		}
	}

	return nil
}

// RegisteredTypes returns the event types currently registered with the validator.
func (v *Validator) RegisteredTypes() []event.Type {
	v.mu.RLock()
	defer v.mu.RUnlock()

	types := make([]event.Type, 0, len(v.types))
	for t := range v.types {
		types = append(types, t)
	}

	return types
}
