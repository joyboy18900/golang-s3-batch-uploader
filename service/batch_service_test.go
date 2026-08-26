package service_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"golang-s3-batch-uploader/mock/mock_repository"
	"golang-s3-batch-uploader/service"

	"go.uber.org/mock/gomock"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", name, err)
	}
}

func fileSet(results []service.FileResult) map[string]bool {
	set := make(map[string]bool, len(results))
	for _, r := range results {
		set[r.File] = true
	}
	return set
}

func TestBatchService_Run(t *testing.T) {
	tests := []struct {
		name          string
		setupDir      func(t *testing.T, dir string)
		setupUploader func(u *mock_repository.MockUploader)
		wantSucceeded []string
		wantFailed    []string
	}{
		{
			name: "all files succeed",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "a.csv", "id,name\n1,foo\n")
				writeFile(t, dir, "b.csv", "id,name\n2,bar\n")
			},
			setupUploader: func(u *mock_repository.MockUploader) {
				u.EXPECT().Upload(gomock.Any(), "a.csv", gomock.Any()).Return(nil)
				u.EXPECT().Upload(gomock.Any(), "b.csv", gomock.Any()).Return(nil)
			},
			wantSucceeded: []string{"a.csv", "b.csv"},
		},
		{
			name: "malformed csv fails without stopping the rest",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "good.csv", "id,name\n1,foo\n")
				writeFile(t, dir, "bad.csv", "id,name\n\"unterminated,foo\n")
			},
			setupUploader: func(u *mock_repository.MockUploader) {
				u.EXPECT().Upload(gomock.Any(), "good.csv", gomock.Any()).Return(nil)
			},
			wantSucceeded: []string{"good.csv"},
			wantFailed:    []string{"bad.csv"},
		},
		{
			name: "upload error on one file does not stop the rest",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "ok.csv", "id,name\n1,foo\n")
				writeFile(t, dir, "flaky.csv", "id,name\n2,bar\n")
			},
			setupUploader: func(u *mock_repository.MockUploader) {
				u.EXPECT().Upload(gomock.Any(), "ok.csv", gomock.Any()).Return(nil)
				u.EXPECT().Upload(gomock.Any(), "flaky.csv", gomock.Any()).Return(errors.New("s3 unavailable"))
			},
			wantSucceeded: []string{"ok.csv"},
			wantFailed:    []string{"flaky.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uploader := mock_repository.NewMockUploader(ctrl)
			tt.setupUploader(uploader)

			dir := t.TempDir()
			tt.setupDir(t, dir)

			svc := service.NewBatchService(uploader, 2)
			result, err := svc.Run(context.Background(), service.BatchRequest{SourceDir: dir})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			gotSucceeded := fileSet(result.Succeeded)
			gotFailed := fileSet(result.Failed)

			for _, name := range tt.wantSucceeded {
				if !gotSucceeded[name] {
					t.Errorf("expected %s in succeeded, got succeeded=%v", name, result.Succeeded)
				}
			}
			for _, name := range tt.wantFailed {
				if !gotFailed[name] {
					t.Errorf("expected %s in failed, got failed=%v", name, result.Failed)
				}
			}
			if len(gotSucceeded)+len(gotFailed) != len(tt.wantSucceeded)+len(tt.wantFailed) {
				t.Errorf("unexpected result counts: succeeded=%v failed=%v", result.Succeeded, result.Failed)
			}
		})
	}
}

type concurrencyTrackingUploader struct {
	inFlight  atomic.Int32
	highWater atomic.Int32
}

func (u *concurrencyTrackingUploader) Upload(_ context.Context, _ string, body io.Reader) error {
	if _, err := io.ReadAll(body); err != nil {
		return err
	}

	current := u.inFlight.Add(1)
	defer u.inFlight.Add(-1)

	for {
		high := u.highWater.Load()
		if current <= high || u.highWater.CompareAndSwap(high, current) {
			break
		}
	}

	time.Sleep(20 * time.Millisecond)
	return nil
}

func TestBatchService_Run_PoolIsBounded(t *testing.T) {
	const workerCount = 3

	dir := t.TempDir()
	for i := 0; i < workerCount*3; i++ {
		writeFile(t, dir, fmt.Sprintf("file%d.csv", i), "id,name\n1,foo\n")
	}

	uploader := &concurrencyTrackingUploader{}
	svc := service.NewBatchService(uploader, workerCount)

	if _, err := svc.Run(context.Background(), service.BatchRequest{SourceDir: dir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if high := uploader.highWater.Load(); high > workerCount {
		t.Errorf("observed %d concurrent uploads, want <= %d (worker pool must stay bounded)", high, workerCount)
	}
}
