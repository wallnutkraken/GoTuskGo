// Package rnn contains the Go wrapper for the Python package textgenrnn, for use in high-level GoTuskGo functions
package rnn

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/memlog"
	"github.com/wallnutkraken/gotuskgo/stringer"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

const (
	pyTrainScript    = "./python/train.py"
	pyGenerateScript = "./python/generate.py"
)

var (
	// ErrAlreadyTraining is an error for when the network is already in the middle of training
	ErrAlreadyTraining = errors.New("The Neural Network is already training")
)

// Network contains the necessary data and functions to interact with
// the Python RNN
type Network struct {
	rnnSettings settings.RNN
	lock        *sync.Mutex
	isTraining  bool
	buffer      *Buffer
	log         *memlog.Child
}

// Buffer contains buffered messages
type Buffer struct {
	rnnSettings settings.RNN
	messages    []string
	size        int
	lock        *sync.Mutex
}

// PopN pops the last n entries
func (b *Buffer) PopN(n int) []string {
	if len(b.messages) <= n {
		n = len(b.messages) - 1
	}

	b.lock.Lock()
	res := b.messages[:n]
	b.messages = b.messages[n:]
	b.lock.Unlock()

	return res
}

// Repopulate the buffer
func (b *Buffer) Repopulate() error {
	if b.size <= len(b.messages) {
		return nil
	}

	diff := b.size - len(b.messages)
	generated, err := generate(b.rnnSettings.SavePath, b.rnnSettings.Temperature, diff, b.rnnSettings.MaxGenerationCharacters)
	if err != nil {
		return err
	}
	lines := stringer.SplitMultiple(string(generated), "\n") // Use SplitMultiple here to ignore empty lines

	b.lock.Lock()
	b.messages = append(b.messages, lines...)
	b.lock.Unlock()
	return nil
}

// RepopulationService runs the repopulation service
func (n *Network) RepopulationService() {
	for {
		if err := n.buffer.Repopulate(); err != nil {
			n.log.ErrorMessage(err, "Failed repopulating buffer")
		}
		time.Sleep(time.Second * 5)
	}
}

// New creates a new instance of the Network
func New(config settings.RNN, log *memlog.Child) *Network {
	net := &Network{
		rnnSettings: config,
		lock:        &sync.Mutex{},
		isTraining:  false,
		log:         log,
		buffer: &Buffer{
			messages:    []string{},
			size:        100,
			lock:        &sync.Mutex{},
			rnnSettings: config,
		},
	}
	go net.RepopulationService()

	return net
}

// UpdateSettings updates the settings the Network uses
func (n *Network) UpdateSettings(newSettings settings.RNN) {
	if newSettings != n.rnnSettings {
		n.rnnSettings = newSettings
		n.buffer.rnnSettings = newSettings
	}
}

// setIsTrainings sets the isTraining bool, secured with mutex
func (n *Network) setIsTraining(new bool) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.isTraining = new
}

// IsTraining returns a boolean determining whether the network is currently being trained
func (n *Network) IsTraining() bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.isTraining
}

// Train trains the neural network with the data given, and will block until the training is complete.
// At which point, it will save the network weights to the path specified in settings.RNN
func (n *Network) Train(data []string) error {
	if n.IsTraining() {
		// It's already training, try again later
		return ErrAlreadyTraining
	}
	// Save the data to a temp path
	path := "/tmp/" + strconv.Itoa(rand.Int())
	// Turn the data into a byte array
	dataBytes := []byte(strings.Join(data, "\n"))
	if err := ioutil.WriteFile(path, dataBytes, os.ModeTemporary); err != nil {
		return errors.WithMessagef(err, "Failed writing temporary file %s", path)
	}
	defer os.Remove(path)

	// Empty dataBytes, then run train
	dataBytes = []byte{}
	return train(path, n.rnnSettings.EpochsPerTraining, n.rnnSettings.SavePath)
}

// train runs the python train.py file, used to interface with textgenrnn
func train(trainDataSetPath string, numEpochs int, savePath string) error {
	cmd := exec.Command("python3", pyTrainScript, trainDataSetPath, strconv.Itoa(numEpochs), savePath)
	errOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error training:\n\n %s", string(errOutput))
	}
	return nil
}

// Generate generates a new string from the neural network
func (n *Network) Generate() (string, error) {
	genBytes, err := generate(n.rnnSettings.SavePath, n.rnnSettings.Temperature, 1, n.rnnSettings.MaxGenerationCharacters)
	if err != nil {
		return "", errors.Wrap(err, "generate")
	}
	return string(genBytes), nil
}

// GenerateN generates amt amount of lines from the neural network
func (n *Network) GenerateN(amt int) ([]string, error) {
	// Try the buffer first
	bufferedLines := n.buffer.PopN(amt)
	if len(bufferedLines) == amt {
		return bufferedLines, nil
	}

	// Fall back to calling generate
	genBytes, err := generate(n.rnnSettings.SavePath, n.rnnSettings.Temperature, amt, n.rnnSettings.MaxGenerationCharacters)
	if err != nil {
		return nil, errors.Wrap(err, "generate")
	}
	// Separate the lines
	lines := stringer.SplitMultiple(string(genBytes), "\n") // Use SplitMultiple here to ignore empty lines
	return lines, nil
}

// geberate rybs the python generate.py file, used to generate text from a trained RNN
func generate(loadPath string, temperature float64, amt int, maxChars int) ([]byte, error) {
	cmd := exec.Command("python3", pyGenerateScript, loadPath, fmt.Sprintf("%f", temperature), strconv.Itoa(amt), strconv.Itoa(maxChars))
	var buf bytes.Buffer
	cmd.Stderr = &buf
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New(buf.String())
	}
	return output, nil
}
