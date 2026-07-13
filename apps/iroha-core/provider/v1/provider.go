// Package v1 contains the versioned provider adapter contract.
package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

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

type ImportOptions struct {
	Requested []Capability
}

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

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Code     string
	Severity Severity
	Message  string
	Count    int
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
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s provider=%s source=%s op=%s: %v", e.Kind, e.Provider, e.SourceKind, e.Op, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

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

func ValidateRequested(adapter Adapter, options ImportOptions) error {
	if err := ValidateAdapter(adapter); err != nil {
		return err
	}
	declared := make(map[Capability]struct{}, len(adapter.Descriptor().Capabilities))
	for _, capability := range adapter.Descriptor().Capabilities {
		declared[capability] = struct{}{}
	}
	for _, capability := range options.Requested {
		if _, ok := declared[capability]; !ok {
			return fmt.Errorf("provider %q does not support requested capability %q", adapter.Descriptor().ID, capability)
		}
	}
	return nil
}

func implementsCapability(adapter Adapter, capability Capability) bool {
	switch capability {
	case CapabilityHealthActivities:
		_, ok := adapter.(ActivityImporter)
		return ok
	case CapabilityHealthSleep:
		_, ok := adapter.(SleepImporter)
		return ok
	case CapabilityHealthDailySummary, CapabilityHealthDailyMetrics:
		_, ok := adapter.(DailyImporter)
		return ok
	case CapabilityMediaLibrary, CapabilityMediaProgress, CapabilityMediaRating:
		return false
	default:
		return false
	}
}
