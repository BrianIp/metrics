//Copyright (c) 2014 Square, Inc

//Compares values of metrics collected against thresholds specified in a config
// file. The script gathers these metrics by listening for json packages
// on an address specified by the user.
// The user specifies the config file to grab these checks from.
// Currently, in the config file, in each section is an expr that is evaluated,
// and messages if the expr evaluates to true/false. These messages are sent
// to stdout.
// see the readme for the formatting of the config file.

package check

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
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

type checker struct {
	sc         *types.Scope
	pkg        *types.Package
	hostport   string
	configFile string
}

func New(hostport string, configFile string) (Checker, error) {
	c := &checker{
		hostport:   hostport,
		configFile: configFile,
	}
	return c, nil
}

func (c *checker) NewScopeAndPackage() error {
	fset := token.NewFileSet()
	src := `package p`
	f, err := parser.ParseFile(fset, "p", src, 0)
	if err != nil {
		return err
	}
	//initialize package and scope to evaluate expressions
	c.pkg, err = types.Check("main", fset, []*ast.File{f})
	if err != nil {
		return err
	}
	c.sc = c.pkg.Scope()
	return nil
}

//ranges through config file and checks all expressions.
// prints result messages to stdout
func (c *checker) CheckAll(w io.Writer) ([]string, error) {
	result := []string{}
	cnf, err := conf.ReadConfigFile(c.configFile)
	if err != nil {
		return nil, err
	}
	for _, section := range cnf.GetSections() {
		if section == "default" {
			continue
		}
		expr, _ := cnf.GetString(section, "expr")
		_, r, err := types.Eval(expr, c.pkg, c.sc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		message := ""
		var m string
		owner, err := cnf.GetString(section, "owner")
		if err == nil {
			message = message + owner
		} else {
			message = message + "owner: unknown"
		}
		if exact.BoolVal(r) {
			m, err = cnf.GetString(section, "true")
			if err != nil {
				continue
			}
		} else {
			m, err = cnf.GetString(section, "false")
			if err != nil {
				continue
			}
		}
		_, msg, err := types.Eval(m, c.pkg, c.sc)
		if err != nil {
			result = append(result, message+" | "+m)
			fmt.Println(err)
		} else {
			result = append(result, message+" | "+exact.StringVal(msg))
		}
	}
	return result, nil
}

//insertMetricValues inserts the values and rates of the metrics collected
// as constants into the scope used to evaluate the expressions
func (c *checker) InsertMetricValuesFromJSON() error {
	//get metrics from json package
	//TODO: get directly from metric context if available
	resp, err := http.Get("http://" + c.hostport + "/api/v1/metrics.json/")
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
			c.sc.Insert(types.NewConst(0, c.pkg, name,
				types.New("float64"), exact.MakeFloat64(val)))
		case map[string]interface{}:
			//TODO: make sure we don't panic in case something is not formatted
			// like expected
			if current, ok := val["current"]; ok {
				name := strings.Replace(m.Name, ".", "_", -1) + "_current"
				c.sc.Insert(types.NewConst(0, c.pkg, name,
					types.New("float64"), exact.MakeFloat64(current.(float64))))
			}
			if rate, ok := val["rate"]; ok {
				name := strings.Replace(m.Name, ".", "_", -1) + "_rate"
				c.sc.Insert(types.NewConst(0, c.pkg, name,
					types.New("float64"), exact.MakeFloat64(rate.(float64))))
			}
		default:
			//a value type came up that wasn't anticipated
			fmt.Fprintln(os.Stderr, reflect.TypeOf(val))
		}
	}
	return nil
}

func (c *checker) InsertMetricValuesFromContext(m *metrics.MetricContext) error {
	for metricName, metric := range m.Gauges {
		name := strings.Replace(metricName, ".", "_", -1) + "_value"
		c.sc.Insert(types.NewConst(0, c.pkg, name,
			types.New("float64"), exact.MakeFloat64(metric.Get())))
		sname := name + "_string"
		c.sc.Insert(types.NewConst(0, c.pkg, sname,
			types.New("string"), exact.MakeString(fmt.Sprintf("%0.2f", metric.Get()))))
	}
	for metricName, metric := range m.Counters {
		name := strings.Replace(metricName, ".", "_", -1) + "_current"
		c.sc.Insert(types.NewConst(0, c.pkg, name,
			types.New("float64"), exact.MakeUint64(metric.Get())))
		sname := name + "_string"
		c.sc.Insert(types.NewConst(0, c.pkg, sname,
			types.New("string"), exact.MakeString(fmt.Sprintf("%d", metric.Get()))))
		name = strings.Replace(metricName, ".", "_", -1) + "_rate"
		c.sc.Insert(types.NewConst(0, c.pkg, name,
			types.New("float64"), exact.MakeFloat64(metric.ComputeRate())))
	}
	return nil
}
