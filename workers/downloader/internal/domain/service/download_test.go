package service

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCode4rena_DownloadHTML(t *testing.T) {
	MB := int64(1024 * 1024)
	url := "https://code4rena.com/reports/2025-06-panoptic-hypovault"
	svc := NewDownloadService(&http.Client{}, 10*MB)

	result, _ := svc.Download(t.Context(), url)
	content := string(result.Content())
	ext := result.Extension()

	assert.Equal(t, ext, ".html")
	assert.Equal(t, result.Hash(), "f2852b49daf89966747f802fecde25a79930d8dcaa4bd3ad7c1bce07ee349aad")
	assert.True(t, strings.Contains(content, "Panoptic Hypovault"))
}
func TestCantina_DownloadPDF(t *testing.T) {
	MB := int64(1024 * 1024)
	url := "https://cdn.cantina.xyz/reports/cantina_opentrade_aug2025.pdf"
	svc := NewDownloadService(&http.Client{}, 10*MB)

	result, _ := svc.Download(t.Context(), url)
	ext := result.Extension()

	assert.Equal(t, ext, ".pdf")
	assert.Equal(t, result.Size(), int64(469704))
	assert.Equal(t, result.Hash(), "ec32db41dd837bc5a4a7b231aebf38354940af179060608854fb1e7a32205641")
}
