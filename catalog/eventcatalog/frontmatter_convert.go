package eventcatalog

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// --- Conversion helpers: catalog types → frontmatter types ---

func toPointers[S ~string](ids []S) []pointer {
	if len(ids) == 0 {
		return nil
	}

	out := make([]pointer, len(ids))
	for i, id := range ids {
		out[i] = pointer{ID: string(id)}
	}

	return out
}

// toChannelRefs converts channel IDs to EventCatalog channelPointer objects
// ({id, version?}). Plain strings fail schema validation
// ("Expected type object, received string").
func toChannelRefs(
	ids []catalog.ChannelID,
	versions map[catalog.ChannelID]catalog.Version,
) []channelRefFM {
	if len(ids) == 0 {
		return nil
	}

	out := make([]channelRefFM, len(ids))
	for i, id := range ids {
		ref := channelRefFM{ID: string(id)}
		if v, ok := versions[id]; ok {
			ref.Version = string(v)
		}
		out[i] = ref
	}

	return out
}

// EventCatalog's message schema declares producers/consumers as plain string
// references into the services collection. The DEFAULT form is the
// composite "<serviceID>-<serviceVersion>" Astro entry ID: @eventcatalog/core's
// content layer (and its visualiser graph edges) resolves exactly that key,
// and a bare service ID makes its build log `Invalid content reference`.
// The composite form is invisible to @eventcatalog/linter, which indexes by
// frontmatter ID — WithPlainRefIDs flips this function to bare IDs for
// governance/lint exports (see the option's doc comment).
// Unknown services fall back to the bare ID (the reference stays resolvable
// once a service with that ID exists).
func toServiceRefs(
	ids []catalog.ServiceID,
	versions map[catalog.ServiceID]catalog.Version,
	plain bool,
) []string {
	if len(ids) == 0 {
		return nil
	}

	out := make([]string, len(ids))
	for i, id := range ids {
		if !plain {
			if v, ok := versions[id]; ok && v != "" {
				out[i] = string(id) + "-" + string(v)

				continue
			}
		}

		out[i] = string(id)
	}

	return out
}

func toRefs(refs []catalog.Ref) []pointer {
	if len(refs) == 0 {
		return nil
	}

	out := make([]pointer, len(refs))
	for i, r := range refs {
		out[i] = pointer{ID: string(r.ID), Version: string(r.Version)}
	}

	return out
}

// toBadges converts badges, filling EventCatalog's REQUIRED backgroundColor
// and textColor when the caller left them empty (the schema rejects badges
// without them: "backgroundColor: Required").
const (
	defaultBadgeBackgroundColor = "blue"
	defaultBadgeTextColor       = "white"
)

func toBadges(badges []catalog.Badge) []badgeFM {
	if len(badges) == 0 {
		return nil
	}

	out := make([]badgeFM, len(badges))
	for i, b := range badges {
		bg, fg := string(b.BackgroundColor), string(b.TextColor)
		if bg == "" {
			bg = defaultBadgeBackgroundColor
		}
		if fg == "" {
			fg = defaultBadgeTextColor
		}
		out[i] = badgeFM{
			Content:         b.Content,
			BackgroundColor: bg,
			TextColor:       fg,
			Icon:            string(b.Icon),
			URL:             string(b.URL),
		}
	}

	return out
}

func toRepository(repo *catalog.Repository) *repositoryFM {
	if repo == nil {
		return nil
	}

	return &repositoryFM{
		Language: string(repo.Language),
		URL:      string(repo.URL),
	}
}

func toOperation(op *catalog.Operation) *operationFM {
	if op == nil {
		return nil
	}

	return &operationFM{
		Method:      string(op.Method),
		Path:        op.Path,
		StatusCodes: op.StatusCodes,
	}
}

func toResponses(specs []catalog.ResponseSpec) []responseFM {
	if len(specs) == 0 {
		return nil
	}

	out := make([]responseFM, len(specs))
	for i, s := range specs {
		out[i] = responseFM{
			StatusCode:  s.StatusCode,
			Description: s.Description,
		}
	}

	return out
}

func toSpecifications(specs []catalog.Specification) []specificationFM {
	if len(specs) == 0 {
		return nil
	}

	out := make([]specificationFM, len(specs))
	for i, s := range specs {
		out[i] = specificationFM{
			Type: s.Type,
			Path: s.Path,
			Name: string(s.Name),
		}
	}

	return out
}

func toAttachments(attachments []catalog.Attachment) []attachmentFM {
	if len(attachments) == 0 {
		return nil
	}

	out := make([]attachmentFM, len(attachments))
	for i, a := range attachments {
		out[i] = attachmentFM{
			URL:         string(a.URL),
			Title:       string(a.Title),
			Description: string(a.Description),
			Type:        a.Type,
			Icon:        string(a.Icon),
		}
	}

	return out
}


