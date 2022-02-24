package commands

import (
	"github.com/spf13/cobra"
	"log"
)

// NewRootCmd initializes the root command.
func NewRootCmd(buildVersion, buildGitHash string) *cobra.Command {

	gtfsTrimCmd := &cobra.Command{
		Use:   "trim <dbPath> <agency>",
		Short: "Trim a GTFS DB to a single agency",
		Long:  ``,
		RunE:  gtfsTrim,
		Args:  cobra.ExactArgs(2),
	}

	gtfsImportCmd := &cobra.Command{
		Use:   "import <gtfsBasePath> <dbPath>",
		Short: "Import GTFS data files into an SQLite DB",
		Long:  ``,
		RunE:  gtfsImport,
		Args:  cobra.ExactArgs(2),
	}

	gtfsVersionCmd := &cobra.Command{
		Use:   "version",
		Short: "Get program version",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			log.Printf("version: %s hash: %s", buildVersion, buildGitHash)
		},
	}

	rootCmd := &cobra.Command{
		Use:           "gtfs",
		Short:         "gtfs - GTFS command line tool",
		Long:          ``,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(gtfsImportCmd)
	rootCmd.AddCommand(gtfsTrimCmd)
	rootCmd.AddCommand(gtfsVersionCmd)

	return rootCmd
}
