/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dburkart/fossil/api"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
	"github.com/dburkart/fossil/pkg/repl"
	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var log zerolog.Logger

var Command = &cobra.Command{
	Use:   "client",
	Short: "Interactive terminal for communicating with the server",

	Run: func(cmd *cobra.Command, args []string) {
		log := viper.Get("logger").(zerolog.Logger)

		host := viper.GetString("fossil.host")
		target := proto.ParseConnectionString(host)

		if target.Local {
			db, err := database.NewDatabase(log, target.Database, target.Database)
			if err != nil {
				log.Fatal().Err(err).Msg("error creating new database")
			}

			localPrompt(db)
		} else {
			client, err := fossil.NewClient(host)
			if err != nil {
				log.Error().Err(err).Str("address", target.Address).Msg("unable to connect to server")
				os.Exit(1)
			}

			// REPL
			clientPrompt(client)
		}
	},
}

func init() {
	log = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).
		With().
		Timestamp().
		Caller().
		Logger()
}

func localPrompt(db *database.Database) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal().Err(err)
		}

		stmt, err := query.Prepare(db, line)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		result := stmt.Execute()

		for _, val := range result.Data {
			fmt.Println(val.ToString())
		}
	}
}

func clientPrompt(c fossil.Client) {
	defer c.Close()

	// Check whether stdin is a pipe, since we'll want to make different choices
	// in terms of prompt and repl termination.
	piped := false
	stdin, _ := os.Stdin.Stat()
	if (stdin.Mode() & os.ModeCharDevice) == 0 {
		piped = true
	}

	exit := false
	history := []string{}
	rdr := bufio.NewReader(os.Stdin)
	for !exit {
		if !piped {
			fmt.Printf("\n> ")
		}
		line, err := rdr.ReadBytes('\n')
		line = line[:len(line)-1]
		history = append(history, string(line))
		if err != nil {
			if piped {
				return
			}
			fmt.Printf("Err: unable to read input\n\t'%s'\n", string(line))
		}

		if strings.HasPrefix(string(line), "QUIT") {
			fmt.Println("bye!")
			return
		}
		log.Trace().Str("line", string(line)).Bytes("buf", line).Msg("sending")
		replMsg := repl.ParseREPLCommand(line)
		msg, err := c.Send(replMsg)
		if err != nil {
			log.Error().Err(err).Msg("error sending message to server")
		}

		switch msg.Command {
		case proto.CommandVersion:
			v := proto.VersionResponse{}
			err = v.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Printf("%d %s\n", v.Code, v.Version)
		case proto.CommandStats:
			t := proto.StatsResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Printf(
				"Allocated Heap: %v\nTotal Memory: %v\nUptime: %s\nSegments: %d\n",
				humanize.Bytes(t.AllocHeap),
				humanize.Bytes(t.TotalMem),
				t.Uptime,
				t.Segments,
			)
		case proto.CommandQuery:
			t := proto.QueryResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Time", "Topic", "Data"})
			for i := range t.Results {
				table.Append([]string{
					t.Results[i].Time.Format(time.RFC3339Nano),
					t.Results[i].Topic,
					string(t.Results[i].Data),
				})
			}

			table.Render()
		case proto.CommandError:
			t := proto.ErrResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Err)
		case proto.CommandOk:
			t := proto.OkResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Message)
		case proto.CommandAppend:
			t := proto.OkResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Message)
		}
	}
}
