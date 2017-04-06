package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fromanirh/procwatch/procfind"
	"github.com/shirou/gopsutil/process"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

const confFile string = "procwatch.json"

type Config struct {
	IntervalSeconds int
	Argv            []string
}

func (conf *Config) ReadFile(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if len(content) > 0 {
		err = json.Unmarshal(content, conf)
		if err != nil {
			return err
		}
	}
	return nil
}

type Interval struct {
	Cmdline time.Duration
	Environ time.Duration
	Config  time.Duration
}

func (intv *Interval) Pick() time.Duration {
	val := time.Duration(1) * time.Second
	if intv.Cmdline < intv.Config {
		val = intv.Config
	} else {
		val = intv.Cmdline
	}
	if val < intv.Environ {
		return intv.Environ
	}
	return val
}

func NewInterval() Interval {
	val := time.Duration(1) * time.Second
	return Interval{Cmdline: val, Environ: val, Config: val}
}

func (intv *Interval) Fill(conf Config, envVar string) error {
	if conf.IntervalSeconds <= 0 {
		return errors.New(fmt.Sprintf("invalid interval: %d", conf.IntervalSeconds))
	}

	intv.Config = time.Duration(conf.IntervalSeconds) * time.Second

	val, err := strconv.Atoi(os.Args[1])
	intv.Cmdline = time.Duration(val) * time.Second
	if err != nil {
		return err
	}
	if envVar != "" {
		val, err = strconv.Atoi(envVar)
		if err != nil {
			return err
		}
		intv.Environ = time.Duration(val) * time.Second
	}

	return nil
}

func loadConf() *Config {
	conf := Config{IntervalSeconds: 1}
	confDirs := []string{
		os.Getenv("PROCWATCH_CONFIG_DIR"),
		"/etc",
		".",
	}
	for _, confDir := range confDirs {
		if confDir == "" {
			continue
		}
		confPath := path.Join(confDir, confFile)
		err := conf.ReadFile(confPath)
		if err == nil {
			log.Printf("Using configuration file %s", confPath)
		}
	}
	return &conf
}

func getInterval(conf Config) time.Duration {
	intv := NewInterval()
	err := intv.Fill(conf, os.Getenv("PROCWATCH_UPDATE_EVERY"))
	if err != nil {
		log.Printf("unknown duration: %s", err)
		return 0
	}
	return intv.Pick()
}

type ProcInfo struct {
	proc  *process.Process
	pid   int32
	argv0 string
}

func notifyCollectd(hostname string, interval time.Duration, pinfo *ProcInfo) error {
	cpu_times, err := pinfo.proc.CPUTimes()
	if err != nil {
		return err
	}
	fmt.Printf("%s/exec-%s-%d/cpu-user interval=%d %f:0:U\n", hostname, pinfo.argv0, pinfo.pid, int(interval.Seconds()), cpu_times.User)
	return nil
}

func collectForever(hostname string, argv []string, interval time.Duration) {
	c := time.Tick(interval)

	log.Printf("collection started")
	for _ = range c {
		// WARNING: we assume collection time is negligible
		pids, err := procfind.FindAll(argv)
		if err != nil {
			log.Printf("error collecting: %v - skipping cycle", err)
			continue
		}

		for _, pid := range pids {
			pinfo := ProcInfo{proc: nil, pid: int32(pid), argv0: filepath.Base(argv[0])}
			proc, err := process.NewProcess(pinfo.pid)
			if err != nil {
				log.Printf("cannot find process %v: %v", pinfo.pid, err)
				continue
			}
			pinfo.proc = proc
			notifyCollectd(hostname, interval, &pinfo)
		}
	}
	log.Printf("collection stopped")

}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s interval", os.Args[0])
	}
	log.Printf("procwatcher started")
	defer log.Printf("procwatcher stopped")

	conf := loadConf()

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("error getting the host name: %s", err)
		return
	}

	if len(conf.Argv) == 0 {
		log.Printf("missing process to track")
		return
	} else {
		log.Printf("configuration: %#v", conf)
	}

	updateInterval := getInterval(*conf)
	if updateInterval <= 0 {
		log.Printf("bad interval: %v", updateInterval)
		return
	} else {
		log.Printf("updating process stats every %v", updateInterval)
	}

	collectForever(hostname, conf.Argv, updateInterval)
}
