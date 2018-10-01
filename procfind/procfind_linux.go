package procfind

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Match(cmdline []string, pid Pid) bool {
	entry := fmt.Sprintf("/proc/%d/cmdline", int(pid))
	argv := readProcCmdline(entry)

	if argv == nil || len(argv) == 0 {
		return false
	}

	matched, err := MatchArgv(argv, cmdline)
	if err != nil || !matched {
		return false
	}
	return true
}

func MatchAll(cmdline []string, pids []Pid) bool {
	for _, pid := range pids {
		if !Match(cmdline, pid) {
			return false
		}
	}
	return true
}

func Argv(pid Pid) []string {
	return readProcCmdline(filepath.Join("/proc", strconv.Itoa(int(pid)), "cmdline"))
}

func Find(argv []string) (Pid, error) {
	pids, err := findInProcFs(argv, true)
	if err != nil {
		return 0, err
	}
	return pids[0], nil
}

func FindAll(argv []string) ([]Pid, error) {
	return findInProcFs(argv, false)
}

type Entry interface {
	AddPid(p Pid)
}

type EntryMatcher interface {
	MatchArgv(argv []string) (Entry, bool)
}

func ScanEntries(em EntryMatcher) (int, error) {
	procEntries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}

	var scanned int
	for _, procEntry := range procEntries {
		argv := readProcCmdline(procEntry)
		if argv == nil || len(argv) == 0 {
			continue
		}

		entry, ok := em.MatchArgv(argv)
		if !ok {
			continue
		}

		items := strings.Split(procEntry, string(os.PathSeparator))
		// "", "proc", "$PID", "cmdline"
		pid, err := strconv.Atoi(items[2])
		if err != nil {
			return 0, err
		}

		entry.AddPid(Pid(pid))
		scanned += 1
	}

	return scanned, nil
}

func PidOf(exename string) ([]Pid, error) {
	exepath, err := Which(exename)
	if err != nil {
		return make([]Pid, 0), err
	}
	return findInProcFs([]string{exepath}, false)
}

func readProcCmdline(pathname string) []string {
	argv := make([]string, 0)
	content, err := ioutil.ReadFile(pathname)
	if err == nil && len(content) > 0 {
		for _, chunk := range bytes.Split(content, []byte{0}) {
			arg := string(chunk)
			if len(arg) > 0 {
				argv = append(argv, arg)
			}
		}
	}
	return argv
}

func findPidByArgv(cmdline []string, entries []string, firstOnly bool) ([]Pid, error) {
	var err error = nil
	pids := make([]Pid, 0)
	for _, entry := range entries {
		argv := readProcCmdline(entry)
		if argv == nil {
			return pids, ErrExeNotFound
		}
		if len(argv) == 0 {
			continue
		}

		matched, err := MatchArgv(argv, cmdline)
		if err != nil {
			return pids, err
		}
		if matched {
			items := strings.Split(entry, string(os.PathSeparator))
			// "", "proc", "$PID", "cmdline"
			pid, err := strconv.Atoi(items[2])
			if err != nil {
				return pids, err
			}

			pids = append(pids, Pid(pid))
			if firstOnly {
				break
			}
		}
	}

	if len(pids) == 0 {
		err = ErrPidNotFound
	}
	return pids, err
}

func findInProcFs(argv []string, firstOnly bool) ([]Pid, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return make([]Pid, 0), err
	}
	return findPidByArgv(argv, entries, firstOnly)
}
