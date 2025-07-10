package main

import (
	"fmt"
	"log"
	"os"

	"github.com/javanhut/harbinger/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "harbinger",
		Short: "A Git conflict monitoring tool that notifies you when your branch needs attention",
		Long: `Harbinger monitors your Git repository in the background and notifies you when:
- Your branch is out of sync with the remote
- There are potential merge conflicts
- Remote changes might affect your work

It provides an interactive conflict resolution interface right in your terminal.`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.harbinger.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		config.SetConfigPath(home)
		config.SetConfigName(".harbinger")
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
