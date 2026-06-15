package backup

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Defaults  Defaults        `yaml:"defaults"`
	Report    ReportConfig    `yaml:"report"`
	Identity  IdentityConfig  `yaml:"identity"`
	Retention RetentionConfig `yaml:"retention"`
	Sources   []SourceConfig  `yaml:"sources"`
	Targets   []TargetConfig  `yaml:"targets"`
	Jobs      []JobConfig     `yaml:"jobs"`
}

type Defaults struct {
	Compress        string `yaml:"compress"`
	TimestampFormat string `yaml:"timestamp_format"`
	Concurrency     int    `yaml:"concurrency"`
	WorkDir         string `yaml:"work_dir"`
	ManifestPath    string `yaml:"manifest_path"`
}

type ReportConfig struct {
	Enabled bool              `yaml:"enabled"`
	URL     string            `yaml:"url"`
	Token   string            `yaml:"token"`
	Headers map[string]string `yaml:"headers"`
	Timeout string            `yaml:"timeout"`
	Retry   RetryConfig       `yaml:"retry"`
}

type IdentityConfig struct {
	Project   string `yaml:"project"`
	Cluster   string `yaml:"cluster"`
	MachineID string `yaml:"machine_id"`
}

type RetentionConfig struct {
	KeepLast int `yaml:"keep_last"`
	KeepDays int `yaml:"keep_days"`
}

type RetryConfig struct {
	Attempts int    `yaml:"attempts"`
	Backoff  string `yaml:"backoff"`
}

type SourceConfig struct {
	Name       string           `yaml:"name"`
	Type       string           `yaml:"type"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Mongo      MongoConfig      `yaml:"mongo"`
	Redis      RedisConfig      `yaml:"redis"`
	File       FileConfig       `yaml:"file"`
	ConfigRepo ConfigRepoConfig `yaml:"config"`
}

type MySQLConfig struct {
	Host               string   `yaml:"host"`
	Port               int      `yaml:"port"`
	User               string   `yaml:"user"`
	Password           string   `yaml:"password"`
	Database           string   `yaml:"database"`
	RestoreDatabase    string   `yaml:"restore_database"`
	Version            string   `yaml:"version"`
	DumpTool           string   `yaml:"dump_tool"`
	RestoreTool        string   `yaml:"restore_tool"`
	Mode               string   `yaml:"mode"`
	XtraBackupTool     string   `yaml:"xtrabackup_tool"`
	XtraBackupDir      string   `yaml:"xtrabackup_dir"`
	XtraCommandDir     string   `yaml:"xtrabackup_command_dir"`
	XtraRestoreDir     string   `yaml:"xtrabackup_restore_dir"`
	IncrementalBaseDir string   `yaml:"incremental_base_dir"`
	SingleTransaction  bool     `yaml:"single_transaction"`
	Quick              bool     `yaml:"quick"`
	Routines           bool     `yaml:"routines"`
	Triggers           bool     `yaml:"triggers"`
	Events             bool     `yaml:"events"`
	SetGTIDPurged      string   `yaml:"set_gtid_purged"`
	NoCreateDB         bool     `yaml:"no_create_db"`
	ExtraArgs          []string `yaml:"extra_args"`
	RestoreExtraArgs   []string `yaml:"restore_extra_args"`
	XtraExtraArgs      []string `yaml:"xtrabackup_extra_args"`
	PrepareExtraArgs   []string `yaml:"prepare_extra_args"`
}

type PostgresConfig struct {
	Host             string   `yaml:"host"`
	Port             int      `yaml:"port"`
	User             string   `yaml:"user"`
	Password         string   `yaml:"password"`
	Database         string   `yaml:"database"`
	RestoreDatabase  string   `yaml:"restore_database"`
	DumpTool         string   `yaml:"dump_tool"`
	RestoreTool      string   `yaml:"restore_tool"`
	Mode             string   `yaml:"mode"`
	Format           string   `yaml:"format"`
	Jobs             int      `yaml:"jobs"`
	ExtraArgs        []string `yaml:"extra_args"`
	RestoreExtraArgs []string `yaml:"restore_extra_args"`
}

type MongoConfig struct {
	URI              string   `yaml:"uri"`
	Database         string   `yaml:"database"`
	DumpTool         string   `yaml:"dump_tool"`
	RestoreTool      string   `yaml:"restore_tool"`
	ExtraArgs        []string `yaml:"extra_args"`
	RestoreExtraArgs []string `yaml:"restore_extra_args"`
}

type RedisConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Password    string `yaml:"password"`
	DB          int    `yaml:"db"`
	Mode        string `yaml:"mode"`
	CLIPath     string `yaml:"cli_path"`
	RDBPath     string `yaml:"rdb_path"`
	RestorePath string `yaml:"restore_path"`
}

type FileConfig struct {
	Paths         []string `yaml:"paths"`
	ArchiveTool   string   `yaml:"archive_tool"`
	RestoreDir    string   `yaml:"restore_dir"`
	Exclude       []string `yaml:"exclude"`
	FollowSymlink bool     `yaml:"follow_symlink"`
}

type ConfigRepoConfig struct {
	Path        string `yaml:"path"`
	Mode        string `yaml:"mode"`
	ArchiveTool string `yaml:"archive_tool"`
	RestoreDir  string `yaml:"restore_dir"`
	TagPrefix   string `yaml:"tag_prefix"`
}

type TargetConfig struct {
	Name  string      `yaml:"name"`
	Type  string      `yaml:"type"`
	Local LocalTarget `yaml:"local"`
	S3    S3Target    `yaml:"s3"`
}

type LocalTarget struct {
	Path string `yaml:"path"`
}

type S3Target struct {
	Endpoint         string      `yaml:"endpoint"`
	Bucket           string      `yaml:"bucket"`
	Region           string      `yaml:"region"`
	AccessKey        string      `yaml:"access_key"`
	SecretKey        string      `yaml:"secret_key"`
	SessionToken     string      `yaml:"session_token"`
	Prefix           string      `yaml:"prefix"`
	UseSSL           bool        `yaml:"use_ssl"`
	InsecureSkipTLS  bool        `yaml:"insecure_skip_tls"`
	ForcePathStyle   bool        `yaml:"force_path_style"`
	ContentType      string      `yaml:"content_type"`
	RetentionDaysTag int         `yaml:"retention_days_tag"`
	Timeout          string      `yaml:"timeout"`
	Retry            RetryConfig `yaml:"retry"`
}

type JobConfig struct {
	Name    string   `yaml:"name"`
	Source  string   `yaml:"source"`
	Targets []string `yaml:"targets"`
}

func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	expanded := os.ExpandEnv(string(raw))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, err
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) ApplyDefaults() {
	if c.Defaults.Compress == "" {
		c.Defaults.Compress = "gzip"
	}
	if c.Defaults.TimestampFormat == "" {
		c.Defaults.TimestampFormat = "20060102-150405"
	}
	if c.Defaults.Concurrency <= 0 {
		c.Defaults.Concurrency = 1
	}
	if c.Defaults.WorkDir == "" {
		c.Defaults.WorkDir = "./backups"
	}
	if c.Defaults.ManifestPath == "" {
		c.Defaults.ManifestPath = "./backups/manifest.jsonl"
	}

	for i := range c.Sources {
		src := &c.Sources[i]
		src.Type = strings.ToLower(src.Type)
		switch src.Type {
		case "mysql":
			if src.MySQL.Port == 0 {
				src.MySQL.Port = 3306
			}
			if src.MySQL.Version == "" {
				src.MySQL.Version = "mysql80"
			}
			src.MySQL.Version = normalizeMySQLVersion(src.MySQL.Version)
			if src.MySQL.DumpTool == "" {
				src.MySQL.DumpTool = DefaultDumpToolName("mysqldump")
			}
			if src.MySQL.RestoreTool == "" {
				src.MySQL.RestoreTool = DefaultDumpToolName("mysql")
			}
			if src.MySQL.Mode == "" {
				src.MySQL.Mode = "mysqldump"
			}
			src.MySQL.Mode = strings.ToLower(src.MySQL.Mode)
			if src.MySQL.XtraBackupTool == "" {
				src.MySQL.XtraBackupTool = DefaultDumpToolName("xtrabackup")
			}
			if !src.MySQL.SingleTransaction {
				src.MySQL.SingleTransaction = true
			}
			if !src.MySQL.Quick {
				src.MySQL.Quick = true
			}
		case "postgres", "postgresql":
			src.Type = "postgres"
			if src.Postgres.Port == 0 {
				src.Postgres.Port = 5432
			}
			if src.Postgres.DumpTool == "" {
				src.Postgres.DumpTool = DefaultDumpToolName("pg_dump")
			}
			if src.Postgres.RestoreTool == "" {
				src.Postgres.RestoreTool = DefaultDumpToolName("pg_restore")
			}
			if src.Postgres.Mode == "" {
				src.Postgres.Mode = "pg_dump"
			}
			if src.Postgres.Format == "" {
				src.Postgres.Format = "custom"
			}
			if src.Postgres.Jobs <= 0 {
				src.Postgres.Jobs = 1
			}
		case "mongo", "mongodb":
			src.Type = "mongo"
			if src.Mongo.DumpTool == "" {
				src.Mongo.DumpTool = DefaultDumpToolName("mongodump")
			}
			if src.Mongo.RestoreTool == "" {
				src.Mongo.RestoreTool = DefaultDumpToolName("mongorestore")
			}
		case "redis":
			if src.Redis.Port == 0 {
				src.Redis.Port = 6379
			}
			if src.Redis.Mode == "" {
				src.Redis.Mode = "bgsave"
			}
			if src.Redis.CLIPath == "" {
				src.Redis.CLIPath = DefaultDumpToolName("redis-cli")
			}
		case "file":
			if src.File.ArchiveTool == "" {
				src.File.ArchiveTool = defaultArchiveTool()
			}
		case "config":
			if src.ConfigRepo.Mode == "" {
				src.ConfigRepo.Mode = "zip"
			}
			if src.ConfigRepo.ArchiveTool == "" {
				src.ConfigRepo.ArchiveTool = defaultArchiveTool()
			}
			if src.ConfigRepo.TagPrefix == "" {
				src.ConfigRepo.TagPrefix = "dbbackup233"
			}
		}
	}
}

func (c Config) Validate() error {
	if c.Defaults.Compress != "gzip" && c.Defaults.Compress != "none" {
		return fmt.Errorf("defaults.compress must be gzip or none")
	}
	if len(c.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	if len(c.Jobs) == 0 {
		return fmt.Errorf("at least one job is required")
	}

	sourceNames := map[string]struct{}{}
	for _, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("source name is required")
		}
		if _, ok := sourceNames[src.Name]; ok {
			return fmt.Errorf("duplicate source %q", src.Name)
		}
		sourceNames[src.Name] = struct{}{}
		if err := validateSource(src); err != nil {
			return err
		}
	}

	targetNames := map[string]struct{}{}
	for _, target := range c.Targets {
		if target.Name == "" {
			return fmt.Errorf("target name is required")
		}
		if _, ok := targetNames[target.Name]; ok {
			return fmt.Errorf("duplicate target %q", target.Name)
		}
		targetNames[target.Name] = struct{}{}
		switch strings.ToLower(target.Type) {
		case "local":
			if target.Local.Path == "" {
				return fmt.Errorf("local target %q requires local.path", target.Name)
			}
		case "s3", "oss", "tos":
			if target.S3.Endpoint == "" || target.S3.Bucket == "" || target.S3.AccessKey == "" || target.S3.SecretKey == "" {
				return fmt.Errorf("s3 target %q requires endpoint, bucket, access_key, and secret_key", target.Name)
			}
		default:
			return fmt.Errorf("target %q has unsupported type %q", target.Name, target.Type)
		}
	}

	for _, job := range c.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job name is required")
		}
		if _, ok := sourceNames[job.Source]; !ok {
			return fmt.Errorf("job %q references unknown source %q", job.Name, job.Source)
		}
		if len(job.Targets) == 0 {
			return fmt.Errorf("job %q requires at least one target", job.Name)
		}
		for _, target := range job.Targets {
			if _, ok := targetNames[target]; !ok {
				return fmt.Errorf("job %q references unknown target %q", job.Name, target)
			}
		}
	}
	return nil
}

func validateSource(src SourceConfig) error {
	switch src.Type {
	case "mysql":
		if src.MySQL.Host == "" || src.MySQL.User == "" || src.MySQL.Database == "" {
			return fmt.Errorf("mysql source %q requires host, user, and database", src.Name)
		}
		if src.MySQL.Version != "" && !isValidMySQLVersion(src.MySQL.Version) {
			return fmt.Errorf("mysql source %q version must be one of mysql57, mysql80", src.Name)
		}
		if !isValidMySQLMode(src.MySQL.Mode) {
			return fmt.Errorf("mysql source %q mode must be one of mysqldump, xtrabackup-full, xtrabackup-incremental", src.Name)
		}
		if isXtraBackupMode(src.MySQL.Mode) && src.MySQL.XtraBackupDir == "" {
			return fmt.Errorf("mysql source %q xtrabackup mode requires xtrabackup_dir", src.Name)
		}
	case "postgres":
		if src.Postgres.Host == "" || src.Postgres.User == "" || src.Postgres.Database == "" {
			return fmt.Errorf("postgres source %q requires host, user, and database", src.Name)
		}
	case "mongo":
		if src.Mongo.URI == "" {
			return fmt.Errorf("mongo source %q requires uri", src.Name)
		}
	case "redis":
		if src.Redis.Mode == "copy_rdb" && src.Redis.RDBPath == "" {
			return fmt.Errorf("redis source %q copy_rdb mode requires rdb_path", src.Name)
		}
	case "file":
		if len(src.File.Paths) == 0 {
			return fmt.Errorf("file source %q requires paths", src.Name)
		}
	case "config":
		if src.ConfigRepo.Path == "" {
			return fmt.Errorf("config source %q requires path", src.Name)
		}
	default:
		return fmt.Errorf("source %q has unsupported type %q", src.Name, src.Type)
	}
	return nil
}

func isValidMySQLMode(mode string) bool {
	switch strings.ToLower(mode) {
	case "mysqldump", "xtrabackup-full", "xtrabackup-incremental":
		return true
	default:
		return false
	}
}

func isXtraBackupMode(mode string) bool {
	switch strings.ToLower(mode) {
	case "xtrabackup-full", "xtrabackup-incremental":
		return true
	default:
		return false
	}
}

func isValidMySQLVersion(version string) bool {
	switch strings.ToLower(version) {
	case "mysql57", "mysql80":
		return true
	default:
		return false
	}
}

func normalizeMySQLVersion(version string) string {
	switch strings.ToLower(version) {
	case "5.7", "57":
		return "mysql57"
	case "8.0", "80":
		return "mysql80"
	default:
		return strings.ToLower(version)
	}
}

func DefaultDumpToolName(tool string) string {
	if runtime.GOOS == "windows" {
		return tool + ".exe"
	}
	return tool
}

func defaultArchiveTool() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "tar"
}
