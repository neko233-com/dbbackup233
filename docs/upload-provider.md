# Upload Provider

`dbbackup233` implements cloud uploads through the S3-compatible API.

## Current Dependency Version

The current implementation uses:

- `github.com/minio/minio-go/v7 v7.0.95`

This SDK speaks the AWS S3 API and is commonly used against compatible object
storage services:

- AWS S3
- Aliyun OSS S3-compatible endpoint
- Volcengine TOS S3-compatible endpoint
- Tencent COS S3-compatible endpoint
- MinIO

Cloud-vendor native SDKs are not required for the current MVP.

## Config Mapping

Use `type: oss`, `type: tos`, or `type: s3`; all three are routed to the same
S3-compatible upload provider.

```yaml
targets:
  - name: "oss"
    type: oss
    s3:
      endpoint: "oss-cn-shanghai.aliyuncs.com"
      bucket: "my-backups"
      region: "oss-cn-shanghai"
      access_key: "${OSS_ACCESS_KEY_ID}"
      secret_key: "${OSS_ACCESS_KEY_SECRET}"
      prefix: "dbbackup233"
      use_ssl: true
```

## Automated Verification

Real cloud credentials are not required for tests. The Docker integration test
starts a local S3-compatible mock endpoint and verifies:

- the upload API is called with `PUT`
- the uploaded object body is received
- AWS streaming chunked payloads are decoded
- the uploaded payload is a valid gzip file
- the gzip content contains the MySQL dump SQL

This proves the dump artifact passed through the upload provider correctly. Real
vendor acceptance still depends on credentials, bucket policy, endpoint, and
network access in the deployment environment.
