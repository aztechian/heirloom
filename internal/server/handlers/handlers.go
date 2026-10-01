// Package handlers implements the HTTP handlers for the Heirloom API,
// organized into one file per API resource (collections, assets, instance).
// API implements types.StrictServerInterface.
package handlers

import (
	"strings"

	"github.com/aztechian/heirloom/internal/storage"
)

const MaxSlugLength = 64

// API implements types.StrictServerInterface. Each resource's storage
// dependency is held as a narrow, resource-specific interface rather than
// the full storage.Storage aggregate, so a test for one resource never
// needs to fake methods belonging to another.
type API struct {
	collections storage.CollectionStorage
}

// New constructs an API backed by store. Each field is assigned
// independently so narrowing a resource to a smaller interface here never
// requires changes anywhere upstream of this constructor.
func New(store storage.Storage) API {
	return API{collections: store}
}

// slugify derives a URL-safe slug from a collection name: lowercased, with
// runs of anything other than a-z0-9 collapsed to a single hyphen, and
// leading/trailing hyphens trimmed. The result always satisfies the spec's
// slug pattern; only its length still needs checking by the caller.
func slugify(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case !prevHyphen && b.Len() > 0:
			b.WriteByte('-')
			prevHyphen = true
		}
	}

	return strings.TrimSuffix(b.String(), "-")
}
