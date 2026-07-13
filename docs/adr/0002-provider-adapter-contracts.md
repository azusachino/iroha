# ADR 0002: Provider adapter contracts

- Status: Proposed
- Date: 2026-07-13
- Depends on: [ADR 0001](0001-provider-observations-and-canonical-records.md)

## Context

Iroha needs to support several data providers without making the server or
worker know provider-specific formats. Apple Health is a ZIP/XML export;
Garmin may be a file or connector; AniList and Bangumi are APIs. They have
different transport and authentication models, but they must produce the same
canonical observation contracts.

The provider boundary should follow normal Go practice:

- interfaces are small and consumed by the import pipeline;
- optional capabilities are expressed by separate interfaces;
- provider implementations own provider-specific types and dependencies;
- core contracts do not import GORM, HTTP clients, CLI code, or provider code;
- adapters return explicit typed observations instead of `map[string]any`.

This is similar to the useful part of Kubernetes' extension model: a stable
core API describes capabilities and objects, while implementations register
themselves and are selected by identity. It is not a requirement to reproduce
Kubernetes' machinery or generic runtime object system.

## Decision

`iroha-core` defines the provider-facing API. `iroha-providers` implements it.
The import pipeline owns registry selection, source reconciliation, and
persistence. The HTTP server and worker do not depend on provider-specific
packages directly.

### Provider descriptor

Every provider exposes a stable descriptor:

```go
type Descriptor struct {
	ID             string
	DisplayName    string
	AdapterVersion string
	SourceKinds    []string
	Domains        []Domain
	Capabilities   []Capability
}

type Capability string

const (
	CapabilityHealthActivities   Capability = "health.activities"
	CapabilityHealthSleep        Capability = "health.sleep"
	CapabilityHealthDailySummary Capability = "health.daily_summary"
	CapabilityHealthDailyMetrics Capability = "health.daily_metrics"
	CapabilityMediaLibrary       Capability = "media.library"
	CapabilityMediaProgress      Capability = "media.progress"
	CapabilityMediaRating        Capability = "media.rating"
)

type Domain string

const (
	DomainHealth Domain = "health"
	DomainMedia  Domain = "media"
)
```

`ID` is a durable storage identity such as `apple_health`, `garmin`, or
`anilist`. It is not a display name and must not change during localization.
`AdapterVersion` changes when the adapter's interpretation of source data
changes. It participates in import reprocessing decisions.

Capabilities are declarative metadata. The domain prefix groups related
capabilities, while `Domains` makes the provider's supported domains explicit.
They are also checked at runtime by the import pipeline before invoking an
optional capability interface.

### Source input

The core contract abstracts source access without assuming a file path, HTTP
connection, or database row:

```go
type Source struct {
	Kind             string
	OriginalFilename string
	SHA256           string
	Open             func(context.Context) (io.ReadCloser, error)
}

type ImportOptions struct {
	Requested []Capability
}
```

`Open` is called by an adapter when it needs the evidence and can be called
more than once. The caller owns the underlying raw-file lifecycle; the adapter
must close every reader it opens. A provider must never mutate the source or
write to the raw-file store.

The import pipeline supplies the source digest and provider identity. The
adapter derives provider source keys and content hashes from the evidence.

### Base adapter interface

The base interface is intentionally small:

```go
type Adapter interface {
	Descriptor() Descriptor
}
```

An adapter is useful only when it also implements one or more capability
interfaces. This keeps optional support explicit and prevents a large
interface whose every method returns `ErrUnsupported`.

```go
type ActivityImporter interface {
	ImportActivities(context.Context, Source, ImportOptions) ([]observations.Activity, error)
}

type SleepImporter interface {
	ImportSleep(context.Context, Source, ImportOptions) ([]observations.Sleep, error)
}

type DailyImporter interface {
	ImportDaily(context.Context, Source, ImportOptions) (DailyObservations, error)
}

type DailyObservations struct {
	Summaries []observations.DailySummary
	Metrics   []observations.DailyMetric
}

type MediaImporter interface {
	ImportMedia(context.Context, Source, ImportOptions) ([]observations.Media, error)
}
```

Media capabilities use typed media observation contracts. They are not forced
into health-specific observations. `MediaImporter` is shared by the library,
progress, and rating capabilities; the descriptor declares which of those
outputs a provider actually supports.

The import pipeline uses type assertions after checking the descriptor:

```go
adapter, ok := registry.Get(providerID)
activityImporter, ok := adapter.(core.ActivityImporter)
```

The registry is an application concern, not a global mutable singleton:

```go
type Registry interface {
	Get(providerID string) (Adapter, bool)
	List() []Descriptor
}
```

The worker constructs one registry at startup with the compiled provider
implementations. Tests can construct a registry with fake adapters.

### Observation entities

Capability interfaces return observation DTOs, not canonical database rows.
They contain provider facts and stable source identity inputs:

```go
type Identity struct {
	Provider   string
	SourceKind string
	SourceKey  string
}

type Envelope struct {
	Identity    Identity
	ContentHash string
}
```

Each domain observation embeds or contains an `Envelope`. The adapter owns the
meaning of `SourceKey`; the import pipeline owns the uniqueness scope and
stores it as `(provider, source_kind, source_key)`. Database UUIDs, selected
canonical values, match status, and conflict confidence are assigned outside
the adapter.

An observation must be deterministic for the same source evidence and adapter
version. It must not contain job IDs, request IDs, timestamps representing
import time, or database-generated identifiers.

### Error contract

Adapters return ordinary wrapped Go errors with a small classification at the
provider boundary:

```go
type ErrorKind string

const (
	ErrorInvalidSource ErrorKind = "invalid_source"
	ErrorUnsupported   ErrorKind = "unsupported"
	ErrorUnavailable   ErrorKind = "unavailable"
	ErrorRateLimited   ErrorKind = "rate_limited"
	ErrorInternal      ErrorKind = "internal"
)

type Error struct {
	Kind       ErrorKind
	Provider   string
	SourceKind string
	Op         string
	Err        error
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
```

Errors should provide source/provider context with `%w`. The pipeline maps
error kinds to retry policy:

- `invalid_source` and `unsupported` are permanent failures;
- `unavailable` and `rate_limited` are retryable, subject to job limits;
- `internal` is retryable only when the operation is known to be idempotent;
- `context.Canceled` and `context.DeadlineExceeded` preserve cancellation
  semantics and are not wrapped into a permanent provider error.

Adapters do not update job state, retry, log secrets, or emit HTTP errors.
The worker owns retry count, backoff, dead-letter behavior, and final job
status.

### Logging

Logging is structured with `log/slog`. The import pipeline emits lifecycle
events; providers only emit diagnostic logs when a provider-specific operation
cannot be represented in the returned result.

Every import log entry should include stable, non-secret fields when available:

```text
job_id, raw_file_id, provider, source_kind, adapter_version,
parser_version, operation, duration_ms, observation_counts, error_kind
```

The raw ZIP contents, access tokens, connector headers, personal titles, and
full source payloads must never be logged. Hashes and IDs are preferred for
correlation. Record-level logs are disabled by default; debugging a provider
must use bounded counters or explicitly redacted diagnostics.

The adapter contract does not accept a logger. This keeps logging policy at the
application boundary and prevents every provider from choosing incompatible
logging fields or levels.

### Diagnostics and metrics

Non-fatal provider findings are returned as bounded diagnostics rather than
being silently logged or treated as errors:

```go
type Diagnostic struct {
	Code     string
	Severity Severity
	Message  string
	Count    int
}

type ImportResult[T any] struct {
	Items       []T
	Diagnostics []Diagnostic
}
```

Messages must be safe for logs and UI. A provider must cap diagnostic cardinality
and aggregate repeated findings.

The pipeline owns metrics, using provider/capability labels only. At minimum it
records import duration, success/failure count, retry count, bytes read,
observation counts, and diagnostic counts. It must not use source keys, user
titles, raw filenames, or external IDs as metric labels.

### Context, cancellation, and timeouts

Every adapter call receives a context. Providers must check cancellation during
large-file scans and between connector requests. They must close every resource
opened through `Source.Open` and stop producing results after cancellation.

The caller sets the overall import deadline. Connector adapters may apply
shorter per-request timeouts, but must derive them from the caller's context.
No adapter may create an uncancellable background context.

### Resource and streaming rules

Adapters must not assume that a source fits in memory. File providers should
stream and use bounded indexes where possible. Connector providers should bound
page size and response bodies.

The initial typed slice API is acceptable for small domain summaries, but large
activity routes and sensor streams must use a future iterator or sink-backed
batch API before Garmin/FIT support lands. The contract must not force a full
900 MB Apple export into one in-memory object.

### Atomicity and partial results

An adapter may return diagnostics with successful observations, but it must not
return observations that are known to be structurally invalid. The import
pipeline validates the complete batch before persistence.

Persistence is atomic per import snapshot: a failed import must not expose a
partial canonical projection. For bounded streaming work, the pipeline writes
to a transaction or staging scope and commits only after source reconciliation
and validation complete.

If a provider can recover individual malformed records, it skips those records,
increments a bounded diagnostic, and continues. If source structure or identity
cannot be trusted, it returns a permanent `invalid_source` error instead.

### Security and data handling

Provider adapters receive the minimum source access needed for their capability.
Credentials belong to connector configuration, not observation DTOs or logs.
Raw evidence remains owned by the raw-file/intake subsystem. Adapters may read
it but do not copy it into logs, errors, metrics, or arbitrary JSON fields.

Provider-specific payloads that are required for later reprocessing must be
stored as raw evidence or an explicitly designed provider table; they must not
be hidden in a generic observation metadata map.

### Capability validation

The pipeline validates these invariants before import:

1. descriptor ID is non-empty and matches the registry key;
2. every declared capability has a matching optional interface;
3. every requested capability is declared and implemented;
4. returned observations have the expected provider identity;
5. source keys and content hashes are non-empty;
6. duplicate identities within one batch are rejected before persistence.

This makes provider registration failures startup/test failures instead of
partial data writes.

## Ownership and dependencies

```text
iroha-core
  observations, capabilities, adapter contracts

iroha-providers
  applehealth, garmin, anilist, bangumi, goodreads, weread

iroha-imports
  registry, source lifecycle, reconciliation, persistence boundary

iroha-runtime
  shared IDs, persistence models, cache client, and Postgres-backed job service

iroha-server
  HTTP/read services and enqueue commands

iroha-job
  worker loop and import-pipeline execution
```

Provider packages may depend on `iroha-core` and narrowly scoped upstream
client libraries. They must not depend on `iroha-server`, `iroha-job`, or
database models. Runtime infrastructure is kept in `iroha-runtime` so the
import pipeline and both executable services share it without a module cycle.

## Alternatives considered

### One large `Provider` interface

Rejected. It forces every provider to implement unrelated methods and creates
`ErrUnsupported` boilerplate. Separate capability interfaces are idiomatic Go
and make support discoverable.

### Generic `[]Observation` or `map[string]any`

Rejected. It moves type errors to runtime and recreates the ambiguous canonical
layer this design is intended to remove.

### One Go module per provider

Rejected for now. A single `iroha-providers` module keeps local development and
versioning simple while allowing package-level dependency isolation. A provider
can become a separate module later if its dependency or release lifecycle
requires it.

## Consequences

Positive:

- server/job depend on stable contracts rather than Apple/Garmin details;
- capability support is explicit and testable;
- provider-specific types remain close to their implementations;
- import reprocessing can use adapter versions consistently;
- fake adapters can test reconciliation without real ZIPs or APIs.

Costs:

- the import pipeline needs a registry and capability validation;
- each new capability adds a typed contract and observation entity;
- adapters must maintain deterministic source identity and hashing.

## Next implementation sequence

1. Add versioned contract packages to `iroha-core`.
2. Add the registry and validation tests with fake adapters.
3. Convert Apple Health to the first `ActivityImporter`, `SleepImporter`, and
   `DailyImporter` implementation.
4. Move import orchestration/persistence into `iroha-imports`.
5. Make the worker construct and execute the registry; keep the server as the
   enqueue/read boundary.
