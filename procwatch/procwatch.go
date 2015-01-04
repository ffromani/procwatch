// procwatch keeps track of statistics about running processes
//
// (C) 2014-2015 Francesco Romani (fromani on gmail)
// License: MIT

package procwatch


import (
    "errors"
    "time"

    "github.com/shirou/gopsutil/process"
)


var (
    ErrProcAlreadyAdded = errors.New("Pid already added")
    ErrProcMissing = errors.New("Pid not added")
    ErrProcNotFound = errors.New("Pid doesn't exist")
)


type WatchCpu struct {
    User uint
    Sys uint
    Total uint
}

type WatchPoint struct {
    Timestamp time.Time
    NumThreads int32
    MemPerc float32
    CpuTotal WatchCpu
}

type watchProc struct {
    proc *process.Process
    cmd chan bool
    res chan []*WatchPoint
}

type Watcher struct {
    procs map[int32]*watchProc
    ticker *time.Ticker;
}


func (wp *watchProc) sample() *WatchPoint {
//    numthreads, err1 := wp.proc.NumThreads()
//    memperc, err2 := wp.proc.MemoryPercent()
    return &WatchPoint{
        Timestamp: time.Now(),
    }
}

func (wp *watchProc) run() {
    data := make([]*WatchPoint, 16) // FIXME

    for sync := range wp.cmd {
        item := wp.sample()
        data = append(data, item)
        if sync {
            wp.res<-[]*WatchPoint{item}
        }
    }

    wp.res<-data
    close(wp.res)
}

/*
Tells if a process given by its pid
is currently being watched or not.
*/
func (w *Watcher) IsTracked(pid int32) bool {
    _, present := w.procs[pid]
    return present
}

/*
Tracks a process by its pid and collect
its samples automatically.
*/
func (w *Watcher) Track(pid int32) error {
    if w.IsTracked(pid) {
        return ErrProcAlreadyAdded
    }

    proc, err := process.NewProcess(pid)
    if err != nil {
        return ErrProcMissing
    }

    wp := &watchProc{
        proc: proc,
        cmd: make(chan bool),
        res: make(chan []*WatchPoint)}
    w.procs[pid] = wp

    go wp.run()

    return nil
}

/*
Untracks the given process by its pid.
Returns a sequence of all the collected WatchPoints.
*/
func (w *Watcher) Untrack(pid int32) ([]*WatchPoint, error) {
    wp, present := w.procs[pid]
    if !present {
        return nil, ErrProcMissing // TODO: add pid
    }

    close(wp.cmd)
    return <-wp.res, nil
}

func (w *Watcher) run() {
    for _ = range w.ticker.C {
        for _, wp := range w.procs { // FIXME
            wp.cmd<-false
        }
    }
}

/*
Request synchronous sampling for a given process
*/
func (w *Watcher) Poll(pid int32) ([]*WatchPoint, error) {
    wp, present := w.procs[pid]
    if !present {
        return nil, ErrProcMissing
    }

    wp.cmd<-true
    return <-wp.res, nil
}

/*
Stops the watcher.
Implicitely and automatically untracks all processes.
*/
func (w *Watcher) Stop() error {
    w.ticker.Stop()
    for pid := range w.procs {
        w.Untrack(pid)
        delete(w.procs, pid)
    }
    return nil
}

/*
Creates a new watcher, which will sample each
tracked process each <interval> milliseconds.
*/
func NewWatcher(interval time.Duration) (*Watcher, error) {
    w := &Watcher{
        procs: make(map[int32]*watchProc),
        ticker: time.NewTicker(time.Millisecond * interval)}
    go w.run()
    return w, nil
}

