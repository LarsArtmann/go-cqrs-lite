package metaengine

import (
	"context"
	"math"
	"sort"
)

// ADTSpatial is the ADT for geographic/geometric range queries.
// Engines implement SpatialBackend to support spatial proximity queries.
const ADTSpatial ADT = "spatial"

// Point is a 2D geographic coordinate for spatial proximity queries.
// X = longitude, Y = latitude. The Memory engine uses haversine great-circle
// distance (meters). Future engines may support Cartesian/euclidean mode.
type Point struct {
	ID string
	X  float64 // longitude
	Y  float64 // latitude
}

// SpatialResult is a single match in a spatial range query.
type SpatialResult struct {
	ID       string
	Distance float64 // meters (haversine great-circle distance)
}

// SpatialBackend is an optional engine capability for spatial proximity
// queries. Implementations may use brute-force (Memory), R-trees, or
// geohash-based indexes.
type SpatialBackend interface {
	// SpatialInsert adds a point to the collection's spatial index.
	SpatialInsert(ctx context.Context, collection string, pt Point) error

	// SpatialRange returns points within radius distance of (x, y).
	// Uses haversine distance for geographic coordinates (lat/long).
	// Results are sorted by distance (nearest first).
	SpatialRange(ctx context.Context, collection string, x, y, radius float64, limit int) ([]SpatialResult, error)
}

// --- Memory implementation (brute-force) ---

// MemorySpatialIndex is a brute-force in-memory spatial index. It computes
// haversine distance for every range query — O(N) per query. Suitable for
// small collections or testing. For production scale, use an R-tree engine.
type MemorySpatialIndex struct {
	points map[string]Point // key → point
}

// NewMemorySpatialIndex creates a brute-force spatial index.
func NewMemorySpatialIndex() *MemorySpatialIndex {
	return &MemorySpatialIndex{points: make(map[string]Point)}
}

// Insert adds a point to the spatial index.
func (m *MemorySpatialIndex) Insert(ctx context.Context, collection string, pt Point) error {
	m.points[pt.ID] = pt
	return nil
}

// Range returns points within the given radius of (x, y), sorted by distance.
func (m *MemorySpatialIndex) Range(ctx context.Context, collection string, x, y, radius float64, limit int) ([]SpatialResult, error) {
	return m.rangeQuery(x, y, radius, limit), nil
}

func (m *MemorySpatialIndex) rangeQuery(x, y, radius float64, limit int) []SpatialResult {
	var results []SpatialResult

	for id, pt := range m.points {
		dist := haversineDistance(y, x, pt.Y, pt.X)
		if dist <= radius {
			results = append(results, SpatialResult{ID: id, Distance: dist})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results
}

// haversineDistance computes the great-circle distance between two
// lat/long points in meters. lat1/lat2 are latitudes, lon1/lon2 are longitudes.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
