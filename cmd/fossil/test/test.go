/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package test

import (
	"net"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = &cobra.Command{
	Use:   "test",
	Short: "Send a series of test messages to the server",

	Run: func(cmd *cobra.Command, args []string) {
		log := viper.Get("logger").(zerolog.Logger)

		host := viper.GetString("host")
		c, err := net.Dial("tcp4", host)
		if err != nil {
			log.Error().Err(err).Str("host", host).Msg("unable to connect to server")
		}

		msg := []byte("INFO all\nINFO all\nINFO all\nINFO all\n")
		n, err := c.Write(msg)
		if err != nil {
			log.Error().Err(err).Str("host", host).Msg("unable to write to server")
		}
		log.Info().Int("bytes", n).Msg("message sent")
	},
}

func init() {
	// Flags for this command
	Command.Flags().StringP("host", "H", "localhost:8001", "Host to send the messages")

	// Bind flags to viper
	viper.BindPFlag("host", Command.Flags().Lookup("host"))
}
