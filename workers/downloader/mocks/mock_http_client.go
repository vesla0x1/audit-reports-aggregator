package mocks

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of ports.HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Download(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, map[string]string, error) {
	args := m.Called(ctx, url, headers)

	var reader io.ReadCloser
	if args.Get(0) != nil {
		reader = args.Get(0).(io.ReadCloser)
	}

	var respHeaders map[string]string
	if args.Get(1) != nil {
		respHeaders = args.Get(1).(map[string]string)
	}

	return reader, respHeaders, args.Error(2)
}
