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

	"github.com/chzyer/readline"
	fossil "github.com/dburkart/fossil/api"
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

var (
	Command = &cobra.Command{
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
				}

				// REPL
				readlinePrompt(client)
			}
		},
	}
)

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

func listDatabases(c fossil.Client) func(string) []string {
	msg, err := c.Send(proto.NewMessageWithType(proto.CommandList, proto.ListRequest{}))
	if err != nil {
		return func(string) []string { return []string{} }
	}
	resp := proto.ListResponse{}
	err = resp.Unmarshal(msg.Data)
	if err != nil {
		return func(string) []string { return []string{} }
	}
	return func(line string) []string {
		return resp.ObjectList
	}
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func readlinePrompt(c fossil.Client) {
	// Configure the completer
	useItem := readline.PcItemDynamic(listDatabases(c))
	completer := readline.NewPrefixCompleter(
		readline.PcItem("USE", useItem),
		readline.PcItem("use", useItem),
		readline.PcItem("APPEND"),
		readline.PcItem("append"),
		readline.PcItem("INSERT"),
		readline.PcItem("insert"),
		readline.PcItem("QUERY"),
		readline.PcItem("query"),
		readline.PcItem("EXIT"),
		readline.PcItem("exit"),
		readline.PcItem("LIST"),
		readline.PcItem("list"),
	)

	// Setup the readline executor
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m>\033[0m ",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// Handle input
	for {
		ln := rl.Line()
		if ln.CanContinue() {
			continue
		} else if ln.CanBreak() {
			break
		}
		line := strings.TrimSpace(ln.Line)

		if strings.ToUpper(line) == "EXIT" {
			os.Exit(0)
		}

		replMsg, err := repl.ParseREPLCommand([]byte(line))
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}
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
		case proto.CommandList:
			t := proto.ListResponse{}
			err = t.Unmarshal(msg.Data)
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			for _, v := range t.ObjectList {
				fmt.Println(v)
			}
		}
		fmt.Println()
	}
	rl.Clean()
}
