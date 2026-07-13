package connectors

import (
	"fmt"
	"sort"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
)

type Registry struct {
	connectors map[string]connector.Connector
}

func New(items ...connector.Connector) (*Registry, error) {
	registry := &Registry{connectors: make(map[string]connector.Connector, len(items))}
	for _, item := range items {
		if item == nil {
			return nil, fmt.Errorf("connector is nil")
		}
		descriptor := item.Descriptor()
		if descriptor.ID == "" {
			return nil, fmt.Errorf("connector id is required")
		}
		if descriptor.SourceKind == "" {
			return nil, fmt.Errorf("connector %q source kind is required", descriptor.ID)
		}
		if _, exists := registry.connectors[descriptor.ID]; exists {
			return nil, fmt.Errorf("connector %q is registered more than once", descriptor.ID)
		}
		registry.connectors[descriptor.ID] = item
	}
	return registry, nil
}

func (r *Registry) Get(id string) (connector.Connector, bool) {
	item, ok := r.connectors[id]
	return item, ok
}

func (r *Registry) List() []connector.Descriptor {
	descriptors := make([]connector.Descriptor, 0, len(r.connectors))
	for _, item := range r.connectors {
		descriptors = append(descriptors, item.Descriptor())
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID < descriptors[j].ID })
	return descriptors
}
