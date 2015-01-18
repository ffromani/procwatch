package procfind

import "errors"

var (
    ErrPidNotFound = errors.New("pid not found")
    ErrExeNotFound = errors.New("executable not found")
    ErrEnvVarNotFound = errors.New("environment variable missing")
)

