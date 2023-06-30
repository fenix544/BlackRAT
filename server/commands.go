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
	Func           func(args []string)
}

var commands []Command

func (command Command) Execute(args []string) {
	if command.RequiredClient && (selectedClient.Conn == nil || !selectedClient.Connected) {
		log.Errorf("No client selected!")
		return
	}
	command.Func(args)
}

func RegisterCommand(name, description string, args []string, requiredClient bool, function func(args []string)) {
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
	command.Execute(args)
}

func CommandLine() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		split := strings.Split(scanner.Text(), " ")

		commandName := split[0]
		commandArgs := split[1:]

		ParseCommand(commandName, commandArgs)
	}
}

func RegisterCommands() {
	RegisterCommand("help", "Prints help", []string{}, false, func(args []string) {
		log.Infof("Available commands:")
		for _, command := range commands {
			if len(command.Args) == 0 {
				log.Infof("%s - %s", command.Name, command.Description)
			} else {
				log.Infof("%s [%s] - %s", command.Name, strings.Join(command.Args, " "), command.Description)
			}
		}
	})
	RegisterCommand("exit", "Exits the program", []string{}, false, func(args []string) {
		log.Info("Exiting...")
		os.Exit(0)
	})
	RegisterCommand("cmd", "Executes a command", []string{"command"}, true, func(args []string) {
		if len(args) == 0 {
			log.Errorf("No command provided")
			return
		}

		selectedClient.SendData("cmd " + strings.Join(args, " "))
	})
	RegisterCommand("connections", "Lists all connections", []string{}, true, func(args []string) {
		if len(connections) == 0 {
			log.Infof("No connections")
			return
		}

		index := 0
		for _, connection := range connections {
			var status string
			if connection.Connected {
				status = "connected"
			} else {
				status = "disconnected"
			}

			log.Infof("[%s] Connection: %s | Status: %s", strconv.Itoa(index), connection.Conn.RemoteAddr(), status)
			index++
		}
	})
	RegisterCommand("files", "Lists all files in the current directory", []string{"directory"}, true, func(args []string) {
		if len(args) != 1 {
			log.Errorf("No directory provided")
			return
		}

		selectedClient.SendData("files " + args[0])
	})
	RegisterCommand("download", "Downloads a file", []string{"file"}, true, func(args []string) {
		if len(args) != 1 {
			log.Errorf("No file provided")
			return
		}

		selectedClient.SendData("download " + args[0])
	})
	RegisterCommand("select", "Selects a connection", []string{"connection"}, false, func(args []string) {
		if len(args) != 1 {
			log.Errorf("No connection provided")
			return
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			log.Errorf("Invalid connection index")
			return
		}

		if index >= len(connections) {
			log.Errorf("Invalid connection index")
			return
		}

		i := 0
		for s := range connections {
			if i == index {
				if connections[s].Connected == false {
					log.Errorf("Connection is not connected")
					return
				}

				selectedClient = connections[s]
				log.Infof("Selected connection %s", selectedClient.Conn.RemoteAddr())
				return
			}
			i++
		}
	})
}
