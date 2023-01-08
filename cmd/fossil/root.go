/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"fmt"
	"os"

	"github.com/dburkart/fossil/cmd/fossil/client"
	"github.com/dburkart/fossil/cmd/fossil/server"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version        = "develop"
	CommitHash     = "n/a"
	BuildTimestamp = "n/a"

	rootCmd = &cobra.Command{
		Use:   "fossil",
		Short: "Fossil is a small and fast tsdb",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogging()
			initLogLevel()
			initConfig(cmd.Root().PersistentFlags().Lookup("config").Value.String())
			initLogLevel()
			traceConfig()
		},
		Version: Version,
	}
)

func init() {
	// Configure the root binary options
	rootCmd.PersistentFlags().CountP("verbose", "v", "-v for debug logs (-vv for trace)")
	rootCmd.PersistentFlags().Bool("local", true, "Configures the logger to print readable logs") //TODO: true until we have a config file format
	rootCmd.PersistentFlags().StringP("host", "H", "./default", "Host to send the messages")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to the fossil config file (default ./config.toml)")

	// Bind viper config to the root flags
	viper.BindPFlag("fossil.local", rootCmd.PersistentFlags().Lookup("local"))
	viper.BindPFlag("fossil.verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("fossil.host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.SetVersionTemplate(fmt.Sprintf("fossil version: %s git_commit: %s build_time: %s\n", Version, CommitHash, BuildTimestamp))

	// Bind viper flags to ENV variables
	// viper.SetEnvPrefix("FOSSIL")
	// viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Register commands on the root binary command
	server.Command.Version = rootCmd.Version
	client.Command.Version = rootCmd.Version
	rootCmd.AddCommand(server.Command)
	rootCmd.AddCommand(client.Command)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("root command failed")
		os.Exit(1)
	}
}
