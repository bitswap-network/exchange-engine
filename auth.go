package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"log"
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
			signature, ok := c.Request.Header["x-server-signature"]
			if !ok {
				c.String(http.StatusBadRequest, "Where da signature at doe?")
			}
			messageBuffer, err := ioutil.ReadAll(c.Request.Body)
			log.Println(string(messageBuffer))
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(messageBuffer))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			if validateHMAC([]byte(signature[0]), messageBuffer) {
				c.Next()
			} else {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
		case "debug":
			c.Next()
		}
	}
}
