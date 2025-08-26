package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWorker is a simple test implementation of Worker
type TestWorker struct {
	name string
}

func (w *TestWorker) Name() string {
	return w.name
}

func (w *TestWorker) Process(ctx context.Context, req Request) (Response, error) {
	return NewSuccessResponse(req.ID, map[string]string{
		"processed_by": w.name,
	})
}

func (w *TestWorker) Health(ctx context.Context) error {
	return nil
}

func TestWorkerInterface(t *testing.T) {
	// Ensure TestWorker implements Worker interface
	var _ Worker = (*TestWorker)(nil)

	worker := &TestWorker{name: "test-worker"}

	assert.Equal(t, "test-worker", worker.Name())

	ctx := context.Background()
	req := Request{ID: "test-123", Type: "test"}

	resp, err := worker.Process(ctx, req)

	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "test-123", resp.ID)

	err = worker.Health(ctx)
	assert.NoError(t, err)
}
