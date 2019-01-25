package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wallnutkraken/gotuskgo/controlpanel"
	"google.golang.org/grpc"
)

// TuskCLi is the CLi GoTuskGo gRPC client
// This application is used to control GoTuskGo via gRPC

var cliReader = bufio.NewReader(os.Stdin)

// Method represents a selectable gRPC method call
type Method struct {
	Name        string
	Function    func(client controlpanel.ControllerClient)
	Description string
}

// MethodList is an int-indexed list of callable methods
type MethodList struct {
	// We have to put it in this kind of indexed struct because, for some reason,
	// Go changes the sorting on these objects between runs
	Index   int
	Methods map[int]Method
}

// Next gets the next method. Returns the next method, the method's index and whether this method doesn't exist.
func (m *MethodList) Next() (Method, int, bool) {
	m.Index++
	method, exists := m.Methods[m.Index]
	return method, m.Index, exists
}

var methods = MethodList{
	Methods: map[int]Method{
		1: Method{
			Name:        "GetLogs",
			Function:    getErrors,
			Description: "Get all logs from the in-memory GoTuskGo Logger",
		},
		2: Method{
			Name:        "GetConfig",
			Function:    getSettings,
			Description: "Save the settings.json file for GoTuskGo to a local file",
		},
		3: Method{
			Name:        "SetConfig",
			Function:    setSettings,
			Description: "Replace the remote GoTuskGo settings.json with a local file",
		},
		4: Method{
			Name:        "AddToDatabase",
			Function:    addMessages,
			Description: "Adds a plaintext file to the GoTuskGo message list, separated by newlines",
		},
		5: Method{
			Name:        "GetDatabase",
			Function:    getDatabase,
			Description: "Get a current database backup file",
		},
		6: Method{
			Name:        "TriggerSendout",
			Function:    triggerSendout,
			Description: "Triggers a message sendout to all available channels",
		},
	},
}
var (
	authCode = flag.String("auth", "changeme", "The authentication code for calling the gRPC Control Panel")
	port     = flag.Int("port", 5025, "The gRPC port to call the server on")
	host     = flag.String("host", "wallnutkraken.com", "The gRPC server's hostname")
	timeout  = flag.Duration("timeout", time.Second*5, "The gRPC call timeout")
)

func main() {
	flag.Parse()

	fmt.Println("Opening connection")
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *host, *port), grpc.WithInsecure())
	if err != nil {
		errorExit(err)
	}
	defer conn.Close()

	client := controlpanel.NewControllerClient(conn)
	fmt.Println("Connection opened. Pick a method to call.")
	for {
		method, index, exists := methods.Next()
		if !exists {
			break
		}
		fmt.Printf("[%d] %s: %s\n", index, method.Name, method.Description)
	}
	choice, _, err := cliReader.ReadLine()
	if err != nil {
		errorExit(err)
	}
	// Turn the rune to an int
	choiceIndex, err := strconv.Atoi(string(choice))
	if err != nil {
		errorExit(err)
	}
	// Get the relevant method
	method, exists := methods.Methods[choiceIndex]
	if !exists {
		fmt.Println("Invalid selection")
		os.Exit(1)
	}
	method.Function(client)
}

func errorExit(err error) {
	fmt.Printf("ERROR: %s\n", err.Error())
	os.Exit(1)
}

func getErrors(client controlpanel.ControllerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}
	appErrors, err := client.GetApplicationErrors(ctx, auth)
	if err != nil {
		errorExit(err)
	}
	// List errors
	for _, appErr := range appErrors.Error {
		// Parse the time
		errorTime := time.Unix(appErr.Unix, 0)
		fmt.Printf("[%v] %s\n", errorTime, appErr.Error)
	}
}

func setSettings(client controlpanel.ControllerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}

	// Get the bytes of the settings file
	fmt.Print("New settings file path: ")
	line, _, err := cliReader.ReadLine()
	if err != nil {
		errorExit(err)
	}
	filePath := string(line)
	setting, err := ioutil.ReadFile(filePath)
	if err != nil {
		errorExit(err)
	}
	// Send it out
	_, err = client.SetConfig(ctx, &controlpanel.SetConfigParams{
		Auth: auth,
		Data: &controlpanel.SerializedData{
			Content: setting,
		},
	})
	if err != nil {
		errorExit(err)
	}

	// No Error, just print Done and exit
	fmt.Println("Done.")
}

func getSettings(client controlpanel.ControllerClient) {
	// Ask where to write the file
	fmt.Print("Filepath (including filename) on where to save the settings file: ")
	pathBytes, _, err := cliReader.ReadLine()
	if err != nil {
		errorExit(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}
	configSerialized, err := client.GetConfig(ctx, auth)
	if err != nil {
		errorExit(err)
	}

	// Write the data to file
	if err := ioutil.WriteFile(string(pathBytes), configSerialized.Content, os.ModePerm); err != nil {
		errorExit(err)
	}
	fmt.Printf("Written file in %s\n", string(pathBytes))
}

func addMessages(client controlpanel.ControllerClient) {
	fmt.Println("Messages file should just be a file with messages, separated by newlines")
	fmt.Println("Calling this endpoint with a lot of messages will disable GoTuskGo for a while")
	// Ask for the file
	fmt.Print("Messages filepath: ")
	pathBytes, _, err := cliReader.ReadLine()
	if err != nil {
		errorExit(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}

	// Read the file given
	file, err := ioutil.ReadFile(string(pathBytes))
	if err != nil {
		errorExit(err)
	}
	// Split it into lines
	lines := strings.Split(string(file), "\n")
	messages := &controlpanel.MessageList{
		Auth:    auth,
		Message: lines,
	}
	_, err = client.AddToDatabase(ctx, messages)
	if err != nil {
		errorExit(err)
	}
}

func getDatabase(client controlpanel.ControllerClient) {
	fmt.Println("Timeouts are disabled for this endpoint due to streaming")
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}

	// Ask for the file
	fmt.Print("Database backup filepath: ")
	pathBytes, _, err := cliReader.ReadLine()
	if err != nil {
		errorExit(err)
	}

	dbStream, err := client.GetDatabase(context.Background(), auth)
	if err != nil {
		errorExit(err)
	}

	// Open the destination file
	file, err := os.Create(string(pathBytes))
	if err != nil {
		errorExit(err)
	}
	defer file.Close()

	// Read the entire stream, write to file as we read
	for {
		// Get the next chunk
		chunk, err := dbStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorExit(err)
		}
		// Write the chunk to file
		bytesWritten, err := file.Write(chunk.Content)
		if err != nil {
			errorExit(err)
		}
		fmt.Printf("Got a [%d] byte chunk\n", bytesWritten)
	}
}

func triggerSendout(client controlpanel.ControllerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	auth := &controlpanel.AuthCode{
		Code: *authCode,
	}
	_, err := client.TriggerSendout(ctx, auth)
	if err != nil {
		errorExit(err)
	}
	fmt.Println("Done.")
}
