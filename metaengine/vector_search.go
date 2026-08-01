package metaengine

import (
	"context"
	"math"
	"sort"
)

// ADTVector is the ADT for vector similarity search (k-NN).
// Engines implement VectorBackend to support vector queries.
const ADTVector ADT = "vector"

// ADTSearch is the ADT for full-text search (inverted index).
// Engines implement SearchBackend to support text search queries.
const ADTSearch ADT = "search"

// Embedding is a vector with an ID and optional metadata. The Values field
// holds the float dimensions. This is the fold input type for vector
// projections — an event carrying an Embedding adds it to the index.
type Embedding struct {
	ID     string
	Values []float32
}

// VectorResult is a single neighbor in a k-NN search result.
type VectorResult struct {
	ID       string
	Distance float32
}

// VectorBackend is an optional engine capability for vector similarity
// search. Implementations may use brute-force (Memory), HNSW, or product
// quantization. The backend handles insertion and k-NN queries.
type VectorBackend interface {
	// VectorInsert adds an embedding to the collection's vector index.
	VectorInsert(ctx context.Context, collection string, emb Embedding) error

	// VectorSearch returns the k nearest neighbors of the query vector
	// using the given distance metric (cosine, euclidean, dot).
	VectorSearch(ctx context.Context, collection string, query []float32, k int, metric string) ([]VectorResult, error)
}

// IndexedText is a text document with an ID and content. The fold input
// type for search projections — an event carrying IndexedText adds it to
// the full-text index.
type IndexedText struct {
	ID      string
	Content string
}

// SearchResult is a single match in a full-text search.
type SearchResult struct {
	ID    string
	Score float64
}

// SearchBackend is an optional engine capability for full-text search.
// Implementations may use an inverted index (Memory), BM25, or trigram.
// The backend handles insertion and text queries.
type SearchBackend interface {
	// SearchInsert adds a document to the collection's search index.
	SearchInsert(ctx context.Context, collection string, doc IndexedText) error

	// SearchQuery returns documents matching the full-text query string.
	// Results are ranked by relevance score (TF-IDF or BM25).
	SearchQuery(ctx context.Context, collection, query string, limit int) ([]SearchResult, error)
}

// --- Memory implementations (brute-force) ---

// MemoryVectorIndex is a brute-force in-memory vector index. It computes
// distances on every search — O(N*D) per query. Suitable for small
// collections (<10K vectors) or testing. For production scale, use an
// engine with ANN (HNSW, PQ).
type MemoryVectorIndex struct {
	embeddings map[string][]float32 // key → values
}

// NewMemoryVectorIndex creates a brute-force vector index.
func NewMemoryVectorIndex() *MemoryVectorIndex {
	return &MemoryVectorIndex{embeddings: make(map[string][]float32)}
}

// Insert adds an embedding to the index.
func (m *MemoryVectorIndex) Insert(ctx context.Context, collection string, emb Embedding) error {
	m.embeddings[emb.ID] = emb.Values

	return nil
}

// Search returns the k nearest neighbors of the query vector.
func (m *MemoryVectorIndex) Search(ctx context.Context, collection string, query []float32, k int, metric string) ([]VectorResult, error) {
	return m.search(query, k, metric), nil
}

func (m *MemoryVectorIndex) search(query []float32, k int, metric string) []VectorResult {
	var results []VectorResult

	for id, vec := range m.embeddings {
		dist := computeDistance(query, vec, metric)
		results = append(results, VectorResult{ID: id, Distance: dist})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > 0 && k < len(results) {
		results = results[:k]
	}

	return results
}

func computeDistance(a, b []float32, metric string) float32 {
	switch metric {
	case "cosine":
		return cosineDistance(a, b)
	case "dot":
		return -dotProduct(a, b) // negate so sort ascending = highest dot first
	case "euclidean", "":
		return euclideanDistance(a, b)
	default:
		return euclideanDistance(a, b)
	}
}

func cosineDistance(a, b []float32) float32 {
	dot := float32(0)
	normA := float32(0)
	normB := float32(0)

	for i := range a {
		if i >= len(b) {
			break
		}

		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1
	}

	return 1 - dot/float32(math.Sqrt(float64(normA*normB)))
}

func dotProduct(a, b []float32) float32 {
	var sum float32

	for i := range a {
		if i >= len(b) {
			break
		}

		sum += a[i] * b[i]
	}

	return sum
}

func euclideanDistance(a, b []float32) float32 {
	var sum float32

	for i := range a {
		if i >= len(b) {
			break
		}

		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// MemorySearchIndex is a brute-force in-memory full-text search index
// using a simple TF-IDF scoring. Suitable for small collections or testing.
type MemorySearchIndex struct {
	docs map[string]string // key → content
}

// NewMemorySearchIndex creates a brute-force search index.
func NewMemorySearchIndex() *MemorySearchIndex {
	return &MemorySearchIndex{docs: make(map[string]string)}
}

// Insert adds a document to the search index.
func (m *MemorySearchIndex) Insert(ctx context.Context, collection string, doc IndexedText) error {
	m.docs[doc.ID] = doc.Content

	return nil
}

// Query returns documents matching the full-text query.
func (m *MemorySearchIndex) Query(ctx context.Context, collection, query string, limit int) ([]SearchResult, error) {
	return m.query(query, limit), nil
}

func (m *MemorySearchIndex) query(query string, limit int) []SearchResult {
	queryTerms := tokenize(query)
	var results []SearchResult

	for id, content := range m.docs {
		contentTerms := tokenize(content)
		score := tfidfScore(queryTerms, contentTerms)
		if score > 0 {
			results = append(results, SearchResult{ID: id, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results
}

func tokenize(s string) []string {
	var tokens []string

	start := -1

	for i, c := range s {
		if isAlphaNumeric(c) {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				tokens = append(tokens, toLower(s[start:i]))
				start = -1
			}
		}
	}

	if start != -1 {
		tokens = append(tokens, toLower(s[start:]))
	}

	return tokens
}

func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func toLower(s string) string {
	b := []byte(s)

	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}

	return string(b)
}

func tfidfScore(queryTerms, contentTerms []string) float64 {
	contentFreq := make(map[string]int)
	for _, t := range contentTerms {
		contentFreq[t]++
	}

	var score float64

	for _, qt := range queryTerms {
		if freq, ok := contentFreq[qt]; ok {
			score += float64(freq)
		}
	}

	return score
}
