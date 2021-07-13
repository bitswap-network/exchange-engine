package config

import (
	"log"
	"os"
	"strings"
	"time"
)

const BITCLOUT_NODEURL = "http://node.bitswap.network"

type WalletStruct struct {
	HashKey           string
	InitBcltTolerance float64
	InitEthTolerance  float64
}

var Wallet = &WalletStruct{}

var IsTest bool

type Util struct {
	ETHERSCAN_KEY string
}

var UtilConfig = &Util{}

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

	log.Println("config setup")
	envMap := getEnvMap(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})

	ServerConfig.RunMode = envMap["ENV_MODE"]
	ServerConfig.Addr = ":" + envMap["PORT"]
	if ServerConfig.Addr == ":" {
		ServerConfig.Addr = "localhost:5050"
	}
	if ServerConfig.RunMode == "release" {
		IsTest = false
	} else {
		IsTest = true
	}
	log.Println(IsTest)
	ServerConfig.ReadTimeout = 60 * time.Second
	ServerConfig.WriteTimeout = 60 * time.Second
	DatabaseConfig.AWSKey = envMap["MONGODB_USERNAME"]
	DatabaseConfig.AWSSecret = envMap["MONGODB_PASSWORD"]
	DatabaseConfig.ClusterEndpoint = envMap["MONGODB_ENDPOINT"]

	DatabaseConfig.DatabaseName = envMap["DB_NAME"]

	S3Config.Region = "us-east-1"
	S3Config.LogName = "orderbook"
	S3Config.Bucket = envMap["BUCKET"]
	UtilConfig.ETHERSCAN_KEY = envMap["ETHERSCAN_KEY"]
	Wallet.HashKey = envMap["WALLET_HASHKEY"]

	if IsTest {
		Wallet.InitBcltTolerance = -54.0773643
		Wallet.InitEthTolerance = 0.80296661
	} else {
		Wallet.InitBcltTolerance = -54.3811639
		Wallet.InitEthTolerance = 7.21748675
	}

	log.Println("config setup complete")
}

func getEnvMap(data []string, getkeyval func(item string) (key, val string)) map[string]string {
	items := make(map[string]string)
	for _, item := range data {
		key, val := getkeyval(item)
		items[key] = val
	}
	return items
}
