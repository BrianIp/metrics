//Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/measure/metrics/check"
)

func main() {
	var hostport, configfile string
	flag.StringVar(&hostport, "address", "localhost:12345",
		"address to listen on for metrics json")
	flag.StringVar(&configfile, "cnf", "",
		"config file to grab thresholds from")
	flag.Parse()
	c, err := check.New(hostport, configfile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.CheckAll(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}
