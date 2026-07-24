package benchkit

// Profile defines the scale parameters for a benchmark run.
type Profile struct {
	// Name is the human-readable identifier (e.g. "dev", "medium").
	Name string

	// Streams is the number of distinct stream IDs to write.
	Streams int

	// EventsPerStream is the number of events written per stream.
	EventsPerStream int

	// Concurrency is the number of parallel goroutines for write/read.
	Concurrency int

	// ReadRatio is the fraction of operations that are reads vs writes.
	// 0.0 = write-only, 1.0 = read-only. Applied during the mixed read phase.
	ReadRatio float64

	// BatchSize is the number of events per Save() call.
	// 1 = single-event saves. Higher values test batch write performance.
	BatchSize int
}

// TotalEvents returns Streams * EventsPerStream.
func (p Profile) TotalEvents() int { return p.Streams * p.EventsPerStream }

// Named workload profiles representing realistic deployment sizes.
// Results from the same profile are comparable across backends and runs.
var (
	// ProfileDev is a quick smoke test (~500 events). Runs in under a second
	// on any backend. Use for CI regression checks.
	ProfileDev = Profile{
		Name: "dev", Streams: 100, EventsPerStream: 5,
		Concurrency: 1, ReadRatio: 0.2, BatchSize: 1,
	}

	// ProfileSmall represents a small deployment (~10K events). Use for
	// development and testing with realistic but fast workloads.
	ProfileSmall = Profile{
		Name: "small", Streams: 1000, EventsPerStream: 10,
		Concurrency: 4, ReadRatio: 0.3, BatchSize: 1,
	}

	// ProfileMedium represents a mid-size deployment (~500K events).
	// The default for backend comparison — large enough to surface latency
	// tails, small enough to finish in seconds.
	ProfileMedium = Profile{
		Name: "medium", Streams: 10_000, EventsPerStream: 50,
		Concurrency: 16, ReadRatio: 0.4, BatchSize: 5,
	}

	// ProfileLarge represents a production-scale deployment (~10M events).
	// Requires significant disk space and time. Use for pre-production
	// capacity planning.
	ProfileLarge = Profile{
		Name: "large", Streams: 100_000, EventsPerStream: 100,
		Concurrency: 32, ReadRatio: 0.5, BatchSize: 10,
	}

	// ProfileStress tests write-heavy throughput under high concurrency
	// (~5M events, 64 goroutines). Use to find the write ceiling.
	ProfileStress = Profile{
		Name: "stress", Streams: 10_000, EventsPerStream: 500,
		Concurrency: 64, ReadRatio: 0.2, BatchSize: 1,
	}

	// ProfileWriteHeavy isolates write performance (~1M events, 90% writes).
	ProfileWriteHeavy = Profile{
		Name: "write-heavy", Streams: 10_000, EventsPerStream: 100,
		Concurrency: 32, ReadRatio: 0.1, BatchSize: 1,
	}

	// ProfileReadHeavy isolates read performance (~1M events, 80% reads).
	ProfileReadHeavy = Profile{
		Name: "read-heavy", Streams: 10_000, EventsPerStream: 100,
		Concurrency: 32, ReadRatio: 0.8, BatchSize: 1,
	}
)

// ProfileByName looks up a named profile. Returns the profile and true if
// found, or ProfileDev and false otherwise.
func ProfileByName(name string) (Profile, bool) {
	switch name {
	case "dev":
		return ProfileDev, true
	case "small":
		return ProfileSmall, true
	case "medium":
		return ProfileMedium, true
	case "large":
		return ProfileLarge, true
	case "stress":
		return ProfileStress, true
	case "write-heavy":
		return ProfileWriteHeavy, true
	case "read-heavy":
		return ProfileReadHeavy, true
	default:
		return ProfileDev, false
	}
}
