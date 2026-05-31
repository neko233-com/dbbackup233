package backup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func AppendManifest(path string, art BackupArtifact) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(art)
}

func ListArtifacts(path string) ([]BackupArtifact, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var out []BackupArtifact
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var art BackupArtifact
		if err := json.Unmarshal(scanner.Bytes(), &art); err != nil {
			return nil, err
		}
		out = append(out, art)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func FindArtifact(path, jobName, version string) (BackupArtifact, error) {
	arts, err := ListArtifacts(path)
	if err != nil {
		return BackupArtifact{}, err
	}
	for _, art := range arts {
		if art.JobName != jobName {
			continue
		}
		if version == "" || art.Version == version {
			return art, nil
		}
	}
	if version == "" {
		return BackupArtifact{}, fmt.Errorf("no backup history found for job %q", jobName)
	}
	return BackupArtifact{}, fmt.Errorf("no backup history found for job %q version %q", jobName, version)
}
