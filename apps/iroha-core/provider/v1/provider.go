// Package v1 contains the versioned provider adapter contract.
package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

type Capability string

const (
	CapabilityActivities   Capability = "activities"
	CapabilitySleep        Capability = "sleep"
	CapabilityDailySummary Capability = "daily_summary"
	CapabilityDailyMetrics Capability = "daily_metrics"
	CapabilityMedia        Capability = "media"
)

type Descriptor struct {
	ID             string
	DisplayName    string
	AdapterVersion string
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

type DailyObservations struct {
	Summaries []observations.DailySummary
	Metrics   []observations.DailyMetric
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
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	registry := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if err := ValidateAdapter(adapter); err != nil {
			return nil, err
		}
		id := adapter.Descriptor().ID
		if _, exists := registry.adapters[id]; exists {
			return nil, fmt.Errorf("provider %q is registered more than once", id)
		}
		registry.adapters[id] = adapter
	}
	return registry, nil
}

func (r *Registry) Get(providerID string) (Adapter, bool) {
	adapter, ok := r.adapters[providerID]
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
	seen := make(map[Capability]struct{}, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		if capability == "" {
			return fmt.Errorf("provider %q declares an empty capability", descriptor.ID)
		}
		if _, exists := seen[capability]; exists {
			return fmt.Errorf("provider %q declares capability %q more than once", descriptor.ID, capability)
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
	case CapabilityActivities:
		_, ok := adapter.(ActivityImporter)
		return ok
	case CapabilitySleep:
		_, ok := adapter.(SleepImporter)
		return ok
	case CapabilityDailySummary, CapabilityDailyMetrics:
		_, ok := adapter.(DailyImporter)
		return ok
	case CapabilityMedia:
		return false
	default:
		return false
	}
}
