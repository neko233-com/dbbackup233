package backup

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

type VerifyResult struct {
	OK       bool
	Warnings []string
	Errors   []string
}

func VerifyConfig(cfg Config) VerifyResult {
	cfg.ApplyDefaults()
	result := VerifyResult{OK: true}
	if err := cfg.Validate(); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.OK = false
		return result
	}

	if err := ensureWritableDir(cfg.Defaults.WorkDir); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("work_dir is not writable: %v", err))
	}
	if err := ensureWritableDir(filepath.Dir(cfg.Defaults.ManifestPath)); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("manifest directory is not writable: %v", err))
	}
	if cfg.Report.Enabled {
		if _, err := url.ParseRequestURI(cfg.Report.URL); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("report.url is invalid: %v", err))
		}
	}

	for _, target := range cfg.Targets {
		if target.Type == "local" {
			if err := ensureWritableDir(target.Local.Path); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("target %s is not writable: %v", target.Name, err))
			}
		}
	}

	for _, source := range cfg.Sources {
		switch source.Type {
		case "mysql":
			checkTool(&result, source.Name, source.MySQL.DumpTool)
			checkTool(&result, source.Name, source.MySQL.RestoreTool)
		case "postgres":
			checkTool(&result, source.Name, source.Postgres.DumpTool)
			checkTool(&result, source.Name, source.Postgres.RestoreTool)
		case "file":
			for _, path := range source.File.Paths {
				if _, err := os.Stat(path); err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("file source %s path check failed: %v", source.Name, err))
				}
			}
			checkTool(&result, source.Name, source.File.ArchiveTool)
		}
	}

	result.OK = len(result.Errors) == 0
	return result
}

func checkTool(result *VerifyResult, sourceName, tool string) {
	if tool == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("source %s has empty tool path", sourceName))
		return
	}
	if _, err := exec.LookPath(tool); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("source %s tool %s not found in PATH: %v", sourceName, tool, err))
	}
}

func ensureWritableDir(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(path, ".dbbackup233-write-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}
