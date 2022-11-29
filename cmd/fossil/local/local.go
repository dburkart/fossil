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
	},
}

func init() {
	// Flags for this command
	Command.Flags().StringP("database", "d", "./", "Path to the database")

	// Bind flags to viper
	viper.BindPFlag("database", Command.Flags().Lookup("database"))
}
