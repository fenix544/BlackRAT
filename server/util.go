package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"golang.org/x/crypto/pbkdf2"
	"os"
	"os/exec"
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

const (
	salt        = "CHUJ"
	keySize     = 32
	iteration   = 65536
	ivSize      = aes.BlockSize
	paddingSize = 16
)

func DecryptData(data []byte, master string) ([]byte, error) {
	// Generate a derived key from the master password and salt
	derivedKey := pbkdf2.Key([]byte(master), []byte(salt), iteration, keySize, sha256.New)

	// Extract the IV from the data
	iv := data[:ivSize]

	// Create a new AES cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// Create a cipher block mode for AES CBC mode
	mode := cipher.NewCBCDecrypter(block, iv)

	// Decrypt the data (excluding the IV)
	decryptedData := make([]byte, len(data)-ivSize)
	mode.CryptBlocks(decryptedData, data[ivSize:])

	// Remove PKCS#7 padding from the decrypted data
	unpaddedData := unpadData(decryptedData)

	return unpaddedData, nil
}

func unpadData(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}
