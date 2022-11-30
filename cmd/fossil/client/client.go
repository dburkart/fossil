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

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
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
		target := proto.ParseConnectionString(host)

		if target.Local {
			db := database.NewDatabase(target.Database)

			localPrompt(db)
		} else {
			c, err := net.Dial("tcp4", target.Address)
			if err != nil {
				log.Error().Err(err).Str("address", target.Address).Msg("unable to connect to server")
			}

			clientPrompt(c)
		}
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

func localPrompt(db *database.Database) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal().Err(err)
		}

		stmt := query.Prepare(db, line)
		result := stmt.Execute()

		for _, val := range result.Data {
			fmt.Println(val.ToString())
		}
	}
}

func clientPrompt(c net.Conn) {
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

		if strings.HasPrefix(string(line), "QUIT") {
			fmt.Println("bye!")
			return
		}

		err = send(c, line)
		if err != nil {
			fmt.Printf("Err: unable to send command\n\t'%s'\n", err)
		}

		respRdr := bufio.NewReader(c)
		for {

			resp, err := respRdr.ReadBytes('\n')
			if err != nil {
				fmt.Printf("Err: unable to read response\n\t'%s'\n", string(resp))
			}
			fmt.Print(string(resp))
			if respRdr.Buffered() <= 0 {
				break
			}
		}
	}
}

func init() {
	// Flags for this command
	// Command.Flags().StringP("host", "H", "fossil://local/default", "Host to send the messages")

	// Bind flags to viper
	// viper.BindPFlag("host", Command.Flags().Lookup("host"))
}
