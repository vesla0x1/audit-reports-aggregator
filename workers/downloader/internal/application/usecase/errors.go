package usecase

import (
	"fmt"
)

func ErrDownloadFileAuditReportNotFound(err error) error {
	return fmt.Errorf("Failed to get audit report: %w", err)
}

func ErrAuditProviderNotFound(err error) error {
	return fmt.Errorf("Failed to get audit provider: %w", err)
}

func ErrDownloadFileUpdateFailed(err error) error {
	return fmt.Errorf("failed to update download status: %w", err)
}

func ErrDownloadFileDownloadFailed(err error) error {
	return fmt.Errorf("failed to download file: %w", err)
}

func ErrDownloadFileUploadFailed(err error) error {
	return fmt.Errorf("failed to upload download status: %w", err)
}

func ErrProcessCreationFailed(err error) error {
	return fmt.Errorf("failed to create process: %w", err)
}

func ErrPublishProcessEvent(err error) error {
	return fmt.Errorf("failed to publish process event: %w", err)
}
