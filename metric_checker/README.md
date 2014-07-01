# metric checker

##example usage

`./metric_checker -address localhost:12345 -cnf ./inspect.confi`

## configuration file format
```
# example comment line

#each section name should be unique
[section name]

#expr indicates the expression to be evaluated
expr = value >= 30

#true indicates the message to send if expr evaluates to true
true = check passed

#false indicates the message to send if expr evaluates to false
false = check failed

[another example]
expr = mysqlstat_somemetric_value < 10
true = check failed

# true or false are not required in each section, there will just
# be no output if the expr evaluates to the missing option
```
Currently, metric gauges' values are accessed by `metric_name_value`. 
Counter values are accessed with `metric_name_current`.
Counter rates are accessed with `metric_name_rate`.
