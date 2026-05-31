package backup

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func storeArtifact(ctx context.Context, target TargetConfig, art BackupArtifact) error {
	switch strings.ToLower(target.Type) {
	case "local":
		return storeLocal(target.Local, art)
	case "s3":
		return storeS3(ctx, target.S3, art)
	default:
		return fmt.Errorf("unsupported target type %q", target.Type)
	}
}

func storeLocal(target LocalTarget, art BackupArtifact) error {
	dst := filepath.Join(target.Path, art.ObjectName)
	srcAbs, err := filepath.Abs(art.FilePath)
	if err != nil {
		return err
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if srcAbs == dstAbs {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	src, err := os.Open(art.FilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(src); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func storeS3(ctx context.Context, target S3Target, art BackupArtifact) error {
	attempts := normalizeAttempts(target.Retry.Attempts)
	backoff := parseBackoff(target.Retry.Backoff)
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		err := storeS3Once(ctx, target, art)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < attempts {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return lastErr
}

func storeS3Once(ctx context.Context, target S3Target, art BackupArtifact) error {
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(target.AccessKey, target.SecretKey, target.SessionToken),
		Secure: target.UseSSL,
		Region: target.Region,
	}
	transport := &http.Transport{}
	if target.InsecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	if target.InsecureSkipTLS {
		opts.Transport = transport
	}
	if target.ForcePathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(target.Endpoint, opts)
	if err != nil {
		return err
	}

	objectName := path.Join(target.Prefix, art.ObjectName)
	contentType := target.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err = client.FPutObject(ctx, target.Bucket, objectName, art.FilePath, minio.PutObjectOptions{
		ContentType: contentType,
		UserTags: map[string]string{
			"source": art.SourceName,
			"tool":   "dbbackup233",
		},
	})
	return err
}
