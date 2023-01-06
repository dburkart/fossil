/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func initConfig(configFile string) {
	log := viper.Get("logger").(zerolog.Logger)

	// config Read
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath("/etc/fossil")
	viper.AddConfigPath("/usr/local/etc/fossil")
	viper.AddConfigPath("$HOME/.fossil")
	viper.AddConfigPath(".")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Debug().Msg("No config file found, using defaults as a base")
	} else if err != nil {
		log.Error().Msg("Error loading config file")
	}

	log.Debug().Str("file", viper.ConfigFileUsed()).Msg("loaded config from file")

	databaseConfigs := viper.GetStringMap("database")
	databases := []string{}

	// Range over the database blocks to set a list of database names
	for k, v := range databaseConfigs {
		// Root fossil database blocks
		if t := reflect.ValueOf(v); t.Kind() == reflect.Map {
			databases = append(databases, k)
			for dbk, dbv := range v.(map[string]interface{}) {
				log.Trace().Msgf("database.%s.%s = %v", k, dbk, dbv)
				viper.Set(fmt.Sprintf("database.%s.%s", k, dbk), dbv)
			}
		} else {
			log.Trace().Msgf("database.default.%s = %v", k, v)
			viper.Set(fmt.Sprintf("database.default.%s", k), v)
		}
	}
	if len(databases) < len(databaseConfigs) || len(databases) == 0 {
		databases = append(databases, "default")
	}

	viper.Set("database", "")
	viper.Set("database.names", databases)
}

func initLogLevel() {
	level := viper.GetInt("fossil.verbose")
	switch clamp(2, level) {
	case 2:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case 1:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func initLogging() {
	var writer io.Writer

	writer = os.Stderr
	if viper.GetBool("fossil.local") {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Caller().
		Logger()

	viper.Set("logger", logger)
}

func traceConfig() {
	log := viper.Get("logger").(zerolog.Logger)

	for _, v := range viper.AllKeys() {
		if v == "logger" {
			continue
		}
		log.Trace().Msgf("%s=%v", v, viper.Get(v))
	}
}

func clamp(clamp, a int) int {
	if a >= clamp {
		return clamp
	}
	return a
}
