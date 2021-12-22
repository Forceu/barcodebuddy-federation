package main

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/import/edeka"
	"BarcodeServer/internal/redis"
	"BarcodeServer/internal/webserver"
	"fmt"
)

func main() {

	configuration.Load()
	redis.Connect()
	syncEdeka()
	webserver.Start()
}

func syncEdeka() {
	apiKey := configuration.Get().ApiKeyEdeka
	if apiKey == "" {
		fmt.Println("No Edeka API Key provided.")
		return
	}
	go edeka.StartPeriodicSync(apiKey)
}
