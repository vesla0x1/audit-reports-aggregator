package service

import (
	"fmt"
	"regexp"
	"strings"
)

type StoragePathService struct {
	sanitizer *regexp.Regexp
}

func NewStoragePathService() *StoragePathService {
	return &StoragePathService{
		sanitizer: regexp.MustCompile("[^a-z0-9_-]"),
	}
}

func (s *StoragePathService) GeneratePath(providerSlug string, reportID int64, title string, extension string) string {
	sanitizedTitle := s.sanitizeTitle(title)
	return fmt.Sprintf("%s/%d_%s.%s", providerSlug, reportID, sanitizedTitle, extension)
}

func (s *StoragePathService) sanitizeTitle(title string) string {
	title = strings.ToLower(title)
	title = strings.ReplaceAll(title, " ", "_")
	title = s.sanitizer.ReplaceAllString(title, "")
	if len(title) > 50 {
		title = title[:50]
	}
	return title
}
