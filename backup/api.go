package backup

import "context"

// Run loads a config file and executes its backup jobs.
func Run(ctx context.Context, configPath string, opt RunnerOptions) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	return NewRunner(cfg, opt).Run(ctx)
}

// RunConfig executes backup jobs from an already constructed config.
func RunConfig(ctx context.Context, cfg Config, opt RunnerOptions) error {
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	return NewRunner(cfg, opt).Run(ctx)
}

func RunSelected(ctx context.Context, configPath string, jobNames []string, opt RunnerOptions) error {
	opt.JobNames = append([]string(nil), jobNames...)
	return Run(ctx, configPath, opt)
}

func RunRestore(ctx context.Context, configPath, jobName, version string, opt RunnerOptions) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	return NewBackupManager(cfg, opt).Restore(ctx, jobName, version)
}

func ListHistory(configPath string) ([]BackupArtifact, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return ListArtifacts(cfg.Defaults.ManifestPath)
}
