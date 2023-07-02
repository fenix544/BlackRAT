package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
)

func RandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func CheckFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func CreateTempFile(fileName string) *os.File {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", fileName)
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return nil
	}

	fmt.Println("Temporary file created:", tempFile.Name())
	return tempFile
}

func CopyFile(originalFile string, tempFile *os.File) {
	// Open the original file
	original, err := os.Open(originalFile)
	if err != nil {
		fmt.Println("Error opening original file:", err)
		return
	}

	// Copy the contents of the original file to the temporary file
	_, err = io.Copy(tempFile, original)
	if err != nil {
		fmt.Println("Error copying original file:", err)
		return
	}

	CloseFile(original)
}

func CloseFile(file *os.File) {
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
}
