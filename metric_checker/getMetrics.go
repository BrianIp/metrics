package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"reflect"
	"strings"

	"code.google.com/p/go.tools/go/exact"
	_ "code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goconf/conf"
	"github.com/measure/metrics"
)

func main() {
	//TODO: get hostport from user
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "./getMetrics.go", nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	pkg, err := types.Check("main", fset, []*ast.File{f})
	sc := pkg.Scope()

	err = insertMetricValues("localhost:12345", sc, pkg)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = checkAll("./inspect.config", pkg, sc)
	if err != nil {
		fmt.Println(err)
	}
}

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
		fmt.Println(expr)
		_, r, err := types.Eval(expr, pkg, sc)
		if err != nil {
			return err
		}
		if exact.BoolVal(r) {
			message, _ := cnf.GetString(section, "true")
			fmt.Println(message)
		} else {
			message, _ := cnf.GetString(section, "false")
			fmt.Println(message)
		}
	}
	return nil
}

func insertMetricValues(hostport string, sc *types.Scope, pkg *types.Package) error {
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

	for _, m := range metrics {
		switch val := m.Value.(type) {
		case float64:
			name := strings.Replace(m.Name, ".", "_", -1) + "_value"
			sc.Insert(types.NewConst(0, pkg, name,
				types.New("float64"), exact.MakeFloat64(val)))
		case map[string]interface{}:
			//TODO: make sure we don't panic in case something is not formatted
			// like expected
			name := strings.Replace(m.Name, ".", "_", -1) + "_current"
			sc.Insert(types.NewConst(0, pkg, name,
				types.New("float64"), exact.MakeFloat64(val["current"].(float64))))
			name = strings.Replace(m.Name, ".", "_", -1) + "_rate"
			sc.Insert(types.NewConst(0, pkg, name,
				types.New("float64"), exact.MakeFloat64(val["rate"].(float64))))
		default:
			fmt.Println(reflect.TypeOf(val))
		}
	}
	return nil
}
