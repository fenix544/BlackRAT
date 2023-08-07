package main

import (
	"fmt"
	"github.com/charmbracelet/log"
	"net"
	"os"
	"strconv"
	"strings"
)

type Client struct {
	Conn      net.Conn
	Connected bool
	Addr      string
}

func (client *Client) SendData(output string) {
	_, _ = client.Conn.Write([]byte(output))
}

var config *Config
var connections = make(map[string]Client)
var selectedClient Client

func main() {
	Run()
}

func StartServer(host string, port int, runnable func()) {
	listener, err := net.Listen("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		log.Errorf("Failed to start the server: %v", err)
	}

	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Errorf("Failed to close the server: %v", err)
		}
	}(listener)

	runnable()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}

		go HandleConnection(conn)
	}
}

func LoadConfig() (*Config, error) {
	file, err := ReadConfigFile(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	parsedConfig, err := ParseConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return parsedConfig, nil
}

func Run() {
	loadedConfig, err := LoadConfig()
	if err != nil {
		log.Errorf("Failed to load config: %v", err)
		return
	}

	config = loadedConfig

	go StartServer(config.Host, config.Port, func() {
		log.Info("Server started on " + config.Host + ":" + strconv.Itoa(config.Port))
		log.Info("Type /help for commands.")
		log.Info("Waiting for connections...")
	})

	RegisterCommands()
	CommandLine()
}

func HandleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}(conn)

	addr := FormatAddress(conn.RemoteAddr().String())
	client := Client{Conn: conn, Connected: true, Addr: addr}

	connections[addr] = client
	log.Infof("New connection [%s]", addr)

	buffer := make([]byte, config.BufferSize)
	for {
		bytesRead, err := conn.Read(buffer)

		if err != nil {
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				client.Connected = false
				log.Warnf("Connection closed [%s]", client.Addr)
				break
			}
			log.Errorf("Error reading from connection: %v", err)
			break
		}

		data := buffer[:bytesRead]
		ParseResponse(string(data))
	}

	client.Connected = false
	connections[addr] = client
}

func ParseResponse(response string) {
	fmt.Println()
	log.Info("Response from " + selectedClient.Addr)
	for _, s := range strings.Split(response, "\n") {
		log.Infof(s)
	}
	fmt.Println()
}
