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
	"v1.1-fulfiller/config"
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
		if !config.IsTest {
			signature, ok := c.Request.Header["Server-Signature"]
			if !ok {
				c.AbortWithStatus(http.StatusBadRequest)
				return
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
				return
			}
		} else {
			c.Next()
		}
	}
}
