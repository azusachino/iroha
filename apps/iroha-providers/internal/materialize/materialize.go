package materialize

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

// Source copies provider evidence to a temporary file so file-oriented
// parsers can read it more than once without reopening the original source.
func Source(ctx context.Context, source provider.Source, providerID, pattern string) (string, func(), error) {
	if source.Open == nil {
		return "", func() {}, sourceError(providerID, source, "open_source", errors.New("source opener is required"))
	}
	reader, err := source.Open(ctx)
	if err != nil {
		return "", func() {}, sourceError(providerID, source, "open_source", err)
	}
	temp, err := os.CreateTemp("", pattern)
	if err != nil {
		_ = reader.Close()
		return "", func() {}, sourceError(providerID, source, "create_temp_source", err)
	}
	path := temp.Name()
	cleanup := func() { _ = os.Remove(path) }
	_, copyErr := io.Copy(temp, ContextReader{ctx: ctx, Reader: reader})
	readerErr := reader.Close()
	tempErr := temp.Close()
	if copyErr != nil {
		cleanup()
		return "", func() {}, sourceError(providerID, source, "copy_source", copyErr)
	}
	if readerErr != nil {
		cleanup()
		return "", func() {}, sourceError(providerID, source, "close_source", readerErr)
	}
	if tempErr != nil {
		cleanup()
		return "", func() {}, sourceError(providerID, source, "close_temp_source", tempErr)
	}
	return path, cleanup, nil
}

type ContextReader struct {
	ctx    context.Context
	Reader io.Reader
}

func NewContextReader(ctx context.Context, reader io.Reader) io.Reader {
	return ContextReader{ctx: ctx, Reader: reader}
}

func (r ContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

func sourceError(providerID string, source provider.Source, operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	kind := provider.ErrorInvalidSource
	if operation == "create_temp_source" || operation == "close_temp_source" {
		kind = provider.ErrorInternal
	}
	return &provider.Error{
		Kind:       kind,
		Provider:   providerID,
		SourceKind: source.Kind,
		Op:         operation,
		Err:        fmt.Errorf("%w", err),
	}
}
