package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"strings"

	"crypto/sha256"
	"golang.org/x/crypto/pbkdf2"
)

const (
	salt        = "CHUJ"
	keySize     = 32
	iteration   = 65536
	ivSize      = aes.BlockSize
	paddingSize = 16
)

func EncryptData(data []byte, master string) ([]byte, error) {
	// Generate a derived key from the master password and salt
	derivedKey := pbkdf2.Key([]byte(master), []byte(salt), iteration, keySize, sha256.New)

	// Generate a random initialization vector (IV)
	iv := make([]byte, ivSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// Apply PKCS#7 padding to the data
	paddedData := padData(data)

	// Create a cipher block mode for AES CBC mode
	mode := cipher.NewCBCEncrypter(block, iv)

	// Encrypt the data
	encryptedData := make([]byte, len(paddedData))
	mode.CryptBlocks(encryptedData, paddedData)

	// Prepend the IV to the encrypted data
	encryptedDataWithIV := append(iv, encryptedData...)

	return encryptedDataWithIV, nil
}

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

func padData(data []byte) []byte {
	padding := paddingSize - (len(data) % paddingSize)
	padText := strings.Repeat(string(byte(padding)), padding)
	return append(data, []byte(padText)...)
}

func unpadData(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}

func RandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
