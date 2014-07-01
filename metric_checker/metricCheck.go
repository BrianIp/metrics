//Copyright (c) 2014 Square, Inc

//Compares values of metrics collected against thresholds specified in a config
// file. The script gathers these metrics by listening for json packages
// on an address specified by the user.
// The user specifies the config file to grab these checks from.
// Currently, in the config file, in each section is an expr that is evaluated,
// and messages if the expr evaluates to true/false. These messages are sent
// to stdout.
// see the readme for the formatting of the config file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"reflect"
	"strings"

	"code.google.com/p/go.tools/go/exact"
	_ "code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goconf/conf"
	"github.com/measure/metrics"
)

func main() {
	var hostport, configfile string
	flag.StringVar(&hostport, "address", "localhost:12345",
		"address to listen on for metrics json")
	flag.StringVar(&configfile, "cnf", "",
		"config file to grab thresholds from")
	flag.Parse()
	fset := token.NewFileSet()
	src := `package p`
	f, err := parser.ParseFile(fset, "p", src, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//initialize package and scope to evaluate expressions
	pkg, err := types.Check("main", fset, []*ast.File{f})
	sc := pkg.Scope()
	//insert metric values as constants into scope
	err = insertMetricValues(hostport, sc, pkg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//check all expressions provided in config file
	err = checkAll(configfile, pkg, sc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

//ranges through config file and checks all expressions.
// prints result messages to stdout
func checkAll(configfile string, pkg *types.Package, sc *types.Scope) error {
	cnf, err := conf.ReadConfigFile(configfile)
	if err != nil {
		return err
	}
	for _, section := range cnf.GetSections() {
		if section == "default" {
			continue
		}
		expr, _ := cnf.GetString(section, "expr")
		_, r, err := types.Eval(expr, pkg, sc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if exact.BoolVal(r) {
			message, err := cnf.GetString(section, "true")
			if err == nil {
				//fmt.Fprintln(expr)
				fmt.Fprintf(os.Stdout, message)
			}
		} else {
			message, err := cnf.GetString(section, "false")
			if err == nil {
				//fmt.Fprintln(expr)
				fmt.Fprintf(os.Stdout, message)
			}
		}
	}
	return nil
}

//insertMetricValues inserts the values and rates of the metrics collected
// as constants into the scope used to evaluate the expressions
func insertMetricValues(hostport string, sc *types.Scope, pkg *types.Package) error {
	//get metrics from json package
	//TODO: get directly from metric context if available
	resp, err := http.Get("http://" + hostport + "/api/v1/metrics.json/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	var metrics []metrics.MetricJSON
	err = d.Decode(&metrics)
	if err != nil {
		return err
	}

	//insert metric value into scope
	for _, m := range metrics {
		switch val := m.Value.(type) {
		case float64:
			name := strings.Replace(m.Name, ".", "_", -1) + "_value"
			sc.Insert(types.NewConst(0, pkg, name,
				types.New("float64"), exact.MakeFloat64(val)))
		case map[string]interface{}:
			//TODO: make sure we don't panic in case something is not formatted
			// like expected
			if current, ok := val["current"]; ok {

				name := strings.Replace(m.Name, ".", "_", -1) + "_current"
				sc.Insert(types.NewConst(0, pkg, name,
					types.New("float64"), exact.MakeFloat64(current.(float64))))
			}
			if rate, ok := val["rate"]; ok {
				name := strings.Replace(m.Name, ".", "_", -1) + "_rate"
				sc.Insert(types.NewConst(0, pkg, name,
					types.New("float64"), exact.MakeFloat64(rate.(float64))))
			}
		default:
			//a value type came up that wasn't anticipated
			fmt.Fprintln(os.Stderr, reflect.TypeOf(val))
		}
	}
	return nil
}
