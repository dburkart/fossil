/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"github.com/dburkart/fossil/pkg/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = &cobra.Command{
	Use:   "server",
	Short: "Database for collecting and querying metrics",

	Run: func(cmd *cobra.Command, args []string) {
		logger := viper.Get("logger").(zerolog.Logger)

		// Initialize database server
		srv := server.New(
			logger,
			viper.GetString("database.directory"),
			viper.GetInt("fossil.port"),
			viper.GetInt("fossil.prom-http"),
		)

		// Serve the database
		go srv.ServeDatabase()

		// Serve the metrics endpoint
		srv.ServeMetrics()
	},
}

func init() {
	// Flags for this command
	Command.Flags().IntP("port", "p", 8001, "Database server port for data collection")
	Command.Flags().Int("prom-http", 2112, "Set the port for /metrics is bound to")
	Command.Flags().StringP("database", "d", "./", "Path to store database files")

	// Bind flags to viper
	viper.BindPFlag("fossil.port", Command.Flags().Lookup("port"))
	viper.BindPFlag("fossil.prom-port", Command.Flags().Lookup("prom-http"))
	viper.BindPFlag("database.directory", Command.Flags().Lookup("database"))
}
