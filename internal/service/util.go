package service

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

func randomCode() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buffer)
}
