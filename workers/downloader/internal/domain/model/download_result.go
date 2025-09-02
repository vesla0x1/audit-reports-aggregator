package model

type DownloadResult struct {
	Content     []byte
	Hash        string
	Size        int64
	ContentType string
	Extension   string
	URL         string
}
