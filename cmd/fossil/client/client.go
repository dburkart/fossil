/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package client

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/schema"

	"github.com/chzyer/readline"
	fossil "github.com/dburkart/fossil/api"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/repl"
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
			output := viper.GetString("fossil.output")
			if len(filterStringSlice([]string{"csv", "text", "json"}, output)) != 1 {
				log.Fatal().Msg("unsupported output format")
			}

			host := viper.GetString("fossil.host")
			target, err := proto.ParseConnectionString(host)
			if err != nil {
				log.Fatal().Err(err).Msg("error parsing URL")
			}

			client, err := fossil.NewClient(host)
			if err != nil {
				log.Error().Err(err).Str("address", target.Address).Msg("unable to connect to server")
			}

			readlinePrompt(client, output)
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

	// Flags for this command
	Command.Flags().StringP("output", "o", "text", "Output format of results in pipe mode [csv, json, text]")

	// Bind flags to viper
	viper.BindPFlag("fossil.output", Command.Flags().Lookup("output"))
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
		if strings.HasPrefix(line, "use") {
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
		if strings.HasPrefix(line, "append") {
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

func completeCreateTopic(c fossil.Client) func(string) []string {
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
		fields := strings.Fields(line)
		if len(fields) <= 2 {
			return nil
		}
		lineTopic := strings.TrimPrefix(line, fields[0])
		lineTopic = strings.TrimPrefix(lineTopic[1:], fields[1])
		lineTopic = strings.TrimPrefix(lineTopic[1:], fields[2])
		lineTopic = strings.TrimPrefix(lineTopic, " ")

		options := filterStringSlice(resp.ObjectList, lineTopic)
		if len(fields[2]) > 0 {
			options = append(options, fields[2])
		}

		return options
	}
}

func makeSchemaOptions() []readline.PrefixCompleterInterface {
	schemaSlice := []string{
		"string",
		"binary",
		"boolean",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"int16",
		"int32",
		"int64",
		"float32",
		"float64",
	}

	ret := []readline.PrefixCompleterInterface{}
	for i := range schemaSlice {
		ret = append(ret, readline.PcItem(schemaSlice[i]))
	}
	return ret
}

func readlinePrompt(c fossil.Client, output string) {
	// Configure the completer
	useItem := readline.PcItemDynamic(listDatabases(c))
	appendItem := readline.PcItemDynamic(listTopics(c))

	listItems := []readline.PrefixCompleterInterface{
		readline.PcItem("topics"), readline.PcItem("databases"), readline.PcItem("schemas"),
	}

	completer := readline.NewPrefixCompleter(
		readline.PcItem("help", useItem),
		readline.PcItem("use", useItem),
		readline.PcItem("append", appendItem),
		readline.PcItem("insert"),
		readline.PcItem("query"),
		readline.PcItem("exit"),
		readline.PcItem("list", listItems...),
		readline.PcItem("create",
			readline.PcItem("topic", readline.PcItemDynamic(completeCreateTopic(c), makeSchemaOptions()...)),
		),
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

	schemas := listSchemas(c)
	recomputeSchemaCache := false

	// Configure output writer
	writer := repl.NewOutputWriter(os.Stdout, output)

	// Handle input
	for {
		ln := rl.Line()
		if ln.CanContinue() {
			continue
		} else if ln.CanBreak() {
			break
		}
		line := strings.TrimSpace(ln.Line)

		if strings.ToUpper(line) == "HELP" {
			fmt.Println("usage:")
			fmt.Println(completer.Tree("    "))
			continue
		}
		if strings.ToUpper(line) == "EXIT" {
			os.Exit(0)
		}

		replMsg, err := repl.ParseREPLCommand([]byte(line), schemas)
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}

		msg, err := c.Send(replMsg)
		if err != nil {
			log.Fatal().Err(err).Msg("error sending message to server")
		}

		// FIXME: This is quite the hack. We need a better heuristic to invalidate our schema cache
		//		  than just looking at the command type we sent over the wire. It would be better if
		//		  we could reach into the message and examine the topic we're appending to or creating
		if replMsg.Command() == proto.CommandAppend || replMsg.Command() == proto.CommandCreate {
			recomputeSchemaCache = true
		}

		switch msg.Command() {
		case proto.CommandVersion:
			v := proto.VersionResponse{}
			err = v.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			writer.Write(v)
		case proto.CommandStats:
			t := proto.StatsResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}

			writer.Write(t)
		case proto.CommandQuery:
			t := proto.QueryResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}

			writer.Write(t)
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
			writer.Write(t)
		case proto.CommandAppend:
			t := proto.OkResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			writer.Write(t)
		case proto.CommandList:
			t := proto.ListResponse{}
			err = t.Unmarshal(msg.Data())
			if err != nil {
				log.Error().Err(err).Send()
				continue
			}
			writer.Write(t)
		}
		fmt.Println()

		if recomputeSchemaCache {
			schemas = listSchemas(c)
			recomputeSchemaCache = false
		}
	}
	rl.Clean()
}
