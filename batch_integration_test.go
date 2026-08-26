package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang-s3-batch-uploader/handler"
	"golang-s3-batch-uploader/service"

	"github.com/gofiber/fiber/v2"
)

type fakeUploader struct {
	mu       sync.Mutex
	uploaded map[string]bool
	failKeys map[string]bool
}

func newFakeUploader(failKeys ...string) *fakeUploader {
	fail := make(map[string]bool, len(failKeys))
	for _, k := range failKeys {
		fail[k] = true
	}
	return &fakeUploader{uploaded: make(map[string]bool), failKeys: fail}
}

func (f *fakeUploader) Upload(_ context.Context, key string, body io.Reader) error {
	if _, err := io.ReadAll(body); err != nil {
		return err
	}
	if f.failKeys[key] {
		return errUploadFailed
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploaded[key] = true
	return nil
}

var errUploadFailed = errors.New("upload failed")

func newIntegrationTestApp(uploader *fakeUploader) *fiber.App {
	batchSvc := service.NewBatchService(uploader, 2)
	batchHdlr := handler.NewBatchHandler(batchSvc)

	app := fiber.New()
	app.Post("/batches", batchHdlr.Run)

	return app
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", name, err)
	}
}

func TestBatchIntegration_PartialFailureDoesNotStopBatch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "good1.csv", "id,name\n1,foo\n")
	writeTestFile(t, dir, "good2.csv", "id,name\n2,bar\n")
	writeTestFile(t, dir, "bad.csv", "id,name\n\"unterminated,foo\n")

	uploader := newFakeUploader()
	app := newIntegrationTestApp(uploader)

	body, _ := json.Marshal(service.BatchRequest{SourceDir: dir})
	req := httptest.NewRequest(http.MethodPost, "/batches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var envelope struct {
		Code    int                 `json:"code"`
		Message string              `json:"message"`
		Data    service.BatchResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(envelope.Data.Succeeded) != 2 {
		t.Errorf("succeeded count = %d, want 2 (got %v)", len(envelope.Data.Succeeded), envelope.Data.Succeeded)
	}
	if len(envelope.Data.Failed) != 1 {
		t.Errorf("failed count = %d, want 1 (got %v)", len(envelope.Data.Failed), envelope.Data.Failed)
	}
	if !uploader.uploaded["good1.csv"] || !uploader.uploaded["good2.csv"] {
		t.Errorf("expected both good files uploaded, got %v", uploader.uploaded)
	}
	if uploader.uploaded["bad.csv"] {
		t.Errorf("malformed file should never reach the uploader")
	}
}

func TestBatchIntegration_UploadFailureDoesNotStopBatch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "ok.csv", "id,name\n1,foo\n")
	writeTestFile(t, dir, "flaky.csv", "id,name\n2,bar\n")

	uploader := newFakeUploader("flaky.csv")
	app := newIntegrationTestApp(uploader)

	body, _ := json.Marshal(service.BatchRequest{SourceDir: dir})
	req := httptest.NewRequest(http.MethodPost, "/batches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var envelope struct {
		Code    int                 `json:"code"`
		Message string              `json:"message"`
		Data    service.BatchResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(envelope.Data.Succeeded) != 1 || envelope.Data.Succeeded[0].File != "ok.csv" {
		t.Errorf("succeeded = %v, want [ok.csv]", envelope.Data.Succeeded)
	}
	if len(envelope.Data.Failed) != 1 || envelope.Data.Failed[0].File != "flaky.csv" {
		t.Errorf("failed = %v, want [flaky.csv]", envelope.Data.Failed)
	}
}
