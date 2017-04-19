package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fromanirh/procwatch/procnotify"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"
)

const confFile string = "procwatch.json"

func ReadFile(conf *procnotify.Config, path string) error {
	log.Printf("trying configuration: %s", path)

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

	log.Printf("Read from file: %s", path)
	log.Printf("Updated configuration: %#v", conf)
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

func (intv *Interval) Fill(conf procnotify.Config) error {
	if conf.Interval <= 0 {
		return errors.New(fmt.Sprintf("invalid interval: %d", conf.Interval))
	}

	intv.Config = time.Duration(conf.Interval) * time.Second

	if len(os.Args) >= 3 {
		val, err := strconv.Atoi(os.Args[2])
		if err != nil {
			return err
		}
		intv.Cmdline = time.Duration(val) * time.Second
	}

	envVar := os.Getenv("COLLECTD_INTERVAL")
	if envVar != "" {
		val, err := strconv.ParseFloat(envVar, 64)
		if err != nil {
			return err
		}
		intv.Environ = time.Duration(int(val)) * time.Second
	}

	return nil
}

func loadConf() procnotify.Config {
	conf := procnotify.Config{Interval: 2, AutoTrack: true, ReportPid: true}
	if len(os.Args) >= 2 {
		err := ReadFile(&conf, os.Args[1])
		if err == nil {
			return conf
		}
	}
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
		log.Printf("trying configuration: %s", confPath)
		err := ReadFile(&conf, confPath)
		if err == nil {
			break
		}
	}
	return conf
}

func selectInterval(conf *procnotify.Config) time.Duration {
	intv := NewInterval()
	err := intv.Fill(*conf)
	if err != nil {
		log.Printf("unknown duration: %s", err)
		return 0
	}
	conf.Interval = intv.Pick()
	return conf.Interval
}

func needHelp() bool {
	if len(os.Args) > 3 {
		return true
	}
	if len(os.Args) >= 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			return true
		}
	}
	return false
}

func main() {
	if needHelp() {
		fmt.Fprintf(os.Stderr, "usage: %s [/path/to/procwatch.json [interval_seconds]]\n", os.Args[0])
		return
	}

	log.Printf("procwatcher started")
	defer log.Printf("procwatcher stopped")

	var err error
	conf := loadConf()

	hostname := os.Getenv("COLLECTD_HOSTNAME")
	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			log.Printf("error getting the host name: %s", err)
			return
		}
	}
	conf.Hostname = hostname

	if len(conf.Argv) == 0 {
		log.Printf("missing process to track")
		return
	} else {
		log.Printf("configuration: %#v", conf)
	}

	selectInterval(&conf)
	if conf.Interval <= 0 {
		log.Printf("bad interval: %v", conf.Interval)
		return
	} else {
		log.Printf("updating process stats every %v", conf.Interval)
	}

	notifier := procnotify.NewNotifier(conf)
	notifier.Loop()
}
