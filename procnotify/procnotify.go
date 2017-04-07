package procnotify

import (
	"fmt"
	"github.com/fromanirh/procwatch/procfind"
	"github.com/shirou/gopsutil/process"
	"log"
	"math"
	"path/filepath"
	"time"
)

type Notifier struct {
	argv      []string
	autotrack bool
	hostname  string
	interval  time.Duration
	name      string
	procs     map[int32]*process.Process
}

func NewNotifier(name string, argv []string, autotrack bool, hostname string, interval time.Duration) *Notifier {
	if name == "" {
		name = filepath.Base(argv[0])
	}
	return &Notifier{
		name:      name,
		argv:      argv,
		autotrack: autotrack,
		hostname:  hostname,
		interval:  interval,
	}
}

func (notif *Notifier) Scan() error {
	notif.procs = make(map[int32]*process.Process)
	pids, err := procfind.FindAll(notif.argv)
	if err != nil {
		return err
	}
	log.Printf("Scanned /proc and found pid(s) %#v", pids)
	for _, pid := range pids {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			log.Printf("cannot find process %v: %v", pid, err)
			continue
		}
		notif.procs[int32(pid)] = proc
	}
	return nil
}

func (notif *Notifier) IsCurrent() bool {
	if len(notif.procs) == 0 {
		return false
	}
	for pid := range notif.procs {
		if !procfind.Match(notif.argv, procfind.Pid(pid)) {
			return false
		}
	}
	return true
}

func round(val float64, roundOn float64, places int) float64 {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	return round / pow
}

func (notif *Notifier) collectd(proc *process.Process) error {
	ident := fmt.Sprintf("PUTVAL %s/exec-%s-%d", notif.hostname, notif.name, proc.Pid)
	interval := int(notif.interval.Seconds())

	cpu_perc, err := proc.Percent(0)
	if err != nil {
		return err
	}
	fmt.Printf("%s/cpu-perc interval=%d N:%d\n", ident, interval, int(round(cpu_perc, 0.5, 0)))
	fmt.Printf("%s/percent-cpu interval=%d N:%d\n", ident, interval, int(round(cpu_perc, 0.5, 0)))

	cpu_times, err := proc.Times()
	if err != nil {
		return err
	}
	fmt.Printf("%s/cpu-user interval=%d N:%d\n", ident, interval, int(round(cpu_times.User, 0.5, 0)))
	fmt.Printf("%s/cpu-system interval=%d N:%d\n", ident, interval, int(round(cpu_times.System, 0.5, 0)))

	mem_info, err := proc.MemoryInfo()
	if err != nil {
		return err
	}
	fmt.Printf("%s/memory-virtual interval=%d N:%d\n", ident, interval, mem_info.VMS/1024)
	fmt.Printf("%s/memory-resident interval=%d N:%d\n", ident, interval, mem_info.RSS/1024)

	return nil
}

func (notif *Notifier) Update() {
	for _, proc := range notif.procs {
		notif.collectd(proc)
	}
}

func (notif *Notifier) Loop() {
	c := time.Tick(notif.interval)

	log.Printf("collection started")
	defer log.Printf("collection stopped")

	err := notif.Scan()
	if err != nil {
		log.Printf("error during the collection setup: %v", err)
	}

	for _ = range c {
		// WARNING: we assume collection time is negligible
		if !notif.IsCurrent() {
			if !notif.autotrack {
				log.Printf("stale pid(s) -- aborting!")
				break
			} else {
				log.Printf("stale pid(s) -- rescanning!")
				err = notif.Scan()
				if err != nil {
					log.Printf("error collecting: %v - skipping cycle", err)
					continue
				}
			}
		}

		notif.Update()
	}
}
