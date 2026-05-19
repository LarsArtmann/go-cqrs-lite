package docserver

import "embed"

// staticAssets contains the embedded documentation UI libraries.
//
//go:embed static/*
var staticAssets embed.FS
