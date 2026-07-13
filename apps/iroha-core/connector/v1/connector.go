// Package v1 contains the versioned connector contract for remote media
// sources. Connectors fetch evidence; parsers remain responsible for turning
// that evidence into provider-neutral observations.
package v1

import "context"

type Credentials struct {
	Values map[string]string `json:"values,omitempty"`
}

type Cursor struct {
	Token        string `json:"token,omitempty"`
	Page         int    `json:"page,omitempty"`
	UpdatedAfter string `json:"updated_after,omitempty"`
}

type Snapshot struct {
	ContentType string
	Body        []byte
	SourceKind  string
	Filename    string
}

type Descriptor struct {
	ID           string
	DisplayName  string
	SourceKind   string
	RequiresAuth bool
}

type Connector interface {
	Descriptor() Descriptor
	Fetch(context.Context, Credentials, *Cursor) (Snapshot, *Cursor, error)
}
