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
	"net"
	"os"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
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
			c, err := net.Dial("tcp4", target.Address)
			if err != nil {
				log.Error().Err(err).Str("address", target.Address).Msg("unable to connect to server")
			}

			// Always send a use first
			useMsg := proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: target.Database})
			b, _ := useMsg.Marshal()
			c.Write(b)
			buf := make([]byte, 256)
			c.Read(buf)
			m, err := proto.ParseMessage(buf)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to parse server use response")
			}
			ok := proto.OkResponse{}
			err = ok.Unmarshal(m.Data)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to parse server use response")
			}
			fmt.Println(ok.Code, ok.Message)

			clientPrompt(c)
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

		err = send(c, line)
		if err != nil {
			fmt.Printf("Err: unable to send command\n\t'%s'\n", err)
		}

		respRdr := bufio.NewReader(c)
		for {
			resp, err := respRdr.ReadBytes('\n')
			if err != nil {
				fmt.Printf("Err: unable to read response\n\t'%s'\n", string(resp))
				continue
			}
			msg, err := proto.ParseMessage(resp)
			if err != nil {
				log.Error().Err(err).Msg("malformed message")
				fmt.Println(resp)
				continue
			}
			switch msg.Command {
			case proto.CommandStats:
				t := proto.StatsResponse{}
				fmt.Println(msg.Data)
				fmt.Println(string(msg.Data))
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
			if respRdr.Buffered() <= 0 {
				break
			}
		}
	}
}
