package procfind

import (
    "bytes"
    "io/ioutil"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)


func readProcCmdline(pathname string) []string {
    argv := make([]string, 1)
    content, err := ioutil.ReadFile(pathname)
    if err == nil {
        for _, chunk := range bytes.Split(content, []byte{0}) {
            argv = append(argv, string(chunk))
        }
    }
    return argv
}


func findInProcFS(exename, scriptname string) (Pid, error) {
    entries, err := filepath.Glob("/proc/*/cmdline")
    if err != nil {
        return 0, err
    }
    for _, entry := range entries {
        argv := readProcCmdline(entry)
        if argv == nil {
            return 0, ErrExeNotFound
        }

        path, err := PathList()
        if err != nil {
            return 0, err
        }

        exepath, err := FindExe(exename, path)
        if err != nil {
            return 0, err
        }

        if argv[0] == exepath && (scriptname == "" || argv[1] == scriptname) {
            items := strings.Split(entry, string(os.PathSeparator))
            pid, err := strconv.Atoi(items[1])
            if err != nil {
                return 0, err
            }

            return Pid(pid), nil
        }
    }
    return 0, ErrPidNotFound
}


func Argv(pid Pid) []string {
    return readProcCmdline(filepath.Join("/proc", strconv.Itoa(int(pid)), "cmdline"))
}

func PidOf(exename string) (Pid, error) {
    return findInProcFS(exename, "")
}

func PidOfScript(interpname string, scriptname string) (Pid, error) {
    return findInProcFS(interpname, scriptname)
}

