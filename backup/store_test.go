package backup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreS3RetriesUpload(t *testing.T) {
	puts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
			return
		}
		if r.Method == http.MethodPut {
			puts++
			if puts == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("ETag", `"etag"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	root := t.TempDir()
	file := filepath.Join(root, "artifact.sql.gz")
	if err := os.WriteFile(file, []byte("backup"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := TargetConfig{
		Name: "oss", Type: "s3",
		S3: S3Target{
			Endpoint: strings.TrimPrefix(server.URL, "http://"),
			Bucket:   "bucket", AccessKey: "ak", SecretKey: "sk", Prefix: "p",
			ForcePathStyle: true,
			Retry:          RetryConfig{Attempts: 2, Backoff: "1ms"},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := storeArtifact(ctx, target, BackupArtifact{SourceName: "s", FilePath: file, ObjectName: "artifact.sql.gz"})
	if err != nil {
		t.Fatal(err)
	}
	if puts != 2 {
		t.Fatalf("expected 2 PUT attempts, got %d", puts)
	}
}
