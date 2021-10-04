package main

import (
	"fmt"
	"log"
	"os"
	"time"
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

func Retry(fn func() error) error {
	return retry(5, time.Second, fn)
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, fn)
		}
		return err
	}
	return nil
}

type stop struct {
	error
}
