/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package test

import (
	"fmt"
	"time"

	fossil "github.com/dburkart/fossil/api"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = &cobra.Command{
	Use:   "test",
	Short: "Send a series of test messages to the server",

	Run: func(cmd *cobra.Command, args []string) {
		log := viper.Get("logger").(zerolog.Logger)

		host := viper.GetString("fossil.host")
		target := proto.ParseConnectionString(host)

		client, err := fossil.NewClient(host)
		if err != nil {
			log.Error().Err(err).Str("address", target.Address).Msg("unable to connect to server")
		}

		// test
		timeIt("RandomCountBytesTest", client, RandomCountBytesTest)
	},
}

func init() {
	// Flags for this command
	Command.Flags().Int("count", 10, "Number of messages to send")

	// Bind flags to viper
	viper.BindPFlag("count", Command.Flags().Lookup("count"))
}

func timeIt(name string, client fossil.Client, f func(client fossil.Client)) {
	t := time.Now()
	defer func() {
		log.Info().Str("dur", time.Since(t).String()).Str("name", name).Send()
	}()
	f(client)
}

func RandomCountBytesTest(client fossil.Client) {
	count := viper.GetInt("count")
	for i := 0; i < count; i++ {
		client.Send(proto.NewMessageWithType(
			proto.CommandAppend,
			proto.AppendRequest{
				Topic: fmt.Sprintf("/test/%d/%d", count, i),
				Data:  []byte{},
			},
		))
	}
}
