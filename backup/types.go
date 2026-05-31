package backup

import "time"

type RunnerOptions struct {
	DryRun bool
	Logf   func(format string, args ...any)
}

type BackupArtifact struct {
	Version     string    `json:"version"`
	JobName     string    `json:"job_name"`
	SourceName  string    `json:"source_name"`
	SourceType  string    `json:"source_type"`
	FilePath    string    `json:"file_path"`
	ObjectName  string    `json:"object_name"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
	Targets     []string  `json:"targets"`
	RestoreHint string    `json:"restore_hint,omitempty"`
}

type ReportEvent struct {
	Event      string          `json:"event"`
	Project    string          `json:"project,omitempty"`
	Cluster    string          `json:"cluster,omitempty"`
	MachineID  string          `json:"machine_id,omitempty"`
	JobName    string          `json:"job_name,omitempty"`
	SourceName string          `json:"source_name,omitempty"`
	SourceType string          `json:"source_type,omitempty"`
	Version    string          `json:"version,omitempty"`
	Status     string          `json:"status"`
	Error      string          `json:"error,omitempty"`
	Artifact   *BackupArtifact `json:"artifact,omitempty"`
	StartedAt  time.Time       `json:"started_at,omitempty"`
	FinishedAt time.Time       `json:"finished_at,omitempty"`
}

type Provider interface {
	Backup(ctx Context, source SourceConfig, artifactPath string) error
	Restore(ctx Context, source SourceConfig, artifactPath string) error
	Extension(source SourceConfig, compress string) string
}

type Context struct {
	DryRun bool
	Logf   func(format string, args ...any)
}
