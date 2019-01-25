// Package settings contains JSON-Encodeable settings for the application
package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// Path is the only path the settings file should be in
const Path = "opdata/settings.json"

// Default is the default application settings
var Default = Application{
	Brain: Brain{
		SplitChars:         "-.,?!/\\\r \n\t",
		MaxGeneratedLength: 30,
		ChainLength:        1,
	},
	GRPC: GRPC{
		AuthCode: "changeme",
		Port:     5025,
	},
	Telegram: Telegram{
		APIKey:  "",
		Verbose: false,
	},
	Database: Database{
		Path: "opdata/gotuskgo.db",
	},
	Messaging: Messaging{
		NormalMinMinutes: 15,
		NormalMaxMinutes: 60,
		SleepMinMinutes:  180,
		SleepMaxMinutes:  200,
	},
}

// Application contains all the setting categories
type Application struct {
	Brain     Brain     `json:"brain"`
	GRPC      GRPC      `json:"grpc"`
	Telegram  Telegram  `json:"telegram"`
	Database  Database  `json:"database"`
	Messaging Messaging `json:"messaging"`
}

// Brain contains the settings for the markov brain
type Brain struct {
	SplitChars         string `json:"split_chars"`
	MaxGeneratedLength int    `json:"max_generated_length"`
	ChainLength        int    `json:"chain_length"`
}

// GRPC contains the GRPC settings
type GRPC struct {
	AuthCode string `json:"auth_code"`
	Port     int    `json:"port"`
}

// GetPort returns the port in the format :n
// Where n is the port
func (g GRPC) GetPort() string {
	return fmt.Sprintf(":%d", g.Port)
}

// Telegram contains the settings for Telegram
type Telegram struct {
	APIKey  string `json:"api_key"`
	Verbose bool   `json:"verbose"`
}

// Database contains the settings for the SQLite database
type Database struct {
	Path string `json:"path"`
}

// Messaging contains the settings related to messaging (e.g. min-max minutes between sendouts)
type Messaging struct {
	NormalMinMinutes int `json:"normal_min"`
	NormalMaxMinutes int `json:"normal_max"`
	SleepMinMinutes  int `json:"sleep_min"`
	SleepMaxMinutes  int `json:"sleep_max"`
}

// Load loads all the settings from the filepath
func Load() (Application, error) {
	// Open the config file
	file, err := os.Open(Path)
	if err != nil {
		return Application{}, errors.Wrap(err, "os.Open")
	}
	defer file.Close()
	// Read the entire file
	settingsBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return Application{}, errors.Wrap(err, "ioutil.ReadAll")
	}
	// Unmarshal the JSON
	sett := Application{}
	if err := json.Unmarshal(settingsBytes, &sett); err != nil {
		return Application{}, errors.Wrap(err, "json")
	}

	return sett, nil
}

// Save writes the given settings to file
func Save(setting Application) error {
	// Open the config file
	file, err := os.Create(Path)
	if err != nil {
		return errors.Wrap(err, "os.Open")
	}
	defer file.Close()
	// Marshal the settings
	settingsBytes, err := json.Marshal(setting)
	if err != nil {
		return errors.Wrap(err, "json")
	}
	// And write the bytes
	_, err = file.Write(settingsBytes)
	return err
}
