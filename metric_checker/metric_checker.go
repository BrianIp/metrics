//Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/measure/metrics/check"
)

func main() {
	var hostport, configfile string
	var step int
	flag.StringVar(&hostport, "address", "localhost:12345",
		"address to listen on for metrics json")
	flag.StringVar(&configfile, "cnf", "",
		"config file to grab thresholds from")
	flag.IntVar(&step, "step", 0, "seconds for a cycle of metric checks")
	flag.Parse()

	stepTime := time.Second * time.Duration(step)

	c, err := check.New(hostport, configfile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.CheckAll(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if step > 0 {
		ticker := time.NewTicker(stepTime)
		for _ = range ticker.C {
			err := c.CheckAll(os.Stdout)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}
	}
}
