// Package v1 contains the versioned provider adapter contract.
package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

type Capability string

const (
	CapabilityHealthActivities   Capability = "health.activities"
	CapabilityHealthSleep        Capability = "health.sleep"
	CapabilityHealthDailySummary Capability = "health.daily_summary"
	CapabilityHealthDailyMetrics Capability = "health.daily_metrics"
	CapabilityMediaLibrary       Capability = "media.library"
	CapabilityMediaProgress      Capability = "media.progress"
	CapabilityMediaRating        Capability = "media.rating"
	CapabilityMediaActivity      Capability = "media.provider_activity"
)

type Domain string

const (
	DomainHealth Domain = "health"
	DomainMedia  Domain = "media"
)

type Descriptor struct {
	ID             string
	DisplayName    string
	AdapterVersion string
	SourceKinds    []string
	Domains        []Domain
	Capabilities   []Capability
}

type Source struct {
	Kind             string
	OriginalFilename string
	SHA256           string
	Open             func(context.Context) (io.ReadCloser, error)
}

type ImportOptions struct{}

type Adapter interface {
	Descriptor() Descriptor
}

type ActivityImporter interface {
	ImportActivities(context.Context, Source, ImportOptions) ([]observations.Activity, error)
}

type SleepImporter interface {
	ImportSleep(context.Context, Source, ImportOptions) ([]observations.Sleep, error)
}

type DailyImporter interface {
	ImportDaily(context.Context, Source, ImportOptions) (DailyObservations, error)
}

type MediaImporter interface {
	ImportMedia(context.Context, Source, ImportOptions) ([]observations.Media, error)
}

// MediaHistoryImporter imports dated provider updates without treating them
// as exact consumption sessions. The importer is separate from MediaImporter
// because an activity feed may mention an item that is no longer in the
// provider's current library projection.
type MediaHistoryImporter interface {
	ImportMediaHistory(context.Context, Source, ImportOptions) ([]observations.MediaHistory, error)
}

// BatchImporter is an optional optimization for file-backed providers that
// can materialize one source and derive several capabilities from it. The
// import pipeline falls back to the individual capability interfaces when it
// is not implemented.
type BatchImporter interface {
	ImportAll(context.Context, Source, ImportOptions) (ImportBatch, error)
}

type DailyObservations struct {
	Summaries []observations.DailySummary
	Metrics   []observations.DailyMetric
}

type ImportBatch struct {
	Activities []observations.Activity
	Sleep      []observations.Sleep
	Daily      DailyObservations
}

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
	RetryAfter *time.Duration
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s provider=%s source=%s op=%s: %v", e.Kind, e.Provider, e.SourceKind, e.Op, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// RetryAfterDuration exposes a provider-supplied retry delay to the durable
// job queue without making the runtime depend on a concrete provider package.
func (e *Error) RetryAfterDuration() (time.Duration, bool) {
	if e == nil || e.RetryAfter == nil || *e.RetryAfter < 0 {
		return 0, false
	}
	return *e.RetryAfter, true
}

// ParseRetryAfter accepts both HTTP delta-seconds and HTTP-date values.
func ParseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return seconds, true
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	if when.Before(now) {
		return 0, true
	}
	return when.Sub(now), true
}

type Registry struct {
	adapters     map[string]Adapter
	bySourceKind map[string]Adapter
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	registry := &Registry{
		adapters:     make(map[string]Adapter, len(adapters)),
		bySourceKind: make(map[string]Adapter),
	}
	for _, adapter := range adapters {
		if err := ValidateAdapter(adapter); err != nil {
			return nil, err
		}
		id := adapter.Descriptor().ID
		if _, exists := registry.adapters[id]; exists {
			return nil, fmt.Errorf("provider %q is registered more than once", id)
		}
		for _, sourceKind := range adapter.Descriptor().SourceKinds {
			if existing, exists := registry.bySourceKind[sourceKind]; exists {
				return nil, fmt.Errorf("source kind %q is registered by both %q and %q", sourceKind, existing.Descriptor().ID, id)
			}
			registry.bySourceKind[sourceKind] = adapter
		}
		registry.adapters[id] = adapter
	}
	return registry, nil
}

func (r *Registry) Get(providerID string) (Adapter, bool) {
	adapter, ok := r.adapters[providerID]
	return adapter, ok
}

func (r *Registry) GetBySourceKind(sourceKind string) (Adapter, bool) {
	adapter, ok := r.bySourceKind[sourceKind]
	return adapter, ok
}

func (r *Registry) List() []Descriptor {
	descriptors := make([]Descriptor, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		descriptors = append(descriptors, adapter.Descriptor())
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID < descriptors[j].ID })
	return descriptors
}

func ValidateAdapter(adapter Adapter) error {
	if adapter == nil {
		return errors.New("provider adapter is nil")
	}
	descriptor := adapter.Descriptor()
	if descriptor.ID == "" {
		return errors.New("provider adapter ID is required")
	}
	if descriptor.AdapterVersion == "" {
		return fmt.Errorf("provider %q adapter version is required", descriptor.ID)
	}
	if len(descriptor.SourceKinds) == 0 {
		return fmt.Errorf("provider %q must declare at least one source kind", descriptor.ID)
	}
	if len(descriptor.Domains) == 0 {
		return fmt.Errorf("provider %q must declare at least one domain", descriptor.ID)
	}
	seenDomains := make(map[Domain]struct{}, len(descriptor.Domains))
	for _, domain := range descriptor.Domains {
		if domain == "" {
			return fmt.Errorf("provider %q declares an empty domain", descriptor.ID)
		}
		if _, exists := seenDomains[domain]; exists {
			return fmt.Errorf("provider %q declares domain %q more than once", descriptor.ID, domain)
		}
		seenDomains[domain] = struct{}{}
	}
	seenSourceKinds := make(map[string]struct{}, len(descriptor.SourceKinds))
	for _, sourceKind := range descriptor.SourceKinds {
		if sourceKind == "" {
			return fmt.Errorf("provider %q declares an empty source kind", descriptor.ID)
		}
		if _, exists := seenSourceKinds[sourceKind]; exists {
			return fmt.Errorf("provider %q declares source kind %q more than once", descriptor.ID, sourceKind)
		}
		seenSourceKinds[sourceKind] = struct{}{}
	}
	seen := make(map[Capability]struct{}, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		if capability == "" {
			return fmt.Errorf("provider %q declares an empty capability", descriptor.ID)
		}
		if _, exists := seen[capability]; exists {
			return fmt.Errorf("provider %q declares capability %q more than once", descriptor.ID, capability)
		}
		capabilityDomainName, _, _ := strings.Cut(string(capability), ".")
		capabilityDomain := Domain(capabilityDomainName)
		if _, exists := seenDomains[capabilityDomain]; !exists {
			return fmt.Errorf("provider %q capability %q belongs to undeclared domain %q", descriptor.ID, capability, capabilityDomain)
		}
		seen[capability] = struct{}{}
		if !implementsCapability(adapter, capability) {
			return fmt.Errorf("provider %q declares capability %q without implementing it", descriptor.ID, capability)
		}
	}
	return nil
}

func implementsCapability(adapter Adapter, capability Capability) bool {
	// A BatchImporter derives every health observation from one materialized
	// source, so it satisfies the health capabilities without also needing the
	// per-capability importer interfaces. Media is not carried by ImportBatch,
	// so it always requires a MediaImporter.
	_, batch := adapter.(BatchImporter)
	switch capability {
	case CapabilityHealthActivities:
		_, ok := adapter.(ActivityImporter)
		return ok || batch
	case CapabilityHealthSleep:
		_, ok := adapter.(SleepImporter)
		return ok || batch
	case CapabilityHealthDailySummary, CapabilityHealthDailyMetrics:
		_, ok := adapter.(DailyImporter)
		return ok || batch
	case CapabilityMediaLibrary, CapabilityMediaProgress, CapabilityMediaRating:
		_, ok := adapter.(MediaImporter)
		return ok
	case CapabilityMediaActivity:
		_, ok := adapter.(MediaHistoryImporter)
		return ok
	default:
		return false
	}
}
