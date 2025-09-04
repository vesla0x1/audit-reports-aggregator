package service

import (
	"context"
	"errors"
	"net/http"

	"downloader/internal/application/ports"
	"downloader/internal/domain/entity/downloadresult"
)

type DownloadService struct {
	httpClient  ports.HTTPClient
	maxFileSize int64
	userAgent   string
}

func NewDownloadService(httpClient ports.HTTPClient, maxFileSize int64) *DownloadService {
	return &DownloadService{
		httpClient:  httpClient,
		maxFileSize: maxFileSize,
		userAgent:   "AuditReportDownloader/1.0",
	}
}

func (s *DownloadService) Download(ctx context.Context, url string) (*downloadresult.DownloadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrRequestCreation(err)
	}

	req.Header.Set("User-Agent", s.userAgent)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ErrHTTPRequest(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatus(resp.StatusCode)
	}

	result, err := downloadresult.NewDownloadResultFromReader(
		resp.Body,
		url,
		resp.Header.Get("Content-Type"),
		s.maxFileSize,
	)

	if err != nil {
		if errors.Is(err, downloadresult.ErrSizeExceeded(s.maxFileSize)) {
			return nil, ErrFileTooLarge
		}
		return nil, ErrReadResponse(err)
	}

	return result, nil
}
