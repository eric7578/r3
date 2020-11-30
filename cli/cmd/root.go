package cmd

import (
	"fmt"
	"os"

	"github.com/eric7578/r3"
	"github.com/spf13/cobra"
)

var (
	port      string
	configDir string
)

var rootCmd = &cobra.Command{
	Use:   "r3",
	Short: "run r3 server",
	Run: func(cmd *cobra.Command, args []string) {
		d := r3.NewDaemon(configDir)
		d.Run(port)
	},
}

func init() {
	rootCmd.Flags().StringVar(&port, "port", ":9009", "server port")
	rootCmd.Flags().StringVar(&configDir, "config", "", "config directory")
}

func Execute() {
	exitOnError(rootCmd.Execute())
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
