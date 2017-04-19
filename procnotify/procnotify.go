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
	conf  Config
	procs map[int32]*process.Process
}

type Config struct {
	Argv      []string
	AutoTrack bool
	ReportPid bool
	Interval  time.Duration
	Name      string
	Hostname  string
}

func NewNotifier(conf Config) *Notifier {
	cfg := conf
	if cfg.Name == "" {
		cfg.Name = filepath.Base(conf.Argv[0])
	}
	return &Notifier{conf: cfg}
}

func (notif *Notifier) Scan() error {
	notif.procs = make(map[int32]*process.Process)
	pids, err := procfind.FindAll(notif.conf.Argv)
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
		if !procfind.Match(notif.conf.Argv, procfind.Pid(pid)) {
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
	interval := int(notif.conf.Interval.Seconds())

	var ident string
	if !notif.conf.ReportPid {
		ident = fmt.Sprintf("PUTVAL %s/exec-%s-%d", notif.conf.Hostname, notif.conf.Name, proc.Pid)
	} else {
		ident = fmt.Sprintf("PUTVAL %s/exec-%s", notif.conf.Hostname, notif.conf.Name)
		fmt.Printf("%s/objects interval=%d N:%d\n", ident, interval, proc.Pid)
	}

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
	c := time.Tick(notif.conf.Interval)

	log.Printf("collection started")
	defer log.Printf("collection stopped")

	err := notif.Scan()
	if err != nil {
		log.Printf("error during the collection setup: %v", err)
	}

	for _ = range c {
		// WARNING: we assume collection time is negligible
		if !notif.IsCurrent() {
			if !notif.conf.AutoTrack {
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
