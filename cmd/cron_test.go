package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestWindowsScheduleCommandSupportsHourly(t *testing.T) {
	got := strings.Join(windowsScheduleCommand("@hourly", "dbbackup233", `"dbbackup233.exe" backup -c "config.yaml"`), " ")
	if !strings.Contains(got, "/SC HOURLY") || !strings.Contains(got, "/TN dbbackup233") {
		t.Fatalf("unexpected command: %s", got)
	}
}

func TestCronListAndRemoveCommands(t *testing.T) {
	list := strings.Join(cronListCommand("dbbackup233"), " ")
	remove := strings.Join(cronRemoveCommand("dbbackup233"), " ")
	if runtime.GOOS == "windows" {
		if !strings.Contains(list, "schtasks /Query") || !strings.Contains(remove, "schtasks /Delete") {
			t.Fatalf("unexpected windows commands: list=%s remove=%s", list, remove)
		}
		return
	}
	if !strings.Contains(list, "crontab -l") || !strings.Contains(remove, "grep -v") {
		t.Fatalf("unexpected unix commands: list=%s remove=%s", list, remove)
	}
}
