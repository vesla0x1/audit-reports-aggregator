package handler

import (
	"downloader/internal/domain/entity/download"
	"errors"
	"shared/application/ports"
)

func successResponse() ports.RuntimeResponse {
	return ports.RuntimeResponse{
		Success: true,
	}
}

func errorResponse(message string) ports.RuntimeResponse {
	return ports.RuntimeResponse{
		Success: false,
		Error:   message,
	}
}

func (h *DownloadHandler) handleDownloadSuccess() (ports.RuntimeResponse, error) {
	h.logger.Info("Download successfully completed!")
	return successResponse(), nil
}

func (h *DownloadHandler) handleDownloadError(downloadID int64, err error) (ports.RuntimeResponse, error) {
	switch {
	case errors.Is(err, download.ErrAlreadyCompleted):
		h.logger.Info("Download already completed")
		return successResponse(), nil

	default:
		h.logger.Error("Download failed",
			"download_id", downloadID,
			"error", err.Error())
		return errorResponse(err.Error()), nil
	}
}

func (h *DownloadHandler) handleError(err error) (ports.RuntimeResponse, error) {
	h.logger.Error("Download failed", "error", err.Error())
	return errorResponse(err.Error()), nil
}
