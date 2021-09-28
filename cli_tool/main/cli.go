package main

import (
	"fmt"
	"log"
	"os"
)

var Verbose bool = true

func Log(isVerbose bool, message string, tabIndex int, params ...interface{}) {
	if isVerbose && !Verbose {
		return
	}
	ret := ""
	for i := 0; i < tabIndex; i++ {
		ret += "\t"
	}
	ret += message
	log.Printf(ret, params...)
}

func LogFatal(message string, params ...interface{}) {
	log.Fatalf(message, params...)
}

func main() {

	if err := root(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
