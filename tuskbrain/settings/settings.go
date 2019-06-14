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
		UseRNN:             false,
	},
	GRPC: GRPC{
		AuthCode: "changeme",
		Port:     5025,
	},
	APIs: APIs{
		Telegram: "",
		Discord:  "",
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
	RNN: RNN{
		SavePath:                "opdata/py-rnn.bin",
		EpochsPerTraining:       30,
		Temperature:             0.2,
		MaxGenerationCharacters: 80,
	},
}

// Application contains all the setting categories
type Application struct {
	Brain     Brain     `json:"brain"`
	GRPC      GRPC      `json:"grpc"`
	APIs      APIs      `json:"api_keys"`
	Database  Database  `json:"database"`
	Messaging Messaging `json:"messaging"`
	RNN       RNN       `json:"rnn"`
}

// Brain contains the settings for the markov brain
type Brain struct {
	SplitChars         string `json:"split_chars"`
	MaxGeneratedLength int    `json:"max_generated_length"`
	ChainLength        int    `json:"chain_length"`
	UseRNN             bool   `json:"use_neuralnet"`
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

// APIs contains the API Keys for all available services
type APIs struct {
	Telegram string `json:"telegram"`
	Discord  string `json:"discord"`
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

// RNN contains the settings for the python RNN
type RNN struct {
	SavePath                string  `json:"save_path"`
	EpochsPerTraining       int     `json:"epochs_per_training"`
	Temperature             float64 `json:"temperature"`
	MaxGenerationCharacters int     `json:"max_gen_chars"`
	// TrainCooldownMins is the minute amount of time inbetween training
	TrainMinsPeriod int `json:"training_mins_period"`
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

	// Go through every section, if it's not set, give it default settings
	if sett.Brain == (Brain{}) {
		sett.Brain = Default.Brain
	}
	if sett.Database == (Database{}) {
		sett.Database = Default.Database
	}
	if sett.GRPC == (GRPC{}) {
		sett.GRPC = Default.GRPC
	}
	if sett.Messaging == (Messaging{}) {
		sett.Messaging = Default.Messaging
	}
	if sett.APIs == (APIs{}) {
		sett.APIs = Default.APIs
	}
	if sett.RNN == (RNN{}) {
		sett.RNN = Default.RNN
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
