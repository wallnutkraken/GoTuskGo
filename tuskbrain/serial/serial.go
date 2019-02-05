// Package serial contains the functions to serialize/deserialize GoTuskGo message caches
package serial

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
)

// Marshal seralizes a given message cache in a gzipped format
func Marshal(messages []dbwrap.Message) ([]byte, error) {
	if len(messages) == 0 {
		return []byte{}, nil
	}
	// Turn the message array into a string with messages on every line
	// But as a byte array
	uncompressed := []byte{}
	for _, msg := range messages {
		msgBytes := []byte(msg.Content + "\n")
		uncompressed = append(uncompressed, msgBytes...)
	}
	// Remove the last newline
	uncompressed = uncompressed[:len(uncompressed)-1]

	// Create the file header
	var outputBuffer bytes.Buffer
	zw := gzip.NewWriter(&outputBuffer)
	defer zw.Close()
	zw.Name = "GoTuskGoDatabaseDump.txt"
	zw.ModTime = time.Now().UTC()

	// Write the content
	_, err := zw.Write(uncompressed)
	if err != nil {
		return nil, errors.Wrap(err, "gzip")
	}

	// And read the written buffer
	output := make([]byte, outputBuffer.Len())
	_, err = outputBuffer.Read(output)
	if err != nil {
		return nil, errors.Wrap(err, "outputBuffer")
	}
	return output, nil
}

// Unmarshal deserializes a previously serialized database backup
func Unmarshal(v []byte) ([]string, error) {
	// Create a deserialization buffer
	inputBuffer := bytes.NewBuffer(v)
	reader, err := gzip.NewReader(inputBuffer)
	if err != nil {
		return nil, errors.Wrap(err, "gzip.NewReader")
	}
	defer reader.Close()
	// Read all lines
	allLines, err := ioutil.ReadAll(reader)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "iotuil")
	}
	// Turn it into a string, split by newline
	return strings.Split(string(allLines), "\n"), nil
}

// LogLine is a log entry, containing both the message (usually errors)
// as well as the Unix time stamp
type LogLine struct {
	Message string
	UNIX    int64
}
