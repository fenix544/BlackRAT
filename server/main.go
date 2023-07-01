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

func InitializeAndRun() {
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

func main() {
	InitializeAndRun()
}

func HandleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}

		client := connections[FormatAddress(conn.RemoteAddr().String())]
		client.Connected = false

		connections[FormatAddress(conn.RemoteAddr().String())] = client
	}(conn)

	client := Client{Conn: conn, Connected: true, Addr: FormatAddress(conn.RemoteAddr().String())}
	connections[FormatAddress(conn.RemoteAddr().String())] = client

	log.Infof("New connection [%s]", client.Addr)

	buffer := make([]byte, 1024)
	for {
		bytesRead, err := conn.Read(buffer)

		if err != nil {
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				client := connections[FormatAddress(conn.RemoteAddr().String())]
				client.Connected = false

				log.Warnf("Connection closed [%s]", client.Addr)

				connections[FormatAddress(conn.RemoteAddr().String())] = client
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
	fmt.Println()
	log.Info("Response from " + selectedClient.Conn.RemoteAddr().String())
	for _, s := range strings.Split(response, "\n") {
		log.Infof(s)
	}
	fmt.Println()
}
