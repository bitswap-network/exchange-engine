package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// func createHandshake(message string) string {
// 	authKey := os.Getenv("SERVER_AUTH")
// 	s := strings.Split(message, ".")
// 	iv, _ := hex.DecodeString(s[0])
// 	messageTimeNonce, _ := hex.DecodeString(s[1])
// 	block, err := aes.NewCipher([]byte(authKey))
// 	if err != nil {
// 		log.Println(err)
// 		return false
// 	}
// 	mode := cipher.NewCBCDecrypter(block, iv)
// 	mode.CryptBlocks(messageTimeNonce, messageTimeNonce)
// 	messageTimeNonce
// }

func validateHMAC(signature, data []byte) bool {
	authKey := os.Getenv("SERVER_AUTH")
	mac := hmac.New(sha256.New, []byte(authKey))
	mac.Write(data)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(signature, []byte(hex.EncodeToString(expectedMAC)))
}

func internalServerAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		switch ENV_MODE {
		case "release":
			signature, ok := c.Request.Header["Server-Signature"]
			if !ok {
				c.String(http.StatusBadRequest, "Where da signature at doe?")
			}
			messageBuffer := new(bytes.Buffer)
			messageBuffer.ReadFrom(c.Request.Body)
			if validateHMAC([]byte(signature[0]), messageBuffer.Bytes()) {
				c.Next()
			} else {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
		case "debug":
			c.Next()
		}
	}
}
