package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/fromanirh/procwatch/podfind"
	"github.com/fromanirh/procwatch/procnotify"
	flag "github.com/spf13/pflag"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const confFile string = "procwatch.json"

type Config struct {
	Targets     []procnotify.Config `json:"targets"`
	Interval    string              `json:"interval"`
	Hostname    string              `json:"hostname"`
	CRIEndPoint string              `json:"criendpoint"`
	AutoTrack   bool                `json:"autotrack"`
}

func (c Config) CountTargets() int {
	return len(c.Targets)
}

func readFile(conf *Config, path string) error {
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
	return nil
}

func findInterval(conf Config, args []string) (time.Duration, error) {
	if len(args) >= 2 {
		ival, err := strconv.Atoi(args[1])
		if err != nil {
			return 0, err
		}
		return time.Duration(ival) * time.Second, nil
	}

	envVar := os.Getenv("COLLECTD_INTERVAL")
	if envVar != "" {
		fval, err := strconv.ParseFloat(envVar, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(int(fval)) * time.Second, nil
	}

	if conf.Interval == "" {
		return 0, errors.New(fmt.Sprintf("invalid interval: %d", conf.Interval))
	}

	dval, err := time.ParseDuration(conf.Interval)
	if err != nil {
		return 0, err
	}
	return dval, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s /path/to/procwatch.json [interval_seconds]\n", os.Args[0])
		flag.PrintDefaults()
	}
	requirePodResolution := flag.BoolP("require-pod", "R", false, "fail if pod resolution is not enabled")
	debugMode := flag.BoolP("debug", "D", false, "enable debug mode")
	sinkPath := flag.StringP("unixsock", "U", "", "send output to <unixsock> not to stdout")
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		return
	}

	log.Printf("procwatcher started")
	defer log.Printf("procwatcher stopped")

	conf := Config{Interval: "5s"}
	err := readFile(&conf, args[0])
	if err != nil {
		log.Fatalf("error reading the configuration on '%s': %s", args[0], err)
	}

	conf.Hostname = os.Getenv("COLLECTD_HOSTNAME")
	if conf.Hostname == "" {
		conf.Hostname, err = os.Hostname()
		if err != nil {
			log.Fatalf("error getting the host name: %s", err)
		}
	}

	interval, err := findInterval(conf, args)
	if err != nil {
		log.Fatalf("error getting the polling interval: %s", err)
	} else {
		log.Printf("polling interval: %v", interval)
	}

	dryRun := os.Getenv("PROCWATCH_DRYRUN")
	if dryRun != "" {
		log.Printf("%s", spew.Sdump(conf))
		return
	}

	if conf.CountTargets() == 0 {
		log.Fatalf("missing process(es) to track")
	}

	var pr *podfind.PodResolver
	if conf.CRIEndPoint != "" {
		log.Printf("enabled POD ID resolution")
		pr, err = podfind.NewPodResolver(conf.CRIEndPoint, 10*time.Second)
		if err != nil {
			log.Printf("unable to set up pod resolution: %s", err)
			pr = nil
		} else {
			pr.Debug = *debugMode
		}
	}

	if pr == nil && *requirePodResolution {
		log.Fatalf("pod resolution required but not enabled!")
	}

	var sink io.Writer = os.Stdout
	if *sinkPath != "" {
		sock, err := net.Dial("unix", *sinkPath)
		if err != nil {
			log.Fatalf("cannot open output sink '%s': %s", *sinkPath, err)
		}
		defer sock.Close()
		sink = sock
	}

	notifier := procnotify.NewNotifier(conf.Targets, pr, sink)
	log.Printf("Tracking:\n")
	notifier.Dump(os.Stderr)

	notifier.Loop(conf.Hostname, interval, conf.AutoTrack)
}
