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
	CreatedAfter int64  `json:"created_after,omitempty"`
	UserID       int    `json:"user_id,omitempty"`
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

// ResumeCursorProvider lets a bounded connector retain a small watermark after
// a successful run. The sync runner stores this cursor without fetching it in
// the same run; the next run starts from the connector's overlap window.
type ResumeCursorProvider interface {
	ResumeCursor() *Cursor
}
