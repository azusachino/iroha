package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
)

// Registry maps job kinds to handlers. Build it with Register (typed payloads)
// or Handle (raw job), then pass Handlers() to NewService. This mirrors River's
// typed JobArgs/Worker[T] and asynq's ServeMux registration, but stays a thin
// layer over the existing kind->Handler map so nothing else has to change.
type Registry struct {
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// Handle registers a raw handler that receives the full job (payload undecoded).
func (r *Registry) Handle(kind string, handler Handler) {
	r.handlers[kind] = handler
}

// Handlers returns the kind->Handler map to pass to NewService.
func (r *Registry) Handlers() map[string]Handler {
	return r.handlers
}

// Register binds a job kind to a handler with a typed payload T, decoding
// job.PayloadJSON into T once here so handlers no longer hand-unmarshal. A
// generic free function (Go has no generic methods) in the style of
// river.AddWorker.
func Register[T any](r *Registry, kind string, handler func(context.Context, T) error) {
	r.handlers[kind] = func(ctx context.Context, job models.Job) error {
		var payload T
		if len(job.PayloadJSON) > 0 && string(job.PayloadJSON) != "null" {
			if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
				return fmt.Errorf("decode %s payload: %w", kind, err)
			}
		}
		return handler(ctx, payload)
	}
}
