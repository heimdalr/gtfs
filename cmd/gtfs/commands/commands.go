package commands

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:           "gtfs",
		Short:         "gtfs - command line tool",
		Long:          ``,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

func init() {
	rootCmd.AddCommand(gtfsImportCmd)
	rootCmd.AddCommand(gtfsTrimCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
