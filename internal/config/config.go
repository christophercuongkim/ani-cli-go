package config

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
)

type (
	Config struct {
		General   general
		Downloads downloads
		History   history
		Anilist   anilist
		Mal       mal
		Syncplay  syncplay
		Tui       tui
	}

	general struct {
		Player        string
		Quality       string
		Mode          string
		Skip_intro    bool
		Detach_player bool
	}

	downloads struct {
		Directory            string
		Concurrent_fragments int
	}

	history struct {
		Db_path string
	}

	anilist struct {
		Enabled bool
		Token   string
	}

	mal struct {
		Enabled   bool
		Client_id string
	}

	syncplay struct {
		Server   string
		Room     string
		Username string
	}

	tui struct {
		Theme string
	}
)

const (
	defaultConfigLocation    = "$XDG_CONFIG_HOME/ani-cli-go/configuration.toml"
	ownerFullOthersReadExec  = 0755
	ownerReadWriteOthersRead = 0644
)

//go:embed template/config.example.toml
var exampleConfigFile []byte

func LoadConfig(filename string) (*Config, error) {

	/*
		We want to create a default XDG_CONFIG_HOME folder and config file
		for ani-cli-go if the user doesn't supply a filepath.

		Should be able to expand env vars.
	*/
	configFileLocation := os.ExpandEnv(filename)

	/*
		We don't want to fail out if the user wants us to create the
		folder path for them.
	*/
	folderPath := filepath.Dir(configFileLocation)
	err := os.MkdirAll(folderPath, ownerFullOthersReadExec)
	if err != nil {
		log.Fatal().Err(err).Str("folderPath", folderPath).Msg("Unable to make directories")
	}

	/*
		We will create a default config for the user if the file doesn't exist.
	*/
	if _, err := os.Stat(configFileLocation); os.IsNotExist(err) {
		err := os.WriteFile(configFileLocation, exampleConfigFile, ownerReadWriteOthersRead)
		if err != nil {
			log.Fatal().Err(err).Str("location", configFileLocation).Msg("Failed to create config file")
		}
		log.Info().Str("location", configFileLocation).Msg("Config file created")
	}

	var config Config

	if _, err := toml.DecodeFile(configFileLocation, &config); err != nil {
		return nil, err
	}

	return &config, nil

}
