package procnotify

import (
	"github.com/fromanirh/procwatch/podfind"
	"github.com/fromanirh/procwatch/procfind"
	"github.com/shirou/gopsutil/process"

	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Name       string   `json:"name"`
	Argv       []string `json:"argv"`
	StableName bool     `json:"stable_name"`
}

type TargetConfigs struct {
	Configs []Config
}

type Target struct {
	Config
	Pids []procfind.Pid
}

func (t *Target) AddPid(p procfind.Pid) {
	log.Printf("new PID: %v -> %v", t.Name, p)
	t.Pids = append(t.Pids, p)
}

type Proc struct {
	t *Target
	p *process.Process
}

type Notifier struct {
	Debug    bool
	targets  []*Target
	procs    map[int32]Proc
	pr       *podfind.PodResolver
	sinkPath string
}

func (notif *Notifier) MatchArgv(argv []string) (procfind.Entry, bool) {
	for _, target := range notif.targets {
		match, err := procfind.MatchArgv(argv, target.Argv)
		if err != nil {
			break
		} else if match {
			return target, true
		}

	}
	return &Target{}, false
}

func NewNotifier(targets []Config, pr *podfind.PodResolver, sinkPath string) *Notifier {
	notif := Notifier{
		pr:       pr,
		sinkPath: sinkPath,
	}
	for _, target := range targets {
		t := &Target{
			Config: target,
		}
		if target.Name == "" {
			t.Name = filepath.Base(target.Argv[0])
		}
		notif.targets = append(notif.targets, t)
	}
	return &notif
}

func (notif *Notifier) Dump(w io.Writer) error {
	for _, target := range notif.targets {
		fmt.Fprintf(w, "- %s [%s] stablename=%v\n",
			target.Name, strings.Join(target.Argv, " "), target.StableName)
	}
	return nil
}

func (notif *Notifier) Scan() error {
	notif.procs = make(map[int32]Proc)
	found, err := procfind.ScanEntries(notif)
	if err != nil {
		return err
	}
	log.Printf("Scanned /proc and found %d pid(s)", found)
	for _, target := range notif.targets {
		for _, pid := range target.Pids {
			proc, err := process.NewProcess(int32(pid))
			if err != nil {
				log.Printf("cannot find process %v: %v", pid, err)
				continue
			}
			notif.procs[int32(pid)] = Proc{p: proc, t: target}
		}
	}
	return nil
}

func (notif *Notifier) HasTargets() bool {
	if len(notif.procs) == 0 {
		return false
	}
	return true
}

func (notif *Notifier) IsCurrent() bool {
	for pid, proc := range notif.procs {
		if !procfind.Match(proc.t.Argv, procfind.Pid(pid)) {
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

func (notif *Notifier) collectd(proc Proc, hostname string, interval int) error {
	var err error
	var ident string

	var sink io.Writer = os.Stdout
	if notif.sinkPath != "" {
		sock, err := net.Dial("unix", notif.sinkPath)
		if err != nil {
			return err
		}
		defer sock.Close()
		sink = sock
	}

	if !proc.t.StableName {
		if notif.pr != nil {
			podName, err := notif.pr.FindPodByPID(proc.p.Pid)
			if err == nil {
				ident = fmt.Sprintf("PUTVAL %s/exec-%s-%s", hostname, proc.t.Name, podName)
			}
		}
		if ident == "" {
			ident = fmt.Sprintf("PUTVAL %s/exec-%s-%d", hostname, proc.t.Name, proc.p.Pid)
		}
	} else {
		ident = fmt.Sprintf("PUTVAL %s/exec-%s", hostname, proc.t.Name)
		fmt.Fprintf(sink, "%s/objects interval=%d N:%d\n", ident, interval, proc.p.Pid)
	}

	cpu_perc, err := proc.p.Percent(0)
	if err != nil {
		return err
	}
	fmt.Fprintf(sink, "%s/cpu-perc interval=%d N:%d\n", ident, interval, int(round(cpu_perc, 0.5, 0)))
	fmt.Fprintf(sink, "%s/percent-cpu interval=%d N:%d\n", ident, interval, int(round(cpu_perc, 0.5, 0)))

	cpu_times, err := proc.p.Times()
	if err != nil {
		return err
	}
	fmt.Fprintf(sink, "%s/cpu-user interval=%d N:%d\n", ident, interval, int(round(cpu_times.User, 0.5, 0)))
	fmt.Fprintf(sink, "%s/cpu-system interval=%d N:%d\n", ident, interval, int(round(cpu_times.System, 0.5, 0)))

	mem_info, err := proc.p.MemoryInfo()
	if err != nil {
		return err
	}
	fmt.Fprintf(sink, "%s/memory-virtual interval=%d N:%d\n", ident, interval, mem_info.VMS/1024)
	fmt.Fprintf(sink, "%s/memory-resident interval=%d N:%d\n", ident, interval, mem_info.RSS/1024)

	return nil
}

func (notif *Notifier) Update(hostname string, interval int) {
	var err error
	for _, proc := range notif.procs {
		err = notif.collectd(proc, hostname, interval)
		if err != nil {
			log.Printf("Update failed: %s", err)
		}
	}

	if notif.Debug {
		log.Printf("updated")
	}
}

func (notif *Notifier) Once(hostname string) {
	var err error

	err = notif.Scan()
	if err != nil {
		log.Printf("error during the collection setup: %v", err)
	}

	// WARNING: we assume collection time is negligible
	if notif.pr != nil {
		err = notif.pr.Update()
		if err != nil {
			log.Printf("error during the kube update: %v", err)
		}
	}

	notif.Update(hostname, 0)
}

func (notif *Notifier) Loop(hostname string, interval time.Duration, autoTrack bool) {
	c := time.Tick(interval)

	log.Printf("collection started")
	defer log.Printf("collection stopped")

	var err error

	err = notif.Scan()
	if err != nil {
		log.Printf("error during the collection setup: %v", err)
	}

	for _ = range c {
		// WARNING: we assume collection time is negligible
		if notif.pr != nil {
			err = notif.pr.Update()
			if err != nil {
				log.Printf("error during the kube update: %v", err)
			}
		}

		if !notif.HasTargets() {
			log.Printf("nothing to do...")
			continue
		}

		if !notif.IsCurrent() {
			if !autoTrack {
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

		notif.Update(hostname, int(interval.Seconds()))
	}
}
