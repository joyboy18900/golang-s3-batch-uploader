package service

import "context"

type FileResult struct {
	File  string `json:"file"`
	Error string `json:"error,omitempty"`
}

type BatchResult struct {
	Succeeded []FileResult `json:"succeeded"`
	Failed    []FileResult `json:"failed"`
}

type BatchRequest struct {
	SourceDir string `json:"source_dir"`
}

type BatchService interface {
	Run(ctx context.Context, req BatchRequest) (*BatchResult, error)
}
