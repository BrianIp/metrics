//Copyright (c) 2014 Square, Inc

package check

type WalkFunc func() bool

type Checker interface {
	// Walk through all the checks defined and invoke walkFn
	// return false from walkFn to abort
	Walk() error
}
