package mocks

import (
	"context"
	"io"

	"downloader/internal/domain"

	"github.com/stretchr/testify/mock"
)

// MockDownloadService is a mock implementation of the download service
type MockDownloadService struct {
	mock.Mock
}

func (m *MockDownloadService) Execute(ctx context.Context, req domain.DownloadRequest) (*domain.DownloadResult, io.ReadCloser, error) {
	args := m.Called(ctx, req)

	var result *domain.DownloadResult
	if args.Get(0) != nil {
		result = args.Get(0).(*domain.DownloadResult)
	}

	var reader io.ReadCloser
	if args.Get(1) != nil {
		reader = args.Get(1).(io.ReadCloser)
	}

	return result, reader, args.Error(2)
}
