package metaengine

import (
	"context"
	"encoding/json"
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
// Metadata is optional per-embedding filter data (e.g. {"tenant": "acme"})
// consumed by metadata-filtered k-NN (VectorFilterBackend).
type Embedding struct {
	ID       string
	Values   []float32
	Metadata map[string]any
}

// VectorResult is a single neighbor in a k-NN search result.
type VectorResult struct {
	ID       string
	Distance float32
}

// VectorFilter is a metadata predicate for filtered k-NN: the search only
// scores embeddings whose Metadata[Field] satisfies Op against Value. Ops are
// the standard FilterOp constants (FilterEq, FilterNe, FilterLt, FilterLe,
// FilterGt, FilterGe, FilterIn). All filters must match (AND semantics).
type VectorFilter struct {
	Field string
	Op    FilterOp
	Value any
}

// VectorMatchesFilters reports whether an embedding's metadata satisfies
// every filter (AND). Missing fields match nothing except FilterNe. Engines
// with brute-force VectorSearchFiltered implementations call this so filter
// semantics are identical across engines.
func VectorMatchesFilters(meta map[string]any, filters []VectorFilter) bool {
	for _, f := range filters {
		if !evalFilterOp(f.Op, meta[f.Field], f.Value) {
			return false
		}
	}

	return true
}

// VectorBackend is an optional engine capability for vector similarity
// search. Implementations may use brute-force (Memory), HNSW, or product
// quantization. The backend handles insertion and k-NN queries.
type VectorBackend interface {
	// VectorInsert adds an embedding to the collection's vector index.
	VectorInsert(ctx context.Context, collection string, emb Embedding) error

	// VectorSearch returns the k nearest neighbors of the query vector
	// using the given distance metric (cosine, euclidean, dot).
	VectorSearch(
		ctx context.Context,
		collection string,
		query []float32,
		k int,
		metric string,
	) ([]VectorResult, error)
}

// VectorFilterBackend is an optional extension of VectorBackend for
// metadata-filtered k-NN: filter + top-k in one query. Embeddings that do not
// match the filters are excluded BEFORE ranking, so the k results are the k
// nearest MATCHING neighbors (unlike post-filtering a bare top-k, which can
// return fewer than k results while matches exist). Engines that persist
// Embedding.Metadata implement this; engines without it fail filtered
// searches explicitly (results carry no metadata to filter on).
type VectorFilterBackend interface {
	VectorBackend

	// VectorSearchFiltered returns the k nearest neighbors whose metadata
	// matches every filter (AND semantics). An empty filter slice behaves
	// exactly like VectorSearch.
	VectorSearchFiltered(
		ctx context.Context,
		collection string,
		query []float32,
		k int,
		metric string,
		filters []VectorFilter,
	) ([]VectorResult, error)
}

// VectorCounter is an optional extension of VectorBackend for size
// introspection: counting a collection's embeddings and enumerating vector
// collections WITHOUT transferring payloads. Doctor uses it to report
// collection sizes; its absence triggers the full-scan WARN — an engine
// without it serves k-NN by scanning and cannot say how large the scan is.
type VectorCounter interface {
	VectorBackend

	// VectorCount returns the number of embeddings stored for the
	// collection (0 for an unknown collection).
	VectorCount(ctx context.Context, collection string) (int64, error)

	// VectorCollections returns the names of collections holding at least
	// one embedding.
	VectorCollections(ctx context.Context) ([]string, error)
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

// memoryVectorEntry is one stored embedding: its dimensions plus optional
// filter metadata.
type memoryVectorEntry struct {
	values   []float32
	metadata map[string]any
}

// MemoryVectorIndex is a brute-force in-memory vector index. It computes
// distances on every search — O(N*D) per query. Collections are isolated
// namespaces: the same embedding ID in two collections is two entries.
// Suitable for small collections (<10K vectors) or testing. For production
// scale, use an engine with ANN search (HNSW, PQ).
type MemoryVectorIndex struct {
	embeddings map[string]map[string]memoryVectorEntry // collection → id → entry
}

// NewMemoryVectorIndex creates a brute-force vector index.
func NewMemoryVectorIndex() *MemoryVectorIndex {
	return &MemoryVectorIndex{embeddings: make(map[string]map[string]memoryVectorEntry)}
}

func (m *MemoryVectorIndex) collection(col string) map[string]memoryVectorEntry {
	if m.embeddings[col] == nil {
		m.embeddings[col] = make(map[string]memoryVectorEntry)
	}

	return m.embeddings[col]
}

// Insert adds an embedding to the index (upsert by collection+ID).
func (m *MemoryVectorIndex) Insert(_ context.Context, collection string, emb Embedding) error {
	m.collection(collection)[emb.ID] = memoryVectorEntry{values: emb.Values, metadata: emb.Metadata}

	return nil
}

// Search returns the k nearest neighbors of the query vector.
func (m *MemoryVectorIndex) Search(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]VectorResult, error) {
	return m.search(collection, query, k, metric, nil), nil
}

// SearchFiltered returns the k nearest neighbors whose metadata matches all
// filters. Implements the filter semantics of VectorFilterBackend.
func (m *MemoryVectorIndex) SearchFiltered(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []VectorFilter,
) ([]VectorResult, error) {
	return m.search(collection, query, k, metric, filters), nil
}

// Count returns the number of embeddings stored for the collection.
// Implements the count member of VectorCounter.
func (m *MemoryVectorIndex) Count(_ context.Context, collection string) (int64, error) {
	return int64(len(m.embeddings[collection])), nil
}

// Collections returns the collection names holding at least one embedding.
// Implements the enumeration member of VectorCounter.
func (m *MemoryVectorIndex) Collections(_ context.Context) ([]string, error) {
	out := make([]string, 0, len(m.embeddings))

	for col, entries := range m.embeddings {
		if len(entries) > 0 {
			out = append(out, col)
		}
	}

	sort.Strings(out)

	return out, nil
}

func (m *MemoryVectorIndex) search(
	collection string,
	query []float32,
	k int,
	metric string,
	filters []VectorFilter,
) []VectorResult {
	var results []VectorResult

	for id, entry := range m.embeddings[collection] {
		if !VectorMatchesFilters(entry.metadata, filters) {
			continue
		}

		dist := computeDistance(query, entry.values, metric)
		results = append(results, VectorResult{ID: id, Distance: dist})
	}

	return TopKNearest(results, k)
}

// VectorDistance returns the distance between two vectors under the given
// metric ("cosine", "dot", "euclidean"; "" defaults to euclidean). Engines
// with brute-force VectorSearch implementations call this so results are
// numerically identical across engines (the adttest matrix asserts parity).
func VectorDistance(a, b []float32, metric string) float32 {
	return computeDistance(a, b, metric)
}

// DecodeVectorJSON decodes a JSON-encoded embedding ([]float32) — the
// storage format used by KV/LSM engines' brute-force vector backends.
func DecodeVectorJSON(data []byte) ([]float32, error) {
	var vec []float32
	if err := json.Unmarshal(data, &vec); err != nil {
		return nil, err
	}

	return vec, nil
}

// TopKNearest sorts results ascending by distance (the "dot" metric is
// negated by VectorDistance so ascending is always nearest-first) and
// truncates to k. Shared by engines' brute-force VectorSearch paths so
// truncation semantics are identical.
func TopKNearest(results []VectorResult, k int) []VectorResult {
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
func (m *MemorySearchIndex) Insert(_ context.Context, _ string, doc IndexedText) error {
	m.docs[doc.ID] = doc.Content

	return nil
}

// Query returns documents matching the full-text query.
func (m *MemorySearchIndex) Query(
	_ context.Context,
	_, query string,
	limit int,
) ([]SearchResult, error) {
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
