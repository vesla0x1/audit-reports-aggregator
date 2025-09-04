package handler

import (
	"context"
	"downloader/internal/application/dto"
	"downloader/internal/application/ports"
	"downloader/internal/application/usecase"
)

type DownloadHandler struct {
	usecase *usecase.DownloadFile
	logger  ports.Logger
}

func NewDownloadHandler(downloadFile *usecase.DownloadFile, obs ports.Observability) *DownloadHandler {
	logger, _ := obs.LoggerScoped("handler.download")
	return &DownloadHandler{
		usecase: downloadFile,
		logger:  logger,
	}
}

func (h *DownloadHandler) Handle(ctx context.Context, request ports.RuntimeRequest) (ports.RuntimeResponse, error) {
	// Parse request
	downloadReq, err := h.parseRequest(request)
	if err != nil {
		return h.handleError(err)
	}

	// Validate
	if err := downloadReq.Validate(); err != nil {
		return h.handleError(ErrHandlerInvalidPayload(err))
	}

	// Process
	if err := h.usecase.Download(ctx, downloadReq); err != nil {
		return h.handleDownloadError(downloadReq.DownloadID, err)
	}

	return h.handleDownloadSuccess()
}

func (w *DownloadHandler) parseRequest(request ports.RuntimeRequest) (*dto.DownloadRequest, error) {
	var downloadReq dto.DownloadRequest
	if err := request.Unmarshal(&downloadReq); err != nil {
		return nil, ErrHandlerUnmarshal(err)
	}
	return &downloadReq, nil
}
