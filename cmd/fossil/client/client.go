/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/dburkart/fossil/pkg/schema"
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
			target, err := proto.ParseConnectionString(host)
			if err != nil {
				log.Fatal().Err(err).Msg("error parsing URL")
			}

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
	err = resp.Unmarshal(msg.Data())
	if err != nil {
		return func(string) []string { return []string{} }
	}
	return func(line string) []string {
		lineTopic := line
		if strings.HasPrefix(line, "USE") || strings.HasPrefix(line, "use") {
			lineTopic = lineTopic[4:]
		}

		return filterStringSlice(resp.ObjectList, lineTopic)
	}
}

func listTopics(c fossil.Client) func(string) []string {
	msg, err := c.Send(proto.NewMessageWithType(proto.CommandList, proto.ListRequest{Object: "topics"}))
	if err != nil {
		return func(string) []string { return []string{} }
	}
	resp := proto.ListResponse{}
	err = resp.Unmarshal(msg.Data())
	if err != nil {
		return func(string) []string { return []string{} }
	}
	return func(line string) []string {
		lineTopic := line
		if strings.HasPrefix(line, "APPEND") || strings.HasPrefix(line, "append") {
			lineTopic = lineTopic[7:]
		}

		return filterStringSlice(resp.ObjectList, lineTopic)
	}
}

func listSchemas(c fossil.Client) map[string]schema.Object {
	msg, err := c.Send(proto.NewMessageWithType(proto.CommandList, proto.ListRequest{Object: "schemas"}))
	if err != nil {
		return nil
	}
	resp := proto.ListResponse{}
	err = resp.Unmarshal(msg.Data())
	if err != nil {
		return nil
	}
	schemaMap := make(map[string]schema.Object, len(resp.ObjectList))
	for _, line := range resp.ObjectList {
		pieces := strings.Split(line, " ")
		obj, err := schema.Parse(pieces[1])
		if err != nil {
			return nil
		}
		schemaMap[pieces[0]] = obj
	}
	return schemaMap
}

func formatDataForTopic(entry database.Entry) string {
	var output string
	schemaName := entry.Schema
	data := entry.Data

	switch {
	case schemaName == "string":
		output = string(data)
	case schemaName == "binary":
		output = fmt.Sprintf("binary(...%d bytes...)", len(data))
	case schemaName == "boolean":
		b := "true"
		if data[0] == 0 {
			b = "false"
		}
		output = fmt.Sprintf("boolean(%s)", b)
	case schemaName == "uint8":
		output = fmt.Sprintf("uint8(%d)", data[0])
	case schemaName == "uint16":
		output = fmt.Sprintf("uint16(%d)", binary.LittleEndian.Uint16(data))
	case schemaName == "uint32":
		output = fmt.Sprintf("uint32(%d)", binary.LittleEndian.Uint32(data))
	case schemaName == "uint64":
		output = fmt.Sprintf("uint64(%d)", binary.LittleEndian.Uint64(data))
	case schemaName == "int16":
		output = fmt.Sprintf("int16(%d)", int16(binary.LittleEndian.Uint16(data)))
	case schemaName == "int32":
		output = fmt.Sprintf("int32(%d)", int32(binary.LittleEndian.Uint32(data)))
	case schemaName == "int64":
		output = fmt.Sprintf("int64(%d)", int64(binary.LittleEndian.Uint64(data)))
	case schemaName == "float32":
		output = fmt.Sprintf("float32(%f)", float32(binary.LittleEndian.Uint32(data)))
	case schemaName == "float64":
		output = fmt.Sprintf("float64(%f)", float64(binary.LittleEndian.Uint64(data)))
	}

	return output
}

func filterStringSlice(s []string, prefix string) []string {
	retList := []string{}
	for i := range s {
		if strings.HasPrefix(s[i], prefix) {
			retList = append(retList, s[i])
		}
	}
	return retList
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
	appendItem := readline.PcItemDynamic(listTopics(c))
	completer := readline.NewPrefixCompleter(
		readline.PcItem("USE", useItem),
		readline.PcItem("use", useItem),
		readline.PcItem("APPEND", appendItem),
		readline.PcItem("append", appendItem),
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
			log.Fatal().Err(err).Msg("error sending message to server")
		}

		switch msg.Command() {
		case proto.CommandVersion:
			v := proto.VersionResponse{}
			err = v.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Printf("%d %s\n", v.Code, v.Version)
		case proto.CommandStats:
			t := proto.StatsResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Printf(
				"Allocated Heap: %v\nTotal Memory: %v\nUptime: %s\nSegments: %d\nTopics: %d\n",
				humanize.Bytes(t.AllocHeap),
				humanize.Bytes(t.TotalMem),
				t.Uptime,
				t.Segments,
				t.Topics,
			)
		case proto.CommandQuery:
			t := proto.QueryResponse{}
			err = t.Unmarshal(msg.Data())
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
					formatDataForTopic(t.Results[i]),
				})
			}

			table.Render()
		case proto.CommandError:
			t := proto.ErrResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Err)
		case proto.CommandOk:
			t := proto.OkResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Message)
		case proto.CommandAppend:
			t := proto.OkResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			fmt.Println(t.Code, t.Message)
		case proto.CommandList:
			t := proto.ListResponse{}
			err = t.Unmarshal(msg.Data())
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
