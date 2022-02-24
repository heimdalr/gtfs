package main

import (
	"fmt"
	"heimdalr/gtfs/cmd/gtfs/commands"
	"os"
)

func main() {
	err := commands.Execute()
	if err != nil && err.Error() != "" {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
