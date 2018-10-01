package procfind

import (
	"testing"
)

func TestReadProcCmdline(t *testing.T) {
	argv := readProcCmdline("/proc/1/cmdline")
	if len(argv) == 0 {
		t.Errorf("failed to read cmdline of pid 1")
	}
}

func TestReadProcCmdlineInexistent(t *testing.T) {
	argv := readProcCmdline("/proc/0/cmdline")
	if len(argv) > 0 {
		t.Errorf("Unexpected data for pid 0: %#v", argv)
	}
}
