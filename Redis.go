package main

import (
	"github.com/mediocregopher/radix/v3"
	"html/template"
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

func logNewRequest(r *http.Request, uuid string, isUpload bool) int {
	ipAddr := getIpAddress(r)
	keyName := "requests:"
	if isUpload {
		keyName = "requests_upload:"
	}
	var requests int
	_ = redisPool.Do(radix.Cmd(&requests, "INCR", keyName+ipAddr))
	_ = redisPool.Do(radix.Cmd(nil, "EXPIRE", keyName+ipAddr, getSecondsToMidnight()))
	_ = redisPool.Do(radix.Cmd(nil, "SADD", "users", uuid))
	return requests
}

func getBarcode(barcode string, increaseHit bool) []string {
	var storedBarcodes []string

	_ = redisPool.Do(radix.Cmd(&storedBarcodes, "ZREVRANGEBYSCORE", "barcode:"+barcode, "+inf", "-1"))
	if increaseHit {
		_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "hits", "1", barcode))
	}
	return storedBarcodes
}

func voteName(barcode, name string, r *http.Request) bool {
	ipAddr := getIpAddress(r)
	var voteCount int
	_ = redisPool.Do(radix.Cmd(&voteCount, "INCR", "vote:"+ipAddr+":"+barcode+":"+name))
	if voteCount != 1 {
		return false
	}
	_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "barcode:"+barcode, "1", name))
	return true
}

func reportName(barcode, name string, r *http.Request) bool {
	ipAddr := getIpAddress(r)
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
		_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "reported:"+barcode, "1", name))
		_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "reports", "1", barcode+":"+name))
		return true
	} else {
		return false
	}
}

func processReport(barcodeAndName string, dismissReport bool) {
	score := "-100"
	if dismissReport {
		score = "1"
	}
	splitArray := strings.SplitN(barcodeAndName, ":", 2)
	barcode := splitArray[0]
	name := splitArray[1]

	_ = redisPool.Do(radix.Cmd(nil, "ZADD", "barcode:"+barcode, score, name))
	_ = redisPool.Do(radix.Cmd(nil, "ZREM", "reported:"+barcode, name))
	_ = redisPool.Do(radix.Cmd(nil, "ZREM", "reports", barcodeAndName))
}

func addGrocyBarcodes(barcodes GrocyBarcodes, uuid string) {
	key := "grocyBarcodes"
	_ = redisPool.Do(radix.WithConn(key, func(conn radix.Conn) error {
		for _, barcode := range barcodes.Barcodes {
			barcodeSanitized := template.HTMLEscapeString(barcode.Barcode)
			nameSanitized := template.HTMLEscapeString(barcode.Name)
			if len(barcodeSanitized) > 4 && len(barcodeSanitized) < 30 && len(nameSanitized) > 2 && len(nameSanitized) < 50 {
				_ = conn.Do(radix.FlatCmd(nil, "ZADD", "barcode:"+barcodeSanitized, "NX", "1", nameSanitized))
				_ = conn.Do(radix.FlatCmd(nil, "SET", "log:uuid:"+barcodeSanitized+":"+nameSanitized, uuid))
			}
		}
		return nil
	}))
}

func getTotalBarcodes() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'barcode:*')", "0"))
	storedBarcodes = amount
	return amount
}

func getTotalVotes() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'vote:*')", "0"))
	return amount
}

func getTotalReports() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'report:*')", "0"))
	return amount
}

func getReportList() []string {
	var result []string
	_ = redisPool.Do(radix.Cmd(&result, "ZREVRANGEBYSCORE", "reports", "+inf", "0", "WITHSCORES"))
	return result
}

func getMostPopularBarcodes() []string {
	var result []string
	_ = redisPool.Do(radix.Cmd(&result, "ZREVRANGEBYSCORE", "hits", "+inf", "1", "WITHSCORES", "LIMIT", "0", "25"))
	return result
}

func getTotalUsers() int {
	var result int
	_ = redisPool.Do(radix.Cmd(&result, "SCARD", "users"))
	return result
}
