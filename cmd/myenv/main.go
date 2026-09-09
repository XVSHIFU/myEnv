package main

import (
	"myenv/internal/cli"
	"myenv/internal/runtrace"
	"os"
)

var version = "0.1.0-dev"

func main() {
	runtrace.Mark("business_enter")
	code := cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, version)
	runtrace.Mark("cli_finished")
	runtrace.Flush()
	os.Exit(code)
}
