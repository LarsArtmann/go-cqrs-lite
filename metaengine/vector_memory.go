package metaengine

import (
	"context"
	"sort"
	"sync"
)

// --- Memory implementation (brute-force, dimension-locked) ---
//
// Lives here instead of vector_search.go because vector_search.go sits on
// the file-size ratchet baseline (growth forbidden); this extraction also
// brings the brute-force index under the 350-line budget.

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
	mu         sync.RWMutex
	embeddings map[string]map[string]memoryVectorEntry // collection → id → entry
	dims       map[string]int                          // collection → established dimension
}

// NewMemoryVectorIndex creates a brute-force vector index.
func NewMemoryVectorIndex() *MemoryVectorIndex {
	return &MemoryVectorIndex{
		embeddings: make(map[string]map[string]memoryVectorEntry),
		dims:       make(map[string]int),
	}
}

// collectionLocked returns or creates a map collection. Caller MUST hold
// m.mu.Lock().
func (m *MemoryVectorIndex) collectionLocked(col string) map[string]memoryVectorEntry {
	if m.embeddings[col] == nil {
		m.embeddings[col] = make(map[string]memoryVectorEntry)
	}

	return m.embeddings[col]
}

// Insert adds an embedding to the index (upsert by collection+ID),
// enforcing the collection's dimension lock (first insert establishes the
// dimension; mismatching or zero-dimension inserts are rejected).
func (m *MemoryVectorIndex) Insert(_ context.Context, collection string, emb Embedding) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := CheckVectorDimension(collection, m.dims[collection], len(emb.Values)); err != nil {
		return err
	}

	m.dims[collection] = len(emb.Values)
	m.collectionLocked(collection)[emb.ID] = memoryVectorEntry{
		values:   emb.Values,
		metadata: emb.Metadata,
	}

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
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.searchLocked(collection, query, k, metric, nil), nil
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.searchLocked(collection, query, k, metric, filters), nil
}

// Count returns the number of embeddings stored for the collection.
// Implements the count member of VectorCounter.
func (m *MemoryVectorIndex) Count(_ context.Context, collection string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return int64(len(m.embeddings[collection])), nil
}

// Collections returns the collection names holding at least one embedding.
// Implements the enumeration member of VectorCounter.
func (m *MemoryVectorIndex) Collections(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]string, 0, len(m.embeddings))

	for col, entries := range m.embeddings {
		if len(entries) > 0 {
			out = append(out, col)
		}
	}

	sort.Strings(out)

	return out, nil
}

// searchLocked scans the collection. Caller MUST hold m.mu (RLock is
// sufficient — it only reads).
func (m *MemoryVectorIndex) searchLocked(
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
