package architecture

import "embed"

// assetsFS holds the stylesheet and scripts that are inlined into every
// generated document, so the file needs no network and no build step.
//
//go:embed assets/reference.css assets/reference.js assets/theme.js
var assetsFS embed.FS

// asset returns an embedded document asset. The files are embedded at build
// time, so a miss can only mean the name was changed without this list.
func asset(name string) markup {
	data, err := assetsFS.ReadFile("assets/" + name)
	if err != nil {
		return ""
	}

	return markup(data)
}
