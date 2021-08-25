package main

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/redis"
	"BarcodeServer/internal/webserver"
)

func main() {
	configuration.Load()
	redis.Connect()
	webserver.Start()
}
