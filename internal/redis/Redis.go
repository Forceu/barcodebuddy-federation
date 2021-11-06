package redis

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/helper"
	"github.com/mediocregopher/radix/v3"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var redisPool *radix.Pool
var AmountStoredBarcodes int

// TimespanActiveUser is the time in seconds in which the user must have sent
// a request in order to count as being active. Default is 30 days (2592000s)
const TimespanActiveUser = "2592000"

type GrocyBarcodes struct {
	Barcodes []Barcode `json:"ServerBarcodes"`
}

type Barcode struct {
	Barcode string `json:"Barcode"`
	Name    string `json:"Name"`
}

func Connect() {
	var err error
	redisPool, err = radix.NewPool("tcp", configuration.Get().RedisUrl, configuration.Get().RedisSize)
	if err != nil {
		log.Fatal(err)
	}
}

func LogNewRequest(r *http.Request, uuid string, isUpload bool) int {
	ipAddr := helper.GetIpAddress(r)
	keyName := "requests:"
	if isUpload {
		keyName = "requests_upload:"
	}
	var requests int
	_ = redisPool.Do(radix.Cmd(&requests, "INCR", keyName+ipAddr))
	_ = redisPool.Do(radix.Cmd(nil, "EXPIRE", keyName+ipAddr, helper.GetSecondsToMidnight()))
	_ = redisPool.Do(radix.Cmd(nil, "SADD", "users", uuid))
	_ = redisPool.Do(radix.Cmd(nil, "SET", "users:active:"+uuid, "1", "EX", TimespanActiveUser))
	return requests
}

func GetBarcode(barcode string, increaseHit bool) []string {
	var storedBarcodes []string

	_ = redisPool.Do(radix.Cmd(&storedBarcodes, "ZREVRANGEBYSCORE", "barcode:"+barcode, "+inf", "-1"))
	if increaseHit {
		_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "hits", "1", barcode))
	}
	return storedBarcodes
}

func VoteName(barcode, name string, r *http.Request) bool {
	ipAddr := helper.GetIpAddress(r)
	var voteCount int
	_ = redisPool.Do(radix.Cmd(&voteCount, "INCR", "vote:"+ipAddr+":"+barcode+":"+name))
	if voteCount != 1 {
		return false
	}
	_ = redisPool.Do(radix.Cmd(nil, "ZINCRBY", "barcode:"+barcode, "1", name))
	return true
}

func ReportName(barcode, name string, r *http.Request) bool {
	ipAddr := helper.GetIpAddress(r)
	var reportCount int
	_ = redisPool.Do(radix.Cmd(&reportCount, "INCR", "report:"+ipAddr+":"+barcode+":"+name))
	if reportCount != 1 {
		return false
	}

	// Checking score first, if "" is returned the item does not exist and would be created by ZINCRBY
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

func ProcessReport(report Report, dismissReport bool) {
	score := "-100"
	if dismissReport {
		score = "1"
	}
	splitArray := strings.SplitN(report.BarcodeAndName, ":", 2)
	barcode := splitArray[0]
	name := splitArray[1]

	_ = redisPool.Do(radix.Cmd(nil, "ZADD", "barcode:"+barcode, score, name))
	_ = redisPool.Do(radix.Cmd(nil, "ZREM", "reported:"+barcode, name))
	_ = redisPool.Do(radix.Cmd(nil, "ZREM", "reports", report.BarcodeAndName))
}

func AddGrocyBarcodes(barcodes GrocyBarcodes, uuid string) {
	key := "grocyBarcodes"
	_ = redisPool.Do(radix.WithConn(key, func(conn radix.Conn) error {
		for _, barcode := range barcodes.Barcodes {
			barcodeSanitized := template.HTMLEscapeString(barcode.Barcode)
			nameSanitized := template.HTMLEscapeString(barcode.Name)
			if len(barcodeSanitized) > 4 && len(barcodeSanitized) < 30 && len(nameSanitized) > 2 && len(nameSanitized) < 50 {
				_ = conn.Do(radix.FlatCmd(nil, "ZADD", "barcode:"+barcodeSanitized, "NX", "1", nameSanitized))
				_ = conn.Do(radix.FlatCmd(nil, "SET", "log:uuid:"+barcodeSanitized+":"+nameSanitized, uuid, "EX", "345600")) // 4 days
			}
		}
		return nil
	}))
}

func GetTotalBarcodes() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'barcode:*')", "0"))
	AmountStoredBarcodes = amount
	return amount
}

func GetTotalVotes() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'vote:*')", "0"))
	return amount
}

func GetTotalActiveUsers() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'users:active:*')", "0"))
	return amount
}

func GetTotalReports() int {
	var amount int
	_ = redisPool.Do(radix.Cmd(&amount, "EVAL", "return #redis.pcall('keys', 'report:*')", "0"))
	return amount
}

func GetReportList() []Report {
	var reports []string
	var result []Report
	_ = redisPool.Do(radix.Cmd(&reports, "ZREVRANGEBYSCORE", "reports", "+inf", "0", "WITHSCORES"))
	length := len(reports)
	for i := 0; i <= length-1; i = i + 2 {
		result = append(result, Report{
			Id:             i,
			BarcodeAndName: reports[i],
			ReportCount:    reports[i+1],
		})
	}
	return result
}

type Report struct {
	Id             int
	BarcodeAndName string
	ReportCount    string
}

func GetMostPopularBarcodes() []TopBarcode {
	var barcodes []string
	var result []TopBarcode
	_ = redisPool.Do(radix.Cmd(&barcodes, "ZREVRANGEBYSCORE", "hits", "+inf", "1", "WITHSCORES", "LIMIT", "0", "50"))
	length := len(barcodes)
	for i := 0; i <= length-1; i = i + 2 {
		barcode := TopBarcode{
			Barcode: barcodes[i],
			Hits:    barcodes[i+1],
		}
		appendNamesToTopBarcode(&barcode)
		result = append(result, barcode)
	}
	return result
}

func appendNamesToTopBarcode(barcode *TopBarcode) {
	var result string
	names := GetBarcode(barcode.Barcode, false)
	if len(names) > 0 {
		for i, name := range names {
			result = result + " " + name
			if i < len(names)-1 {
				result = result + ","
			}
		}
	}
	barcode.Names = result
}

type TopBarcode struct {
	Barcode string
	Hits    string
	Names   string
}

func GetTotalUsers() int {
	var result int
	_ = redisPool.Do(radix.Cmd(&result, "SCARD", "users"))
	return result
}

func GetRamUsage() string {
	var result []string
	_ = redisPool.Do(radix.Cmd(&result, "MEMORY", "STATS"))
	for i, item := range result {
		if item == "total.allocated" {
			totalAmount, err := strconv.ParseUint(result[i+1], 10, 64)
			if err != nil {
				return "Invalid Value"
			}
			return helper.ByteCountSI(totalAmount)
		}
	}
	return "Unknown"
}

func GetDownloadBarcodesAsCsv() [][]string {
	var redisResult []string
	var result [][]string

	result = append(result, []string{"barcode", "names"})

	_ = redisPool.Do(radix.Cmd(&redisResult, "EVAL", "local result = {} local matches = redis.call('KEYS', 'barcode:*') for _,key in ipairs(matches) do result[#result+1] = key local names = redis.call('ZREVRANGEBYSCORE', key, '+inf', -1) for _,keyName in ipairs(names) do result[#result+1] = keyName end end return result", "0"))
	for _, value := range redisResult {
		if strings.HasPrefix(value, "barcode:") {
			result = append(result, []string{strings.Replace(value, "barcode:", "", 1)})
		} else {
			lastIndex := len(result) - 1
			result[lastIndex] = append(result[lastIndex], value)
		}
	}
	return result
}
