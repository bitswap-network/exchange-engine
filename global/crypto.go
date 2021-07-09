package global

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strings"
)

const GCMNonceSize = 12

func DecryptGCM(encryptedString string, keyString string) (decryptedString string, err error) {
	s := strings.Split(encryptedString, ":")
	data, err := base64.StdEncoding.DecodeString(s[0])
	nonce, err := base64.StdEncoding.DecodeString(s[1])
	tag, err := base64.StdEncoding.DecodeString(s[2])
	dataWithTag := append(data, tag...)
	var block cipher.Block
	block, err = aes.NewCipher([]byte(keyString))
	var aead cipher.AEAD
	aead, err = cipher.NewGCM(block)
	decrypted, err := aead.Open(nil, nonce, dataWithTag, nil)
	decryptedString = string(decrypted)
	return
}
func EncryptGCM(text string, keyString string) (encryptedString string, err error) {
	block, err := aes.NewCipher([]byte(keyString))
	if err != nil {
		log.Println(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Println(err)
	}
	nonce := make([]byte, GCMNonceSize)
	_, err = io.ReadFull(rand.Reader, nonce)
	encrypted := gcm.Seal(nil, nonce, []byte(text), nil)
	encData := encrypted[0 : len(encrypted)-gcm.Overhead()]
	tag := encrypted[len(encrypted)-gcm.Overhead():]
	encryptedString = fmt.Sprintf("%s:%s:%s", base64.StdEncoding.EncodeToString(encData), base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(tag))
	return
}
