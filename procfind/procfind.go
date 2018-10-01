package procfind

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Pid int32

var (
	ErrPidNotFound    = errors.New("pid not found")
	ErrExeNotFound    = errors.New("executable not found")
	ErrEnvVarNotFound = errors.New("environment variable missing")
)

func Path() (string, error) {
	for _, envvar := range os.Environ() {
		if strings.HasPrefix(envvar, "PATH") {
			items := strings.SplitN(envvar, "=", 2)
			return items[1], nil
		}
	}
	return "", ErrEnvVarNotFound
}

func FindExe(exename, pathlist string) (string, error) {
	var exepath string
	var exeinfo os.FileInfo
	err := ErrExeNotFound

	if strings.HasPrefix(exename, string(os.PathSeparator)) {
		exepath = exename
		exeinfo, err = os.Lstat(exepath)
	} else {
		for _, exedir := range filepath.SplitList(pathlist) {
			exepath = filepath.Join(exedir, exename)
			exeinfo, err = os.Lstat(exepath)
			if err == nil {
				break
			}
		}
	}

	if err != nil || !isExecutable(exeinfo) {
		return "", ErrExeNotFound
	}
	return exepath, nil
}

func Which(exename string) (string, error) {
	pathlist, err := Path()
	if err != nil {
		return "", err
	}
	return FindExe(exename, pathlist)
}

func isExecutable(info os.FileInfo) bool {
	return bool(int(info.Mode().Perm()&0111) != 0)
}

func MatchArgv(argv, model []string) (bool, error) {
	ref := model
	oth := argv
	if len(argv) < len(model) {
		ref = argv
		oth = model
	}
	for idx, elem := range ref {
		matched, err := filepath.Match(elem, oth[idx])
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}
