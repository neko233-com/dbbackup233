package backup

import "testing"

func TestValidateRequiresKnownJobReferences(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{{
			Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", Port: 3306, User: "backup", Database: "game"},
		}},
		Targets: []TargetConfig{{
			Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"},
		}},
		Jobs: []JobConfig{{
			Name: "game", Source: "missing", Targets: []string{"local"},
		}},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBackupObjectName(t *testing.T) {
	job := JobConfig{Name: "Game Main"}
	src := SourceConfig{Name: "Game Main", Type: "mysql"}
	version := mustParseTime(t, "2026-05-31T10:00:00Z").Format("20060102-150405")
	got := artifactObjectName(job, src, "sql.gz", version)
	want := "mysql/game-main-20260531-100000.sql.gz"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMySQLVersionEnum(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{{
			Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", User: "backup", Database: "game", Version: "mysql56"},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"}}},
		Jobs:    []JobConfig{{Name: "game", Source: "game", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid mysql version error")
	}
}

func TestMySQLVersionNormalize(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{{
			Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", User: "backup", Database: "game", Version: "8.0"},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"}}},
		Jobs:    []JobConfig{{Name: "game", Source: "game", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if got := cfg.Sources[0].MySQL.Version; got != "mysql80" {
		t.Fatalf("got %q", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLXtraBackupModeValidation(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{{
			Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", User: "backup", Database: "game", Mode: "xtrabackup-full"},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"}}},
		Jobs:    []JobConfig{{Name: "game", Source: "game", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing xtrabackup_dir error")
	}
}

func TestMySQLPhysicalModeDefaults(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{{
			Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", User: "backup", Database: "game", Mode: "incremental", XtraBackupDir: "./xb"},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"}}},
		Jobs:    []JobConfig{{Name: "game", Source: "game", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if got := cfg.Sources[0].MySQL.Mode; got != "xtrabackup-incremental" {
		t.Fatalf("got mode %q", got)
	}
	if got := cfg.Sources[0].MySQL.PhysicalEngine; got != "xtrabackup" {
		t.Fatalf("got engine %q", got)
	}
	if got := cfg.Sources[0].MySQL.PhysicalFormat; got != "zip" {
		t.Fatalf("got format %q", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdditionalProviderDefaults(t *testing.T) {
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", Concurrency: 1},
		Sources: []SourceConfig{
			{Name: "es", Type: "elastic", Elastic: ElasticConfig{URL: "http://127.0.0.1:9200", Repository: "repo"}},
			{Name: "ch", Type: "clickhouse", ClickHouse: ClickHouseConfig{Host: "127.0.0.1", Database: "game", BackupDestination: "Disk('backups', 'game.zip')"}},
			{Name: "dir", Type: "directory", File: FileConfig{Paths: []string{"./data"}}},
			{Name: "cmd", Type: "exec", Command: CommandConfig{BackupCommand: []string{"tool", "backup"}}},
		},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: "./backups"}}},
		Jobs: []JobConfig{
			{Name: "es", Source: "es", Targets: []string{"local"}},
			{Name: "ch", Source: "ch", Targets: []string{"local"}},
			{Name: "dir", Source: "dir", Targets: []string{"local"}},
			{Name: "cmd", Source: "cmd", Targets: []string{"local"}},
		},
	}
	cfg.ApplyDefaults()
	if cfg.Sources[0].Type != "elasticsearch" || cfg.Sources[0].Elastic.Mode != "snapshot" {
		t.Fatalf("bad elastic defaults: %+v", cfg.Sources[0])
	}
	if cfg.Sources[1].ClickHouse.Port != 9000 || cfg.Sources[1].ClickHouse.Mode != "full" {
		t.Fatalf("bad clickhouse defaults: %+v", cfg.Sources[1])
	}
	if cfg.Sources[2].Type != "file" {
		t.Fatalf("bad file alias: %+v", cfg.Sources[2])
	}
	if cfg.Sources[3].Type != "command" || cfg.Sources[3].Command.Extension != "artifact" {
		t.Fatalf("bad command defaults: %+v", cfg.Sources[3])
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
