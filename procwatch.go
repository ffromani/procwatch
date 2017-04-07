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

type Config struct {
	Argv      []string
	AutoTrack bool
	Interval  int
	Name      string
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

func (intv *Interval) Fill(conf Config) error {
	if conf.Interval <= 0 {
		return errors.New(fmt.Sprintf("invalid interval: %d", conf.Interval))
	}

	intv.Config = time.Duration(conf.Interval) * time.Second

	if len(os.Args) >= 2 {
		val, err := strconv.Atoi(os.Args[1])
		if err != nil {
			return err
		}
		intv.Cmdline = time.Duration(val) * time.Second
	}

	envVar := os.Getenv("PROCWATCH_UPDATE_EVERY")
	if envVar != "" {
		val, err := strconv.Atoi(envVar)
		if err != nil {
			return err
		}
		intv.Environ = time.Duration(val) * time.Second
	}

	return nil
}

func loadConf() *Config {
	conf := Config{Interval: 2, AutoTrack: true}
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
	err := intv.Fill(conf)
	if err != nil {
		log.Printf("unknown duration: %s", err)
		return 0
	}
	return intv.Pick()
}

func main() {
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

	interval := getInterval(*conf)
	if interval <= 0 {
		log.Printf("bad interval: %v", interval)
		return
	} else {
		log.Printf("updating process stats every %v", interval)
	}

	notifier := procnotify.NewNotifier(conf.Name, conf.Argv, conf.AutoTrack, hostname, interval)
	notifier.Loop()
}
