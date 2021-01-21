package main

import (
	"github.com/mediocregopher/radix/v3"
	"log"
	"net/http"
	"strings"
)

var redisPool *radix.Pool

func connectToRedis() {
	var err error
	redisPool, err = radix.NewPool("tcp", globalConfig.RedisUrl, globalConfig.RedisSize)
	if err != nil {
		log.Fatal(err)
	}
}

func logNewRequest(r *http.Request) int {
	ipAddr := strings.Split(r.RemoteAddr, ":")[0]
	var requests int
	_ = redisPool.Do(radix.Cmd(&requests, "INCR", "requests:"+ipAddr))
	_ = redisPool.Do(radix.Cmd(nil, "EXPIRE", "requests:"+ipAddr, getSecondsToMidnight()))
	return requests
}

func getBarcode(barcode string) []string {
	var storedBarcodes []string

	_ = redisPool.Do(radix.Cmd(&storedBarcodes, "ZRANGEBYSCORE", "barcode:"+barcode, "-1", "+inf"))
	_ = redisPool.Do(radix.Cmd(nil, "INCR", "hits:"+barcode))
	return storedBarcodes
}

func voteName(barcode, name string, r *http.Request) bool {
	ipAddr := strings.Split(r.RemoteAddr, ":")[0]
	var voteCount int
	_ = redisPool.Do(radix.Cmd(&voteCount, "INCR", "vote:"+ipAddr+":"+barcode+":"+name))
	if voteCount != 1 {
		return false
	}
	_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "barcode:"+barcode, "1", name))
	return true
}

func reportName(barcode, name string, r *http.Request) bool {
	ipAddr := strings.Split(r.RemoteAddr, ":")[0]
	var reportCount int
	_ = redisPool.Do(radix.Cmd(&reportCount, "INCR", "report:"+ipAddr+":"+barcode+":"+name))
	if reportCount != 1 {
		return false
	}

	//Checking score first, if "" is returned the item does not exist and would be created by ZINCRBY
	var score string
	_ = redisPool.Do(radix.Cmd(&score, "ZSCORE", "barcode:"+barcode, name))
	if score != "" {
		_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "barcode:"+barcode, "-2", name))
		return true
	} else {
		return false
	}
}

func addGrocyBarcodes(barcodes GrocyBarcodes, uuid string) {
	key := "grocyBarcodes"
	_ = redisPool.Do(radix.WithConn(key, func(conn radix.Conn) error {
		for _, barcode := range barcodes.Barcodes {
			_ = conn.Do(radix.FlatCmd(nil, "ZADD", "barcode:"+barcode.Barcode, "NX", "1", barcode.Name))
			_ = conn.Do(radix.FlatCmd(nil, "SET", "log:uuid:"+barcode.Barcode+":"+barcode.Name, uuid))
		}
		return nil
	}))

}
