// Package main is a package that contains subcommands for the spectest CLI command.
package main

import (
	"os"

	"github.com/nao1215/spectest/cmd/spectest/sub"
)

// osExit is wrapper for  os.Exit(). It's for unit test.
var osExit = os.Exit //nolint

func main() {
	osExit(sub.Execute())
}
