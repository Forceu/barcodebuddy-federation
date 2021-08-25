package configuration

import (
	"BarcodeServer/internal/helper"
	models "BarcodeServer/internal/webserver/sessions/model"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

const configFilePath = "config/"
const configFile = configFilePath + "config.json"

var config Configuration
var sessionMutex sync.Mutex

const currentConfigVersion = 2

type Configuration struct {
	RedisUrl            string                    `json:"RedisUrl"`
	RedisSize           int                       `json:"RedisSize"`
	ApiDailyCalls       int                       `json:"ApiDailyCalls"`
	ApiDailyCallsUpload int                       `json:"ApiDailyCallsUpload"`
	AdminUser           string                    `json:"AdminUser"`
	AdminPassword       string                    `json:"AdminPassword"`
	WebserverPort       string                    `json:"WebserverPort"`
	WebserverRedirect   string                    `json:"WebserverRedirect"`
	ConfigVersion       int                       `json:"ConfigVersion"`
	Sessions            map[string]models.Session `json:"Sessions"`
}

func Load() {
	if !helper.FileExists(configFile) {
		generateDefault()
	}
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config = Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	if config.ConfigVersion < currentConfigVersion {
		upgrade()
	}
}

func Get() *Configuration {
	return &config
}

func GetSessions() *map[string]models.Session {
	sessionMutex.Lock()
	return &config.Sessions
}
func UnlockSession() {
	sessionMutex.Unlock()
}

func SaveSessions() {
	save()
	UnlockSession()
}

func generateDefault() {
	config = Configuration{
		RedisUrl:            "127.0.0.1:6379",
		RedisSize:           10,
		ApiDailyCalls:       200,
		ApiDailyCallsUpload: 5,
		AdminUser:           "admin",
		AdminPassword:       "admin",
		WebserverPort:       "127.0.0.1:18900",
		WebserverRedirect:   "https://github.com/Forceu/barcodebuddy",
		ConfigVersion:       currentConfigVersion,
		Sessions:            make(map[string]models.Session),
	}
	fmt.Println("First start, generated initial configuration")
	_ = os.Mkdir(configFilePath, 0700)
	save()
}

func upgrade() {
	if config.ConfigVersion < 2 {
		config.Sessions = make(map[string]models.Session)
	}
	config.ConfigVersion = currentConfigVersion
	save()
}

func save() {
	file, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&config)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}
