package search

import "strings"

// DefaultFieldWeight is the weight of a field declared with Weight 0.
const DefaultFieldWeight = 1.0

// Field declares one searchable field of a record type.
//
// Weight orders fields against each other when a term matches several: a
// title at 1.0 outranks a body at 0.2. A Weight of 0 means [DefaultFieldWeight].
//
// Private marks a field that must never leave the machine: it is searched
// lexically and left out of every static index and of every text sent to an
// embedder.
type Field[T any] struct {
	Name    string
	Weight  float64
	Private bool
	Value   func(T) string
}

// resolvedWeight returns the weight with its zero value defaulted.
func (f Field[T]) resolvedWeight() float64 {
	if f.Weight == 0 {
		return DefaultFieldWeight
	}
	return f.Weight
}

// Spec declares how a record type is searched. An author writes one Spec and
// hands it to [New]; every later change goes through a new Spec.
type Spec[T any] struct {
	// Fields are the searchable fields. At least one is required.
	Fields []Field[T]
	// Mode selects the kind of search. The zero value is [ModeAuto].
	Mode Mode
	// MaxQueryRunes is the longest accepted query. 0 means [MaxQueryRunes].
	MaxQueryRunes int
	// ScanLimit bounds how many records the scan tier reads before the
	// result is reported as truncated. 0 means [DefaultScanLimit].
	ScanLimit int
	// Keys are the operator tokens the product understands, such as
	// "from" or "status". A key:value pair whose key is listed here becomes
	// an operator token; every other pair stays free text. What a key means
	// is the product's business: the Spec only splits it out.
	Keys []string
	// Sorts declares the sort keys a [Request] may name. Each comparator
	// returns negative when a sorts before b. An empty Request.Sort means
	// relevance, or source order when no text search ran.
	Sorts map[string]func(a, b T) int
}

// Validate reports the first reason the spec cannot build an index.
func (s Spec[T]) Validate() error {
	if len(s.Fields) == 0 {
		return ErrNoFields
	}
	seen := make(map[string]struct{}, len(s.Fields))
	for _, field := range s.Fields {
		switch {
		case strings.TrimSpace(field.Name) == "" || field.Value == nil:
			return ErrInvalidField
		case field.Weight < 0:
			return ErrInvalidWeight
		}
		if _, dup := seen[field.Name]; dup {
			return ErrDuplicateField
		}
		seen[field.Name] = struct{}{}
	}
	if !s.Mode.IsValid() {
		return rejectionWith(
			ErrModeUnavailable,
			"the mode "+s.Mode.String()+" is not one of auto, lexical, fulltext, vector, hybrid",
		)
	}
	if _, err := limitOrDefault(s.MaxQueryRunes, MaxQueryRunes); err != nil {
		return err
	}
	if _, err := limitOrDefault(s.ScanLimit, DefaultScanLimit); err != nil {
		return err
	}
	return nil
}

// maxQueryRunes returns the query length limit with its zero value defaulted.
func (s Spec[T]) maxQueryRunes() int {
	if s.MaxQueryRunes == 0 {
		return MaxQueryRunes
	}
	return s.MaxQueryRunes
}

// scanLimit returns the scan limit with its zero value defaulted.
func (s Spec[T]) scanLimit() int {
	if s.ScanLimit == 0 {
		return DefaultScanLimit
	}
	return s.ScanLimit
}

// Tokens splits a query into free terms and operator tokens, keeping the
// spec's declared [Spec.Keys]. It is [Tokenize] with the spec's keys.
func (s Spec[T]) Tokens(query string) (terms []string, tokens map[string][]string) {
	return Tokenize(query, s.Keys...)
}

// clone returns a copy that shares no slice or map with the receiver.
func (s Spec[T]) clone() Spec[T] {
	copied := s
	copied.Fields = append([]Field[T](nil), s.Fields...)
	copied.Keys = append([]string(nil), s.Keys...)
	if s.Sorts != nil {
		copied.Sorts = make(map[string]func(a, b T) int, len(s.Sorts))
		for key, less := range s.Sorts {
			copied.Sorts[key] = less
		}
	}
	return copied
}
