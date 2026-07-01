package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
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
	decoders   map[codec.Encoding]func([]byte, any) error
	strict     bool
}

// ValidatorOption configures a Validator.
type ValidatorOption func(*Validator)

// WithCodec configures the validator to use the given codec's Decode function
// for events whose encoding matches the codec's Encoding(). For example,
// passing codec.JSONCodec{} overrides the default JSON decoder with the codec's
// own implementation, and passing codec.CBORCodec{} overrides the CBOR decoder.
//
// To register decoders for multiple encodings, call WithCodec once per codec or
// use WithDecoder for individual encodings.
func WithCodec(c codec.Codec) ValidatorOption {
	return func(v *Validator) {
		if c != nil {
			v.decoders[c.Encoding()] = c.Decode
		}
	}
}

// WithDecodeFunc overrides the default JSON decode function and the fallback
// decoder for unknown encodings. This is a low-level escape hatch for callers
// who need to inject a raw function rather than a full codec.Codec.
//
// Deprecated: Use WithCodec(codec.JSONCodec{}) or WithDecoder(codec.EncodingJSON, fn)
// for type safety. WithDecodeFunc will be removed in v4.
func WithDecodeFunc(decode func([]byte, any) error) ValidatorOption {
	return func(v *Validator) {
		v.decode = decode
		v.decoders[codec.EncodingJSON] = decode
	}
}

// WithStrictMode rejects events whose type is not registered with the validator.
// By default, unregistered event types pass validation (lenient mode).
func WithStrictMode() ValidatorOption {
	return func(v *Validator) {
		v.strict = true
	}
}

// WithDecoder registers a decode function for a specific encoding. Use this when
// you need a non-standard codec for a specific encoding (e.g. a custom CBOR
// decoder with different options):
//
//	v := schema.NewValidator(schema.WithDecoder(codec.EncodingCBOR, myCBORDecoder))
func WithDecoder(enc codec.Encoding, decode func([]byte, any) error) ValidatorOption {
	return func(v *Validator) {
		if decode != nil {
			v.decoders[enc] = decode
		}
	}
}

// NewValidator creates a new schema Validator with the given options.
// The validator auto-detects event payload encoding via evt.Encoding() and
// picks the matching decoder — JSON and CBOR work out of the box.
func NewValidator(opts ...ValidatorOption) *Validator {
	cborCodec := codec.CBORCodec{}

	v := &Validator{
		types:      make(map[event.Type]reflect.Type),
		validators: make(map[event.Type]func(any) error),
		decode:     json.Unmarshal,
		decoders: map[codec.Encoding]func([]byte, any) error{
			codec.EncodingJSON: json.Unmarshal,
			codec.EncodingCBOR: cborCodec.Decode,
		},
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
			return errTypeAssertion
		}

		return validate(typed)
	}
}

var errTypeAssertion = errors.New("type assertion failed during validation")

var errUnregisteredType = errors.New("no schema registered for event type")

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
				errUnregisteredType,
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

	decode := v.decoderFor(evt.Encoding())

	instance := reflect.New(t).Interface()

	err := decode(payload, instance)
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

// decoderFor selects the decode function based on the event's declared encoding.
// Falls back to the default JSON decoder for unknown encodings.
func (v *Validator) decoderFor(enc codec.Encoding) func([]byte, any) error {
	if dec, ok := v.decoders[enc]; ok {
		return dec
	}

	return v.decode
}
