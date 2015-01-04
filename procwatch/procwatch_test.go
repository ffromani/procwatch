/* procwatch keeps track of statistics about running processes */
package procwatch

import (
    "testing"
    "time"
)

func newWatcher(t *testing.T) *Watcher {
    w, err := NewWatcher(100)
    if err != nil {
        t.Fail()
    }
    return w
}


func TestEmptyTracking(t *testing.T) {
    w := newWatcher(t)

    if w.IsTracked(1) {
        t.Fail()
    }
}


func TestEmptyStop(t *testing.T) {
    w := newWatcher(t)

    err := w.Stop()
    if err != nil {
        t.Fail()
    }
}

func TestTrackedStop(t *testing.T) {
    var err error = nil
    w := newWatcher(t)

    err = w.Track(1)
    if err != nil {
        t.Error("track failed")
    }

    err = w.Stop()
    if err != nil {
        t.Error("stop failed")
    }
}

func TestEmptyIsNotTracked(t *testing.T) {
    w := newWatcher(t)

    if w.IsTracked(1) {
        t.Error("who is tracking who?")
    }
}


func TestTrackIsTracked(t *testing.T) {
    var err error = nil
    w := newWatcher(t)

    err = w.Track(1)
    if err != nil {
        t.Error("track failed")
    }

    if !w.IsTracked(1) {
        t.Error("proc seems not tracked")
    }
}

func TestTrackTwice(t *testing.T) {
    var err error = nil
    w := newWatcher(t)

    err = w.Track(1)
    if err != nil {
        t.Fail()
    }

    err = w.Track(1)
    if err != ErrProcAlreadyAdded {
        t.Fail()
    }
}

func TestUntrackNotAdded(t *testing.T) {
    w := newWatcher(t)

    stats, err := w.Untrack(1)

    if err != ErrProcMissing {
        t.Fail()
    }
    if stats != nil {
        t.Fail()
    }
}

func checkStats(t *testing.T, stats []*WatchPoint, err error) {
    if err != nil {
        t.Errorf("unexpected error: %s", err.Error())
    }
    if stats == nil {
        t.Error("nil stats")
    }
}


func TestPollOnce(t *testing.T) {
    var err error = nil
    w := newWatcher(t)

    err = w.Track(1)
    if err != nil {
        t.Fail()
    }

    stats, err2 := w.Poll(1)

    checkStats(t, stats, err2)
}

func TestPollUnexistingProc(t *testing.T) {
    w := newWatcher(t)

    stats, err := w.Poll(0)
    if err != ErrProcMissing {
        t.Error("unexpected error: %s", err.Error())
    }
    if stats != nil {
        t.Error("unexpected stats")
    }
}

func TestTrackUnexistingProc(t *testing.T) {
    w := newWatcher(t)

    err := w.Track(0)
    if err != ErrProcMissing {
        t.Fail()
    }
}

func TestTrack(t *testing.T) {
    if testing.Short() {
        t.Skip()
    }

    var err error = nil
    w := newWatcher(t)

    err = w.Track(1)
    if err != nil {
        t.Fail()
    }

    time.Sleep(500 * time.Millisecond)

    stats, err2 := w.Untrack(1)

    checkStats(t, stats, err2)

    if len(stats) <= 1 {
        t.Error("not enough stats")
    }
}

