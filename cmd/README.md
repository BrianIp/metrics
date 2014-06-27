#metric checks

## Usage

### Command Line Utility

```
./bin/check -conf in.cfg
```



Can be run with the `-conf` option to specify path to the config file 
described in the section below. The `-hostport` flag will specify the 
host port to listen on for metrics. The `-nagConf` option specifies to 
the path to the nagios configuration file. `-basic=true` and `-nagios=true` will set the output format to basic and nagios, respectively.

### Config File

The config file specifies the checks that will be done on the metrics values. An example:
```
[constants]
default_pct = 50

[metric1]
check1 = metric1.Value < 16384
check2 = metric1.Value == 16384

[test rates]
check rate = metric1.Rate > 700
check rate 2 = metric2.Rate > 900

[check user percentage]
user 50 pct = cpustat.cpu.User.Value / cpustat.cpu.Total.Value * 100 > default_pct
user 30 pct = cpustat.cpu.User.Value / cpustat.cpu.Total.Value * 100 > 30
```
Each section title serves to describe its set of metric checks. Only the sections `default` and `nagios` and `constants` are reserved for special information. The fields in the section will specify the checks. On the left hand side of the `=` is the name of the check; each name must be unique within its section. On the right hand side is the check that will be performed. Metrics can be specified by their full name, and will be replaced by the appropriate value. Constants can be defined in the `[constants]` section, as shown in the example above.


## Testing
Testing is done using Go's testing packages and tests can be found in the `check_test.go` file.
To run tests, cd to the measure/metrics/check directory and run `go test`.
