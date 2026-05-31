package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type PruneOptions struct {
	DryRun bool
	Logf   func(format string, args ...any)
}

type PruneResult struct {
	Deleted []BackupArtifact
	Kept    []BackupArtifact
}

func PruneManifest(manifestPath string, retention RetentionConfig, opt PruneOptions) (PruneResult, error) {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	arts, err := ListArtifacts(manifestPath)
	if err != nil {
		return PruneResult{}, err
	}
	if retention.KeepLast <= 0 && retention.KeepDays <= 0 {
		return PruneResult{Kept: arts}, nil
	}

	keep := map[int]bool{}
	byJob := map[string][]int{}
	for i, art := range arts {
		byJob[art.JobName] = append(byJob[art.JobName], i)
	}
	for _, indexes := range byJob {
		sort.Slice(indexes, func(i, j int) bool {
			return arts[indexes[i]].CreatedAt.After(arts[indexes[j]].CreatedAt)
		})
		for pos, idx := range indexes {
			if retention.KeepLast > 0 && pos < retention.KeepLast {
				keep[idx] = true
			}
		}
	}
	if retention.KeepDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -retention.KeepDays)
		for i, art := range arts {
			if art.CreatedAt.After(cutoff) {
				keep[i] = true
			}
		}
	}

	var result PruneResult
	for i, art := range arts {
		if keep[i] {
			result.Kept = append(result.Kept, art)
			continue
		}
		result.Deleted = append(result.Deleted, art)
		if !opt.DryRun && art.FilePath != "" {
			_ = os.Remove(art.FilePath)
		}
	}
	if !opt.DryRun {
		if err := rewriteManifest(manifestPath, result.Kept); err != nil {
			return PruneResult{}, err
		}
	}
	return result, nil
}

func rewriteManifest(path string, arts []BackupArtifact) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	for _, art := range arts {
		if err := enc.Encode(art); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
