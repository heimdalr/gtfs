package main

import (
	"fmt"
	"github.com/heimdalr/gtfs/cmd/gtfs/commands"
	"os"
)

var (
	buildVersion = "to be set by linker"
	buildGitHash = "to be set by linker"
)

func main() {
	c := commands.NewRootCmd(buildVersion, buildGitHash)
	err := c.Execute()
	if err != nil && err.Error() != "" {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
