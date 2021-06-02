package config

import (
	"log"
	"os"
	"strings"
	"time"
)

var IsTest bool

type Server struct {
	RunMode      string
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var ServerConfig = &Server{}

type Database struct {
	AWSKey          string
	AWSSecret       string
	ClusterEndpoint string
	DatabaseName    string
}

var DatabaseConfig = &Database{}

type S3 struct {
	Region  string
	Bucket  string
	LogName string
}

var S3Config = &S3{}

func Setup() {
	envMap := getEnvMap(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})
	log.Println(envMap)

	ServerConfig.RunMode = envMap["ENV_MODE"]
	ServerConfig.Addr = ":"+envMap["PORT"]
	if ServerConfig.Addr == ":" {
		ServerConfig.Addr = ":5050"
	}
	if ServerConfig.RunMode == "release" {
		IsTest = false
	} else {
		IsTest = true
	}
	ServerConfig.ReadTimeout = 60 * time.Second
	ServerConfig.WriteTimeout = 60 * time.Second
	DatabaseConfig.AWSKey = envMap["MONGODB_USERNAME"]
	DatabaseConfig.AWSSecret = envMap["MONGODB_PASSWORD"]
	DatabaseConfig.ClusterEndpoint = envMap["MONGODB_ENDPOINT"]
	if IsTest {
		DatabaseConfig.DatabaseName = "staging"
	} else {
		DatabaseConfig.DatabaseName = "production"
	}

	S3Config.Region = "us-east-1"
	S3Config.LogName = "orderbook"
	S3Config.Bucket = envMap["BUCKET"]
}

func getEnvMap(data []string, getkeyval func(item string) (key, val string)) map[string]string {
	items := make(map[string]string)
	for _, item := range data {
		key, val := getkeyval(item)
		items[key] = val
	}
	return items
}
