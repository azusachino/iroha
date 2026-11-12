// Package v1 contains the versioned connector contract for remote media
// sources. Connectors fetch evidence; parsers remain responsible for turning
// that evidence into provider-neutral observations.
package v1

import (
	"context"
	"errors"
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
	ContentType       string
	Body              []byte
	SourceKind        string
	Filename          string
	SourceInstanceKey string
	IngestionMode     string
	// OrderingBasis is the ordering fact this page proves, such as a feed
	// watermark. It is stored on the page's source receipt so the next run can
	// resume from it; a page that proves nothing about ordering leaves it empty.
	OrderingBasis string
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

// CoverageWindowProvider lets a bounded connector report where the window it
// fetches begins, given the cursor a run starts from. Connectors that fetch a
// whole current state do not implement it.
type CoverageWindowProvider interface {
	CoverageWindowStart(cursor *Cursor, now time.Time) time.Time
}

// DefaultMaxSnapshotAge is how old a connector's last completed full snapshot
// may grow before the follower reads another one. A changes feed can only add
// or update what it lists, so it cannot report an unlink or a removal; only a
// full snapshot shows the current truth. The follower therefore refreshes it
// on this schedule, whatever the changes feed's own horizon is.
const DefaultMaxSnapshotAge = 7 * 24 * time.Hour

// SnapshotAgeProvider lets a change-feed connector state its own maximum
// snapshot age instead of DefaultMaxSnapshotAge.
type SnapshotAgeProvider interface {
	MaxSnapshotAge() time.Duration
}

// ErrSnapshotRequired reports that an incremental feed can no longer serve the
// caller's watermark. The caller must read a full snapshot instead; the signal
// is a normal state transition, not a failure.
var ErrSnapshotRequired = errors.New("change horizon exceeded; full snapshot required")

// ChangePage is one page of an incremental feed: the raw evidence to store,
// the ordering fact the page proves, and the cursor for the next page. A nil
// cursor means the feed is exhausted for this run.
type ChangePage struct {
	Snapshot  Snapshot
	Watermark string
	Cursor    *Cursor
}

// ChangeFetcher lets a connector serve only what changed since a watermark
// instead of re-reading its whole current state. FetchChanges returns
// ErrSnapshotRequired when it cannot serve that watermark; any other error is a
// real failure. A connector that implements it must also keep Fetch working:
// Fetch is the bootstrap and the fallback.
type ChangeFetcher interface {
	FetchChanges(context.Context, Credentials, time.Time, *Cursor) (ChangePage, error)
}
