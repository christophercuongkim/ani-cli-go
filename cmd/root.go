package cmd

import (
	"os"

	"github.com/christophercuongkim/ani-cli-go/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigLocation = "$XDG_CONFIG_HOME/ani-cli-go/configuration.toml"
)

var cfgFileName string

var rootCmd = &cobra.Command{
	Use:   "ani-cli-go",
	Short: "Go implementation of ani-cli.",
	Long:  "TODO",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("not yet implemented.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("rootCmd failed")
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFileName, "config", defaultConfigLocation, "path to config toml.")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	viper.SetEnvPrefix("ANICLI")
	viper.AutomaticEnv()
}

func initConfig() {
	configLoad, err := config.LoadConfig(viper.GetString("config"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed config load.")
	}

	log.Info().Str("Player", configLoad.General.Player).Msg("Config Loaded")

}
