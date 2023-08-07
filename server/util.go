package main

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/charmbracelet/log"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// SetTitle https://github.com/lxi1400/GoTitle/blob/main/title.go
func SetTitle(title string) (int, error) {
	handle, err := syscall.LoadLibrary("Kernel32.dll")
	if err != nil {
		return 0, err
	}
	defer func(handle syscall.Handle) {
		_ = syscall.FreeLibrary(handle)
	}(handle)
	proc, err := syscall.GetProcAddress(handle, "SetConsoleTitleW")
	if err != nil {
		return 0, err
	}
	r, _, err := syscall.Syscall(proc, 1, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))), 0, 0)
	return int(r), err
}

// https://stackoverflow.com/questions/22891644/how-can-i-clear-the-terminal-screen-in-go
var clear map[string]func() //create a map for storing clear funcs

func init() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	}
}

func IsUrl(input string) bool {
	return strings.HasPrefix(input, "https") || strings.HasPrefix(input, "http")
}

func MakeRequest(url string) io.ReadCloser {
	// Make the GET request
	response, err := http.Get(url)
	if err != nil {
		log.Errorf("Error making GET request: %s", err)
		return nil
	}

	// Check the status code of the response
	if response.StatusCode != http.StatusOK {
		log.Errorf("Unexpected status code: %d", response.StatusCode)
		return nil
	}

	return response.Body
}

func ExtractDownloadLink(body io.ReadCloser) string {
	// Parse the HTML
	doc, err := html.Parse(body)
	if err != nil {
		log.Errorf("Error parsing HTML: %s", err)
		return ""
	}

	// Find and extract the href attribute of the <a> tag with id "download-url"
	var extractHref func(*html.Node) string
	extractHref = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "a" {
			// Check if the <a> tag has an "id" attribute with value "download-url"
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "download-url" {
					// Get the value of the "href" attribute
					for _, a := range n.Attr {
						if a.Key == "href" {
							return a.Val
						}
					}
				}
			}
		}

		// Recursively search for the <a> tag with id "download-url"
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			href := extractHref(c)
			if href != "" {
				return href
			}
		}

		return ""
	}

	return extractHref(doc)
}

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

func FormatAddress(address string) string {
	return strings.Split(address, ":")[0]
}

func GetExecutablePath() string {
	executablePath, err := os.Executable()
	if err != nil {
		log.Errorf("Error getting executable path: %s", err)
		return ""
	}

	return filepath.Dir(executablePath)
}

func ClearFile(filePath string) {
	// Open the file in read-write mode
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		log.Errorf("Error opening file: %s", err)
		return
	}
	defer file.Close()

	// Clear the file by truncating it to size 0
	if err := os.Truncate(filePath, 0); err != nil {
		log.Errorf("Error truncating file: %s", err)
		return
	}

	log.Infof("File '%s' has been cleared.", filePath)
}

const (
	salt        = "CHUJ"
	keySize     = 32
	iteration   = 65536
	ivSize      = aes.BlockSize
	paddingSize = 16
)

func DecryptData(encryptedData []byte, masterKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(masterKey))
	if err != nil {
		return nil, err
	}

	// Extract the IV from the encrypted data
	iv := encryptedData[:aes.BlockSize]
	encryptedData = encryptedData[aes.BlockSize:]

	// Decrypt the data
	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)

	// Remove padding from the decrypted data
	decryptedData = removePadding(decryptedData)

	return decryptedData, nil
}

// removePadding removes PKCS#7 padding from the data.
func removePadding(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}
