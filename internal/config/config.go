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
		General   general   `toml:"general"`
		Downloads downloads `toml:"downloads"`
		History   history   `toml:"history"`
		Anilist   anilist   `toml:"anilist"`
		Mal       mal       `toml:"mal"`
		Syncplay  syncplay  `toml:"syncplay"`
		Tui       tui       `toml:"tui"`
	}

	general struct {
		Player       string `toml:"player"`
		Quality      string `toml:"quality"`
		Mode         string `toml:"mode"`
		SkipIntro    bool   `toml:"skip_intro"`
		DetachPlayer bool   `toml:"detach_player"`
	}

	downloads struct {
		Directory           string `toml:"directory"`
		ConcurrentFragments int    `toml:"concurrent_fragments"`
	}

	history struct {
		DbPath string `toml:"db_path"`
	}

	anilist struct {
		Enabled bool   `toml:"enabled"`
		Token   string `toml:"token"`
	}

	mal struct {
		Enabled  bool   `toml:"enabled"`
		ClientId string `toml:"client_id"`
	}

	syncplay struct {
		Server   string `toml:"server"`
		Room     string `toml:"room"`
		Username string `toml:"username"`
	}

	tui struct {
		Theme string `toml:"theme"`
	}
)

const (
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

	if os.Getenv("XDG_CONFIG_HOME") == "" {
		home, _ := os.UserHomeDir()
		os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	}
	configFileLocation := os.ExpandEnv(filename)

	/*
		We don't want to fail out if the user wants us to create the
		folder path for them.
	*/
	folderPath := filepath.Dir(configFileLocation)
	err := os.MkdirAll(folderPath, ownerFullOthersReadExec)
	if err != nil {
		log.Error().Err(err).Str("folderPath", folderPath).Msg("Unable to make directories")
		return nil, err
	}

	/*
		We will create a default config for the user if the file doesn't exist.
	*/
	if _, err := os.Stat(configFileLocation); os.IsNotExist(err) {
		err := os.WriteFile(configFileLocation, exampleConfigFile, ownerReadWriteOthersRead)
		if err != nil {
			log.Error().Err(err).Str("location", configFileLocation).Msg("Failed to create config file")
			return nil, err
		}
		log.Info().Str("location", configFileLocation).Msg("Config file created")
	}

	var config Config

	if _, err := toml.DecodeFile(configFileLocation, &config); err != nil {
		log.Error().Err(err).Msg("Failed to decode file.")
		return nil, err
	}

	return &config, nil

}
