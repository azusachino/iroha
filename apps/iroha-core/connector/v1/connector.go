// Package v1 contains the versioned connector contract for remote media
// sources. Connectors fetch evidence; parsers remain responsible for turning
// that evidence into provider-neutral observations.
package v1

import (
	"context"
	"time"
)

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
	// ObservedAt is when the connector received this source snapshot. It is
	// distinct from the time Iroha stores the raw file or processes its job.
	ObservedAt time.Time
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
