package braidproto

// Patch represents a patch operation in the Braid protocol
type Patch struct {
	Unit    string `json:"unit"`    // Unit represents the operational unit of the patch, e.g. "replace"
	Range   string `json:"range"`   // Range represents the path of the patch, e.g. "/foo/bar/0/id"
	Content string `json:"content"` // Content is the actual content of the patch, can be a JSON object
}

// Update represents a Braid protocol update with version, parents, and either patches or a full body
type Update struct {
	Version []string `json:"version"`           // Version identifiers for this update
	Parents []string `json:"parents"`           // Parent versions this update is based on
	Patches []Patch  `json:"patches,omitempty"` // Optional list of patches
	Body    string   `json:"body,omitempty"`    // Optional full body content
}
