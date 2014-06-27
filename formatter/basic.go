package formatter

import (
	"fmt"
	"strconv"

	"github.com/measure/metrics/check"
)

// format result of a check to a human readable
// message

func Basic(hc check.Checker, s ...string) error {
	for metric, result := range hc.GetWarnings() {
		fmt.Println(metric + ": " + result.Message)
		for checkName, val := range result.Checks {
			fmt.Println("    " + checkName + ": " + strconv.FormatBool(val))
		}
	}
	return nil
}
