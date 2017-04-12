package procfind

import (
	"os"
	"testing"
)

func TestPathNoEnv(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Unsetenv("PATH")
	defer os.Setenv("PATH", oldPath)

	_, err := Path()
	if err != ErrEnvVarNotFound {
		t.Errorf("unexpected error: %s", err)
	}
}

func expect(t *testing.T, err error, val, exp string) {
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if val != exp {
		t.Errorf("unexpected result: %s (instead of %s)", val, exp)
	}
}

func TestPathWithEnv(t *testing.T) {
	testPath := "/bin:/sbin:/usr/bin:/usr/sbin"
	oldPath := os.Getenv("PATH")
	os.Unsetenv("PATH")
	os.Setenv("PATH", testPath)
	defer os.Setenv("PATH", oldPath)

	val, err := Path()
	expect(t, err, val, testPath)
}

func TestFindExeSh(t *testing.T) {
	testPath := "/bin:/usr/bin"
	val, err := FindExe("sh", testPath)
	expect(t, err, val, "/bin/sh")
}

func TestFindExeShAbsolute(t *testing.T) {
	val, err := FindExe("/bin/sh", "")
	expect(t, err, val, "/bin/sh")
}

func TestFindExeInexistent(t *testing.T) {
	testPath := "/bin:/usr/bin"
	_, err := FindExe("inexistent", testPath)
	if err != ErrExeNotFound {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestFindShNoPath(t *testing.T) {
	_, err := FindExe("sh", "")
	if err != ErrExeNotFound {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestFindSomethingNotExec(t *testing.T) {
	_, err := FindExe("/proc/cmdline", "")
	if err != ErrExeNotFound {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestWhichNoEnv(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Unsetenv("PATH")
	defer os.Setenv("PATH", oldPath)

	_, err := Which("sh")
	if err != ErrEnvVarNotFound {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestWhichWithEnv(t *testing.T) {
	testPath := "/bin:/sbin:/usr/bin:/usr/sbin"
	oldPath := os.Getenv("PATH")
	os.Unsetenv("PATH")
	os.Setenv("PATH", testPath)
	defer os.Setenv("PATH", oldPath)

	val, err := Which("sh")
	expect(t, err, val, "/bin/sh")
}

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
