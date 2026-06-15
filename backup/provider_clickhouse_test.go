package backup

import (
	"strings"
	"testing"
)

func TestClickHouseBackupQuery(t *testing.T) {
	cfg := ClickHouseConfig{
		Database:          "game",
		Table:             "events",
		Mode:              "incremental",
		BackupDestination: "Disk('backups', 'events-inc.zip')",
		BaseBackup:        "Disk('backups', 'events-full.zip')",
	}
	got := buildClickHouseBackupQuery(cfg)
	for _, want := range []string{"BACKUP TABLE `game`.`events`", "TO Disk('backups', 'events-inc.zip')", "SETTINGS base_backup = Disk('backups', 'events-full.zip')"} {
		if !strings.Contains(got, want) {
			t.Fatalf("query %q missing %q", got, want)
		}
	}
}

func TestClickHouseRestoreQuery(t *testing.T) {
	cfg := ClickHouseConfig{Database: "game", BackupDestination: "Disk('backups', 'game.zip')"}
	got := buildClickHouseRestoreQuery(cfg)
	want := "RESTORE DATABASE `game` FROM Disk('backups', 'game.zip')"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
