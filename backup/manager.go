package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type BackupManager struct {
	cfg Config
	opt RunnerOptions
}

func NewBackupManager(cfg Config, opt RunnerOptions) *BackupManager {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	return &BackupManager{cfg: cfg, opt: opt}
}

func NewRunner(cfg Config, opt RunnerOptions) *BackupManager {
	return NewBackupManager(cfg, opt)
}

func (m *BackupManager) Run(ctx context.Context) error {
	return m.Backup(ctx)
}

func (m *BackupManager) Backup(ctx context.Context) error {
	sources := mapSources(m.cfg.Sources)
	targets := mapTargets(m.cfg.Targets)
	runCtx := Context{DryRun: m.opt.DryRun, Logf: m.opt.Logf}
	reporter := NewReporter(m.cfg.Report, m.cfg.Identity)
	jobs, err := selectJobs(m.cfg.Jobs, m.opt.JobNames)
	if err != nil {
		return err
	}

	sem := make(chan struct{}, m.cfg.Defaults.Concurrency)
	var wg sync.WaitGroup
	errs := make(chan error, len(jobs))

	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}

			src := sources[job.Source]
			if err := m.runBackupJob(ctx, runCtx, reporter, job, src, targetsForJob(job, targets)); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)
	return joinErrors(errs)
}

func (m *BackupManager) runBackupJob(ctx context.Context, runCtx Context, reporter Reporter, job JobConfig, source SourceConfig, targets []TargetConfig) error {
	provider, err := providerFor(source.Type)
	if err != nil {
		return err
	}
	now := time.Now()
	version := now.Format(m.cfg.Defaults.TimestampFormat)
	if err := reporter.Send(ctx, ReportEvent{
		Event: "backup.started", Status: "started", JobName: job.Name, SourceName: source.Name, SourceType: source.Type, Version: version, StartedAt: now,
	}); err != nil {
		m.opt.Logf("report backup start failed: %v", err)
	}
	objectName := artifactObjectName(job, source, provider.Extension(source, m.cfg.Defaults.Compress), version)
	localPath := filepath.Join(m.cfg.Defaults.WorkDir, objectName)
	m.opt.Logf("backup %s (%s) -> %s", job.Name, source.Type, localPath)

	if m.opt.DryRun {
		m.opt.Logf("dry-run artifact: %s", localPath)
		for _, target := range targets {
			m.opt.Logf("dry-run target: %s (%s)", target.Name, target.Type)
		}
		_ = reporter.Send(ctx, ReportEvent{Event: "backup.completed", Status: "dry-run", JobName: job.Name, SourceName: source.Name, SourceType: source.Type, Version: version, StartedAt: now, FinishedAt: time.Now()})
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	if err := provider.Backup(runCtx, source, localPath); err != nil {
		_ = reporter.Send(ctx, ReportEvent{Event: "backup.completed", Status: "failed", JobName: job.Name, SourceName: source.Name, SourceType: source.Type, Version: version, Error: err.Error(), StartedAt: now, FinishedAt: time.Now()})
		return fmt.Errorf("%s backup failed: %w", job.Name, err)
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	sum, err := SHA256File(localPath)
	if err != nil {
		return err
	}
	art := BackupArtifact{
		Version:    version,
		JobName:    job.Name,
		SourceName: source.Name,
		SourceType: source.Type,
		FilePath:   localPath,
		ObjectName: objectName,
		Size:       info.Size(),
		SHA256:     sum,
		CreatedAt:  now,
		Targets:    targetNames(targets),
	}

	for _, target := range targets {
		if err := storeArtifact(ctx, target, art); err != nil {
			return fmt.Errorf("%s store to %s failed: %w", job.Name, target.Name, err)
		}
		m.opt.Logf("stored %s -> %s", job.Name, target.Name)
	}
	if err := AppendManifest(m.cfg.Defaults.ManifestPath, art); err != nil {
		_ = reporter.Send(ctx, ReportEvent{Event: "backup.completed", Status: "failed", JobName: job.Name, SourceName: source.Name, SourceType: source.Type, Version: version, Error: err.Error(), StartedAt: now, FinishedAt: time.Now()})
		return err
	}
	if err := reporter.Send(ctx, ReportEvent{Event: "backup.completed", Status: "success", JobName: job.Name, SourceName: source.Name, SourceType: source.Type, Version: version, Artifact: &art, StartedAt: now, FinishedAt: time.Now()}); err != nil {
		m.opt.Logf("report backup complete failed: %v", err)
	}
	return nil
}

func (m *BackupManager) Restore(ctx context.Context, jobName, version string) error {
	art, err := FindArtifact(m.cfg.Defaults.ManifestPath, jobName, version)
	if err != nil {
		return err
	}
	source, ok := mapSources(m.cfg.Sources)[art.SourceName]
	if !ok {
		return fmt.Errorf("manifest source %q no longer exists in config", art.SourceName)
	}
	provider, err := providerFor(source.Type)
	if err != nil {
		return err
	}
	runCtx := Context{DryRun: m.opt.DryRun, Logf: m.opt.Logf}
	m.opt.Logf("restore %s version %s from %s", jobName, art.Version, art.FilePath)
	if m.opt.DryRun {
		return nil
	}
	if err := verifyArtifactChecksum(art); err != nil {
		return err
	}
	return provider.Restore(runCtx, source, art.FilePath)
}

func selectJobs(jobs []JobConfig, names []string) ([]JobConfig, error) {
	if len(names) == 0 {
		return jobs, nil
	}
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			wanted[name] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return jobs, nil
	}
	out := make([]JobConfig, 0, len(wanted))
	for _, job := range jobs {
		if _, ok := wanted[job.Name]; ok {
			out = append(out, job)
			delete(wanted, job.Name)
		}
	}
	if len(wanted) > 0 {
		missing := make([]string, 0, len(wanted))
		for name := range wanted {
			missing = append(missing, name)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("unknown backup job(s): %s", strings.Join(missing, ", "))
	}
	return out, nil
}

func verifyArtifactChecksum(art BackupArtifact) error {
	if art.SHA256 == "" {
		return fmt.Errorf("backup artifact %s has no sha256 in manifest; refuse restore without integrity metadata", art.Version)
	}
	got, err := SHA256File(art.FilePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, art.SHA256) {
		return fmt.Errorf("backup artifact checksum mismatch for %s: manifest=%s actual=%s", art.FilePath, art.SHA256, got)
	}
	return nil
}

func artifactObjectName(job JobConfig, source SourceConfig, ext, version string) string {
	return fmt.Sprintf("%s/%s-%s.%s", sanitizeName(source.Type), sanitizeName(job.Name), version, ext)
}

func mapSources(values []SourceConfig) map[string]SourceConfig {
	out := make(map[string]SourceConfig, len(values))
	for _, value := range values {
		out[value.Name] = value
	}
	return out
}

func mapTargets(values []TargetConfig) map[string]TargetConfig {
	out := make(map[string]TargetConfig, len(values))
	for _, value := range values {
		value.Type = strings.ToLower(value.Type)
		if value.Type == "oss" || value.Type == "tos" {
			value.Type = "s3"
		}
		out[value.Name] = value
	}
	return out
}

func targetsForJob(job JobConfig, targets map[string]TargetConfig) []TargetConfig {
	out := make([]TargetConfig, 0, len(job.Targets))
	for _, name := range job.Targets {
		out = append(out, targets[name])
	}
	return out
}

func targetNames(targets []TargetConfig) []string {
	out := make([]string, len(targets))
	for i, target := range targets {
		out[i] = target.Name
	}
	return out
}

func sanitizeName(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	var out strings.Builder
	for _, r := range v {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			out.WriteRune(r)
			continue
		}
		out.WriteByte('-')
	}
	return strings.Trim(out.String(), "-")
}

func joinErrors(errs <-chan error) error {
	var joined []string
	for err := range errs {
		if err != nil {
			joined = append(joined, err.Error())
		}
	}
	if len(joined) > 0 {
		return fmt.Errorf(strings.Join(joined, "; "))
	}
	return nil
}
