/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package client

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = &cobra.Command{
	Use:   "client",
	Short: "Interactive terminal for communicating with the server",

	Run: func(cmd *cobra.Command, args []string) {
		log := viper.Get("logger").(zerolog.Logger)

		host := viper.GetString("host")
		c, err := net.Dial("tcp4", host)
		if err != nil {
			log.Error().Err(err).Str("host", host).Msg("unable to connect to server")
		}

		prompt(c)
	},
}

func send(c net.Conn, msg []byte) error {
	n, err := c.Write(msg)
	if err != nil {
		log.Error().Err(err).Msg("unable to write to server")
		return err
	}
	log.Trace().Int("bytes", n).Msg("message sent")
	return nil
}

func prompt(c net.Conn) {
	exit := false
	history := []string{}
	for !exit {
		fmt.Printf("\n> ")
		rdr := bufio.NewReader(os.Stdin)
		line, err := rdr.ReadBytes('\n')
		history = append(history, string(line))
		if err != nil {
			fmt.Printf("Err: unable to read input\n\t'%s'\n", string(line))
		}

		err = send(c, line)
		if err != nil {
			fmt.Printf("Err: unable to send command\n\t'%s'\n", err)
		}
	}
}

func init() {
	// Flags for this command
	Command.Flags().StringP("host", "H", "localhost:8001", "Host to send the messages")

	// Bind flags to viper
	viper.BindPFlag("host", Command.Flags().Lookup("host"))
}
