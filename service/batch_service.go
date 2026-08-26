package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang-s3-batch-uploader/errs"
	"golang-s3-batch-uploader/logs"
	"golang-s3-batch-uploader/repository"
)

type batchService struct {
	uploader    repository.Uploader
	workerCount int
}

func NewBatchService(uploader repository.Uploader, workerCount int) BatchService {
	return batchService{uploader: uploader, workerCount: workerCount}
}

func (s batchService) Run(ctx context.Context, req BatchRequest) (*BatchResult, error) {
	entries, err := os.ReadDir(req.SourceDir)
	if err != nil {
		return nil, errs.NewValidationError("source_dir is not readable")
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".csv" {
			continue
		}
		files = append(files, filepath.Join(req.SourceDir, e.Name()))
	}

	results := s.processAll(ctx, files)

	result := &BatchResult{}
	for _, r := range results {
		if r.Error == "" {
			result.Succeeded = append(result.Succeeded, r)
		} else {
			result.Failed = append(result.Failed, r)
		}
	}

	return result, nil
}

func (s batchService) processAll(ctx context.Context, files []string) []FileResult {
	workers := s.workerCount
	if workers <= 0 {
		workers = 1
	}
	if len(files) > 0 && workers > len(files) {
		workers = len(files)
	}

	jobs := make(chan string, len(files))
	results := make(chan FileResult, len(files))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- s.processFile(ctx, path)
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	wg.Wait()
	close(results)

	all := make([]FileResult, 0, len(files))
	for r := range results {
		all = append(all, r)
	}
	return all
}

func (s batchService) processFile(ctx context.Context, path string) FileResult {
	name := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		logs.Error("open file: ", err)
		return FileResult{File: name, Error: err.Error()}
	}
	defer f.Close()

	if err := validateCSV(f); err != nil {
		return FileResult{File: name, Error: err.Error()}
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		logs.Error("seek file: ", err)
		return FileResult{File: name, Error: err.Error()}
	}

	if err := s.uploader.Upload(ctx, name, f); err != nil {
		logs.Error("upload file: ", err)
		return FileResult{File: name, Error: err.Error()}
	}

	return FileResult{File: name}
}
