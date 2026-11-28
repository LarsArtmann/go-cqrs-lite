package search

import (
	"context"
	"net/url"
	"strings"
)

// Capabilities reports which kinds of search a deployment can serve.
type Capabilities struct {
	// FullText is true when the engine keeps a full-text index for the
	// searched records.
	FullText bool
	// Vector is true when the engine keeps a vector index for them.
	Vector bool
}

// None is the capability set of a deployment without a search engine: the
// lexical scan is all it can serve.
var None = Capabilities{}

// Embedder turns text into vectors. The operator supplies it; the SDK never
// embeds without one, so a nil embedder means lexical search only.
type Embedder interface {
	// Name is the namespace of the vector cache.
	Name() string
	// Model names the model behind the vectors. A change makes every stored
	// vector stale; the SDK reports it and never mixes vectors from two models.
	Model() string
	// Embed turns texts into vectors, one per text, in order.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Engine reports what the deployment offers. An [Index] asks it before it
// plans; execution methods for the full-text and vector tiers are added to
// it by the engines that serve those tiers.
type Engine interface {
	// Capabilities reports which tiers the engine can serve.
	Capabilities() Capabilities
	// Embedder returns the operator's embedder, or nil when the deployment
	// runs lexical search only.
	Embedder() Embedder
}

// NoEngine is the engine of a deployment that only scans. It has no
// capabilities and no embedder, so [New] never needs a nil engine.
var NoEngine Engine = noEngine{}

type noEngine struct{}

func (noEngine) Capabilities() Capabilities { return None }
func (noEngine) Embedder() Embedder         { return nil }

// Listing is one page of records from a [Source].
type Listing[T any] struct {
	// Items are the records, at most as many as the requested limit.
	Items []T
	// Truncated is true when the source stopped at its limit before it had
	// seen every record.
	Truncated bool
}

// Source is how an [Index] reads records. A product writes one over its read
// model or its store; it never interprets a query.
type Source[T any] interface {
	// List returns the records that match the facets, in the source's own
	// order, at most limit of them. Filters are the request's facets,
	// possibly empty; nil and empty mean the same. A source that cannot
	// honour the limit exactly reports Truncated when it may have left
	// records behind.
	List(ctx context.Context, filters url.Values, limit int) (Listing[T], error)
	// ByIDs returns the records with the given identifiers, in no particular
	// order. The full-text and vector tiers use it to hydrate their matches.
	ByIDs(ctx context.Context, ids []string) ([]T, error)
	// ID returns the identifier of a record.
	ID(item T) string
}

// Tier resolves the tier a request runs on. It is a pure function of the
// mode, the free terms, the capabilities and the presence of an embedder.
//
// An explicit mode is strict: a tier the deployment cannot serve is
// [ErrModeUnavailable]. Auto never fails; it applies the five rules:
//
//  1. Fewer than two characters of terms: no text search ([ViaNone]).
//  2. Identifier-like text: lexical.
//  3. A single token of at most six characters: lexical.
//  4. Otherwise, with an embedder: hybrid, or vector without a full-text index.
//
// Rule 5, the breaker after embedder failures, is applied at execution time
// by the Index, not here.
func Tier(mode Mode, terms []string, caps Capabilities, hasEmbedder bool) (Via, error) {
	if !mode.IsValid() {
		return ViaNone, rejectionWith(ErrModeUnavailable,
			"the mode "+mode.String()+" is not one of auto, lexical, fulltext, vector, hybrid")
	}
	if mode == ModeAuto {
		return autoTier(terms, caps, hasEmbedder), nil
	}
	switch mode {
	case ModeLexical:
		if caps.FullText {
			return ViaFullText, nil
		}
		return ViaScan, nil
	case ModeFullText:
		if !caps.FullText {
			return ViaNone, modeUnavailable(mode, "no full-text index on the engine")
		}
		return ViaFullText, nil
	case ModeVector:
		if !caps.Vector || !hasEmbedder {
			return ViaNone, modeUnavailable(mode, "the deployment has no vector index or no embedder")
		}
		return ViaVector, nil
	case ModeHybrid:
		if !caps.FullText || !caps.Vector || !hasEmbedder {
			return ViaNone, modeUnavailable(mode,
				"the deployment has no full-text index, no vector index or no embedder")
		}
		return ViaHybrid, nil
	default:
		return ViaNone, ErrModeUnavailable
	}
}

// autoTier applies the rules of [ModeAuto].
func autoTier(terms []string, caps Capabilities, hasEmbedder bool) Via {
	length := 0
	for _, term := range terms {
		length += len([]rune(term))
	}
	if length < minQueryRunes {
		return ViaNone
	}
	lexical := ViaScan
	if caps.FullText {
		lexical = ViaFullText
	}
	if IsIdentifierLike(strings.Join(terms, " ")) || !isProseWorthEmbedding(terms) || !hasEmbedder || !caps.Vector {
		return lexical
	}
	if caps.FullText {
		return ViaHybrid
	}
	return ViaVector
}

// minQueryRunes is the shortest text that starts a text search (Auto rule 1).
const minQueryRunes = 2

// modeUnavailable builds the strict-mode rejection with its reason.
func modeUnavailable(mode Mode, reason string) error {
	return rejectionWith(ErrModeUnavailable,
		"mode "+mode.String()+" needs a capability this deployment lacks: "+reason)
}
