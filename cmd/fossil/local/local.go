/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package local

import (
	"bufio"
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var Command = &cobra.Command{
	Use:   "local",
	Short: "Interact with a local Fossil database",

	Run: func(cmd *cobra.Command, args []string) {
		log := viper.Get("logger").(zerolog.Logger)
		db := database.NewDatabase(viper.GetString("database"))

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal().Err(err)
			}

			var results []database.Datum = nil
			stmt := query.Prepare(db, line)

			for i := len(stmt) - 1; i >= 0; i-- {
				results = stmt[i](results)
			}

			for _, val := range results {
				fmt.Println(val.ToString())
			}
		}
	},
}

func init() {
	// Flags for this command
	Command.Flags().StringP("database", "d", "./", "Path to the database")

	// Bind flags to viper
	viper.BindPFlag("database", Command.Flags().Lookup("database"))
}
