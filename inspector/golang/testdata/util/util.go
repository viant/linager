package util

import (
	"fmt"
	"reflect"
	"runtime"
)

func Inspect[T any](target T) {
	fmt.Println("=== Inspector Report ===")
	t := reflect.TypeOf(target)
	fmt.Printf("Type: %s\n", t.String())

	// Print package path
	if t.Kind() == reflect.Ptr {
		fmt.Println("Package:", t.Elem().PkgPath())
	} else {
		fmt.Println("Package:", t.PkgPath())
	}

	// Print call stack info
	pc := make([]uintptr, 10)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])

	for {
		frame, more := frames.Next()
		fmt.Printf("Called from: %s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	fmt.Println("========================")
}
