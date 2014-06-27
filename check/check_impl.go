//Copyright (c) 2014 Square, Inc
package check

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/measure/metrics"
	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goconf/conf" // used for parsing config files
)

type checker struct {
	address string
	Metrics  map[string]metric
	Warnings map[string]metricResults
	c        *conf.ConfigFile
	Logger   *log.Logger
	pkg      *types.Package
	scope    *types.Scope
	m        *metrics.MetricContext
}

//stores the checks gathered from the config file
type metricThresholds struct {
	metricblob string
	checks     map[string]string
}

//stores the results of checks for a section of the config file
type metricResults struct {
	Message string
	Checks  map[string]bool // maps check name to result
}

//struct holding metric's value
type metric struct {
	Type  string
	Name  string
	Value float64
	Rate  float64
}

// Creates new checker based on configFile
func New(configFile string, metrics *metric[]) (Checker, error) {
	cnf, err := conf.ReadConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	hc := &checker{
		hostport: hostport, //hostport to listen on for metrics json
		Metrics:  make(map[string]metric),
		Warnings: make(map[string]metricResults),
		c:        c,
		Logger:   log.New(os.Stderr, "LOG: ", log.Lshortfile),
	}
	hc.setupConstants()
	return hc, nil
}



//getMetrics gathers metric values.
// If there a metric context to collect metrics,
// use that instead of listening for incoming json packages
func (hc *checker) getMetrics() error {
	if hc.m != nil {
		err := hc.getMetricsFromContext()
		return err
	}
	//get metrics from metrics collector
	resp, err := http.Get("http://" + hc.hostport +
		"/api/v1/metrics.json/Counters|Gauges|StatTimers?allowNaN=false")
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	//unmarshal metrics
	var metrics []metric
	err = d.Decode(&metrics)
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	//store metrics in map, so they can be found easily by name
	for _, m := range metrics {
		hc.Metrics[m.Name] = m
	}
	return nil
}

//Checks all sections metrics.
//iterates through checks in config file and checks against collected metrics
func (hc *checker) CheckMetrics() error {
	err := hc.getMetrics()
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	//iterate through all sections of tests
	for _, sectionName := range hc.c.GetSections() {
		if sectionName == "default" ||
			sectionName == "nagios" ||
			sectionName == "constants" {
			continue
		}
		m := getConfigChecks(hc.c, sectionName)
		hc.Warnings[sectionName] = hc.checkMetric(m)
	}
	return nil
}

//Check single section of metrics tests
func (hc *checker) checkMetric(m metricThresholds) metricResults {
	res := &metricResults{}
	res.Checks = make(map[string]bool)
	for name, check := range m.checks {
		checkVal, err := hc.replaceNames(check)
		if err != nil {
			hc.Logger.Println(err)
		}
		resultType, result, err := types.Eval(checkVal, hc.pkg, hc.scope)
		//error evaluating expression, don't store result
		if err != nil {
			hc.Logger.Println(err)
			continue
		}
		//check that expression evaluated to bool
		if !types.Identical(resultType, types.Typ[types.UntypedBool]) &&
			!types.Identical(resultType, types.Typ[types.Bool]) {
			hc.Logger.Println("Check: " + name + ": " +
				check + " does not evaluate to bool")
			continue
		}
		res.Checks[name], _ = strconv.ParseBool(result.String())
	}
	return *res
}

//finds and replaces names of other metrics inside expression
func (hc *checker) replaceNames(expr string) (string, error) {
	words := strings.Split(expr, " ")
	for _, word := range words {
		//metric names must contain a '.'
		// for instance, metric.Value and metric.Rate are valid metric names
		if strings.Contains(word, ".") {
			parts := strings.Split(word, ".")
			metricName := strings.Join(parts[:len(parts)-1], ".")
			m, ok := hc.Metrics[metricName]
			//if the metric was not collected, skip ahead
			if !ok {
				continue
			}
			if parts[len(parts)-1] == "Value" {
				expr = strings.Replace(expr, word,
					strconv.FormatFloat(m.Value, 'f', 5, 64), -1)
			} else if parts[len(parts)-1] == "Rate" {
				expr = strings.Replace(expr, word,
					strconv.FormatFloat(m.Rate, 'f', 5, 64), -1)
			}
		}
	}
	return expr, nil
}

//Reads the thresholds and messages from the config file
func getConfigChecks(c *conf.ConfigFile, test string) metricThresholds {
	m := &metricThresholds{}
	m.checks = make(map[string]string)
	checks, _ := c.GetOptions(test)
	//iterate through sections of config file
	for _, checkName := range checks {
		if checkName == "metric-name" {
			continue
		}
		m.checks[checkName], _ = c.GetString(test, checkName)
	}
	return *m
}

//Returns results of metrics check
func (hc *checker) GetWarnings() map[string]metricResults {
	return hc.Warnings
}

//initialize constants defined in the config file
func (hc *checker) setupConstants() error {
	constants, err := hc.c.GetOptions("constants")
	if err != nil {
		return err
	}
	// collect constants in the form of a string as such"
	// " package p
	//   const1 = val1
	//   const2 = val2 ..."
	src := "package p\n"
	for _, name := range constants {
		val, _ := hc.c.GetString("constants", name)
		src += "const " + name + " = " + val + "\n"
	}
	//Parse src as if it were a go source code file
	// and save the package and scope, so that metrics checks
	// evaluated in the same package and scope can use the same
	// constants
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p", src, 0)
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	hc.pkg, err = types.Check("p", fset, []*ast.File{file})
	if err != nil {
		hc.Logger.Println(err)
		return err
	}
	hc.scope = hc.pkg.Scope().Child(0)
	return nil
}
