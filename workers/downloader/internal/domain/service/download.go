package service

import (
	"context"
	"fmt"
	"io"

	"downloader/internal/domain/model"
	"downloader/internal/domain/util"
	"shared/application/ports"
)

type DownloadService struct {
	httpClient ports.HTTPClient
}

func NewDownloadService(httpClient ports.HTTPClient) *DownloadService {
	return &DownloadService{
		httpClient: httpClient,
	}
}

func (s *DownloadService) Download(ctx context.Context, url string) (*model.DownloadResult, error) {
	headers := map[string]string{
		"User-Agent": "AuditReportDownloader/1.0",
	}

	reader, respHeaders, err := s.httpClient.Download(ctx, url, headers)
	if err != nil {
		return nil, model.NewDownloadError(model.NetworkError,
			fmt.Sprintf("failed to download file: %v", err), url)
	}
	defer reader.Close()

	content, err := s.readContent(reader, respHeaders)
	if err != nil {
		return nil, err
	}

	return &model.DownloadResult{
		Content:     content,
		Hash:        util.CalculateHash(content),
		Size:        int64(len(content)),
		ContentType: util.ExtractContentType(respHeaders),
		Extension:   util.DetermineExtension(url, util.ExtractContentType(respHeaders)),
		URL:         url,
	}, nil
}

func (s *DownloadService) readContent(reader io.ReadCloser, headers map[string]string) ([]byte, error) {
	const maxFileSize = 100 * 1024 * 1024 // 100MB

	// Check content length if provided
	if contentLength := headers["content-length"]; contentLength != "" {
		var size int64
		fmt.Sscanf(contentLength, "%d", &size)
		if size > maxFileSize {
			return nil, model.NewDownloadError(
				model.FileTooLargeError,
				fmt.Sprintf("file too large: %d bytes (max: %d)", size, maxFileSize),
				"",
			)
		}
	}

	// Use LimitReader to prevent excessive memory usage
	limitedReader := io.LimitReader(reader, maxFileSize+1)

	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, model.NewDownloadError(
			model.ReadError,
			fmt.Sprintf("failed to read response body: %v", err),
			"",
		)
	}

	// Check if we hit the limit
	if len(content) > maxFileSize {
		return nil, model.NewDownloadError(
			model.FileTooLargeError,
			fmt.Sprintf("file exceeds maximum size of %d bytes", maxFileSize),
			"",
		)
	}

	return content, nil
}
