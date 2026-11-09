package testdata

// Permanent mutation fixture for the aggregate_* family-code tripwire
// (TestAggregateTripwireScannerBites). testdata/ is skipped by the repo-wide
// walk, so the planted strings here only bite through the self-assert.

const plantedReintroduction = "event.aggregate_not_found"

func plantedParseCode() string {
	return "storage.parse_aggregate_id"
}

// Negative control: a legitimate aggregate identifier that the exact-string
// table must NOT fire on (broad-substring sweeps would false-positive here).
const legitimateProjectionName = "listing.aggregate_projection"
