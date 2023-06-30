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
}

func (client *Client) SendData(output string) {
	_, _ = client.Conn.Write([]byte(output))
}

var connections = make(map[string]Client)
var selectedClient Client

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

func main() {
	file, err := ReadConfigFile(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Errorf("Failed to read config file: %v", err)
		return
	}

	config, err := ParseConfig(file)
	if err != nil {
		log.Errorf("Failed to parse config file: %v", err)
		return
	}

	go StartServer(config.Host, config.Port, func() {
		log.Info("Server started on " + config.Host + ":" + strconv.Itoa(config.Port))
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

		client := connections[conn.RemoteAddr().String()]
		client.Connected = false

		connections[conn.RemoteAddr().String()] = client
	}(conn)

	log.Infof("New connection (%s)", conn.RemoteAddr().String())
	connections[conn.RemoteAddr().String()] = Client{Conn: conn, Connected: true}

	buffer := make([]byte, 1024)
	for {
		bytesRead, err := conn.Read(buffer)

		if err != nil {
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				log.Warnf("Connection closed (%s)", conn.RemoteAddr().String())

				client := connections[conn.RemoteAddr().String()]
				client.Connected = false

				connections[conn.RemoteAddr().String()] = client
				break
			}
			log.Errorf("Error reading from connection: %v", err)
			break
		}

		data := buffer[:bytesRead]
		ParseResponse(string(data))
	}
}

func ParseResponse(response string) {
	fmt.Println(response)
}
