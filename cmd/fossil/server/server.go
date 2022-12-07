/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"path/filepath"
	"strings"

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
			buildDatabaseConfigs(),
			viper.GetInt("fossil.port"),
			viper.GetInt("fossil.prom-port"),
		)

		// Serve the database
		go srv.ServeDatabase()

		// Serve the metrics endpoint
		srv.ServeMetrics()
	},
}

func buildDatabaseConfigs() map[string]server.DatabaseConfig {
	ret := make(map[string]server.DatabaseConfig)

	for _, v := range viper.GetStringSlice("database.names") {
		// If this is a non-default db look up the config value for it
		dbConfig := server.DatabaseConfig{
			Name:      v,
			Directory: viper.GetString(strings.Join([]string{"database", v, "directory"}, ".")),
		}

		// If this is the default, use the [database] block value
		if v == "default" {
			dbConfig.Directory = filepath.Clean(viper.GetString("database.directory"))
		}

		ret[v] = dbConfig
	}

	// After the config has been loaded, any database block without a value receives the default directory
	for k, v := range ret {
		// Set the db directory to `defaultpath`
		if v.Directory == "" {
			v.Directory = ret["default"].Directory
		}
		v.Directory = filepath.Clean(v.Directory)
		ret[k] = v
	}

	return ret
}

func init() {
	// Flags for this command
	Command.Flags().IntP("port", "p", 8001, "Database server port for data collection")
	Command.Flags().Int("prom-port", 2112, "Set the port for /metrics")
	Command.Flags().StringP("database", "d", "./", "Path to store database files")

	// Bind flags to viper
	viper.BindPFlag("fossil.port", Command.Flags().Lookup("port"))
	viper.BindPFlag("fossil.prom-port", Command.Flags().Lookup("prom-port"))
	viper.BindPFlag("database.directory", Command.Flags().Lookup("database"))
}
