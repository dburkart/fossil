package fossil

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/dburkart/fossil/cmd/fossil/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "fossil",
	Short: "Fossil is a small and fast tsdb",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
		// TODO: Trace log config and options
	},
}

func init() {
	// Configure the common binary options
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().CountP("verbose", "v", "-v for debug logs (-vv for trace)")
	rootCmd.PersistentFlags().Bool("local", true, "Configures the logger to print readable logs") //TODO: true until we have a config file format

	// Bind viper config to the root flags
	viper.BindPFlag("local", rootCmd.PersistentFlags().Lookup("local"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Bind viper flags to ENV variables
	viper.SetEnvPrefix("FOSSIL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Register commands on the root binary command
	rootCmd.AddCommand(server.Command)
}

func initConfig() {
	// config Read
}

func initLogging() {
	level := viper.GetInt("verbose")
	switch clamp(2, level) {
	case 2:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case 1:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	var writer io.Writer

	writer = os.Stderr
	if viper.GetBool("local") {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Caller().
		Logger()

	viper.Set("logger", logger)
}

func clamp(clamp, a int) int {
	if a >= clamp {
		return clamp
	}
	return a
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("root command failed")
		os.Exit(1)
	}
}
