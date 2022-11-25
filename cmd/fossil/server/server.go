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
			viper.GetInt("collection-port"),
			viper.GetInt("database-port"),
			viper.GetInt("prom-http"),
		)

		// Serve the database
		go srv.ServeDatabase()

		// Serve the metrics endpoint
		srv.ServeMetrics()
	},
}

func init() {
	// Flags for this command
	Command.Flags().IntP("collection-port", "c", 8001, "Database server port for data collection")
	Command.Flags().IntP("database-port", "p", 8000, "Database server port for client connections")
	Command.Flags().Int("prom-http", 2112, "Set the port for /metrics is bound to")
	Command.Flags().StringP("database", "d", "./", "Path to store database files")

	// Bind flags to viper
	viper.BindPFlag("collection-port", Command.Flags().Lookup("collection-port"))
	viper.BindPFlag("database-port", Command.Flags().Lookup("database-port"))
	viper.BindPFlag("prom-http", Command.Flags().Lookup("prom-http"))
	viper.BindPFlag("database", Command.Flags().Lookup("database"))
}
