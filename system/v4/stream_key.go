package system

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// legacyStreamKey returns the pre-2026-10-17 storage key form: the stream
// ID's branded display form ("StreamMarker:…") instead of the bare value.
// Journals written before the disk-truth wave keyed their rows this way.
func legacyStreamKey(ref id.StreamRef) string {
	return ref.Type.String() + ":" + ref.ID.String()
}

// resolveStreamKey returns the storage key a stream's rows live under.
//
// Stream keys are BARE ("<type>:<id.Get()>") since the 2026-10-17
// disk-truth wave; journals written by older deployments keyed rows with
// the branded display form instead. Streams are STICKY: when rows already
// exist under the legacy key they stay under it — re-keying a live stream
// by appending under the bare form would split its history across two
// keys, and loads read exactly one key. Fresh streams always start bare.
//
// Resolution costs one indexed COUNT against the legacy form (zero on
// fresh databases); once a deployment's journals are all bare, the probe
// is a no-op returning the bare key.
func resolveStreamKey(
	ctx context.Context,
	backend metaengine.StreamLogBackend,
	collection string,
	ref id.StreamRef,
) (string, error) {
	bare := ref.StreamKey()
	legacy := legacyStreamKey(ref)

	if legacy == bare {
		return bare, nil
	}

	count, err := backend.StreamVersion(ctx, collection, legacy)
	if err != nil {
		return "", err
	}

	if count > 0 {
		return legacy, nil
	}

	return bare, nil
}
