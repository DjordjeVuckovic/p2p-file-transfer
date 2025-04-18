package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"p2p-file-transfer/pkg/zipper"
)

func main() {
	var (
		outputFile string
	)
	rootCmd := &cobra.Command{
		Use:   "zipper [files/directories...]",
		Short: "A tool to zip project files with ignore patterns",
		Long: `A flexible tool that creates zip archives of project files
while respecting ignore patterns from .gitignore and similar files.
Automatically excludes common build artifacts like node_modules and bin directories.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Set default output file if not specified
			if outputFile == "" {
				outputFile = zipper.DefaultOutputPath
			}
			err := zipper.Zip(zipper.WithPaths(args, outputFile))
			if err != nil {
				fmt.Printf("Error creating zip: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output zip file path (default is output.zip)")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
