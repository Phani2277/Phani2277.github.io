package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"sort"
	"strings"
)

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

func checkTelegramAuth(initData string) bool {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	hash := values.Get("hash")
	if hash == "" {
		return false
	}
	values.Del("hash")

	var data []string
	for k, v := range values {
		data = append(data, k+"="+v[0])
	}

	sort.Strings(data)
	dataCheckString := strings.Join(data, "\n")

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(os.Getenv("BOT_TOKEN")))

	h := hmac.New(sha256.New, secret.Sum(nil))
	h.Write([]byte(dataCheckString))

	return hmac.Equal([]byte(hex.EncodeToString(h.Sum(nil))), []byte(hash))
}

func extractTelegramUser(initData string) (*TelegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	userJSON := values.Get("user")
	if userJSON == "" {
		return nil, nil
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, err
	}

	return &user, nil
}
