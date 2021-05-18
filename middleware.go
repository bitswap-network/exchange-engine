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

func internalServerAuth() gin.HandlerFunc {

	authKey := os.Getenv("SERVER_AUTH")
	return func(c *gin.Context) {
		switch ENV_MODE {
		case "release":
			signature, ok := c.Request.Header["Server-Signature"]
			if !ok {
				c.String(http.StatusBadRequest, "Where da signature at doe?")
			}

			mac := hmac.New(sha256.New, []byte(authKey))
			messageBuffer := new(bytes.Buffer)
			messageBuffer.ReadFrom(c.Request.Body)
			mac.Write(messageBuffer.Bytes())
			expectedMAC := mac.Sum(nil)
			if hmac.Equal([]byte(signature[0]), []byte(hex.EncodeToString(expectedMAC))) {
				c.Next()
			} else {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
		case "debug":
			c.Next()
		}
	}
}
