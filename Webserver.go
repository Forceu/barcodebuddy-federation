package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//GET
//PUT
//VOTE
//report

// Block IP for 6 hours
var blockedIPs []string

/** Prevent brute force. With too many invalid password attempts, admin access is disabled*/
var remainingLoginTries = 10

func startWebserver() {
	go clearBlockedIPs()
	http.HandleFunc("/get", handleGetBarcode)
	http.HandleFunc("/vote", handleVote)
	http.HandleFunc("/report", handleReport)
	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/admin", basicAuth(handleAdmin, "Admin"))
	fmt.Println("Starting webserver on " + globalConfig.WebserverPort)
	log.Fatal(http.ListenAndServe(globalConfig.WebserverPort, nil))
}

func basicAuth(handler http.HandlerFunc, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ipAddr := strings.Split(r.RemoteAddr, ":")[0]
		if isIPBlocked(ipAddr) {
			sendTooManyRequests(w)
			return
		}
		if remainingLoginTries < 1 {
			sendTooManyRequests(w)
			blockedIPs = append(blockedIPs, ipAddr)
			fmt.Println("Blocked IP " + ipAddr)
			remainingLoginTries = 10
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user),
			[]byte(globalConfig.AdminUser)) != 1 || subtle.ConstantTimeCompare([]byte(pass),
			[]byte(globalConfig.AdminPassword)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("You are Unauthorized to access the application.\n"))
			remainingLoginTries--
			fmt.Println("Invalid login from " + ipAddr)
			return
		}
		handler(w, r)
	}
}

func handleGetBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	requests := logNewRequest(r)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) && false { //TODO
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 {
		storedNames := getBarcode(barcode)
		if len(storedNames) > 0 {
			response := ResponseBarcodeFound{
				Result:     "ok",
				FoundNames: storedNames,
			}
			responseString, _ := json.Marshal(response)
			sendResultOK(w, responseString)
		} else {
			sendBarcodeNotFound(w)
		}
	} else {
		sendBadRequest(w)
		return
	}
}

func handleVote(w http.ResponseWriter, r *http.Request) {
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	name := r.Header.Get("name")
	requests := logNewRequest(r)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) && false { //TODO
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 && len(name) > 1 {
		voteName(barcode, name, r)
		sendGenericResultOK(w)
	} else {
		sendBadRequest(w)
		return
	}
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	uuid := r.Header.Get("uuid")
	requests := logNewRequest(r)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sendBadRequest(w)
		return
	}
	var barcodes GrocyBarcodes
	err = json.Unmarshal(body, &barcodes)
	if err != nil {
		sendBadRequest(w)
		return
	}
	if barcodes.Barcodes == nil {
		sendBadRequest(w)
		return
	}
	addGrocyBarcodes(barcodes, uuid)
	sendGenericResultOK(w)
	if !isValidUuid(uuid) && false { //TODO
		sendBadRequest(w)
		return
	}
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	name := r.Header.Get("name")
	requests := logNewRequest(r)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) && false { //TODO
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 && len(name) > 1 {
		reportName(barcode, name, r)
		sendGenericResultOK(w)
	} else {
		sendBadRequest(w)
		return
	}
}

type ResponseError struct {
	Result       string `json:"Result"`
	ErrorMessage string `json:"ErrorMessage"`
}

type GrocyBarcodes struct {
	Barcodes []Barcode `json:"ServerBarcodes"`
}

type Barcode struct {
	Barcode string `json:"Barcode"`
	Name    string `json:"Name"`
}

const GENERIC_RESPONSE_OK = "{\"Result\":\"ok\"}"

type ResponseBarcodeFound struct {
	Result     string   `json:"Result"`
	FoundNames []string `json:"FoundNames"`
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><h2>BarcodeServer</h2><br>Total barcodes: "+strconv.Itoa(getTotalBarcodes())+"<br>")
	fmt.Fprintf(w, "Total votes: "+strconv.Itoa(getTotalVotes())+"<br>")
	fmt.Fprintf(w, "Total reports: "+strconv.Itoa(getTotalReports())+"<br><br>")
	fmt.Fprintf(w, "<h3>Reports</h3>")

	reports := getReportList()
	length := len(reports)
	for i := 0; i <= length-1; i = i + 2 {
		fmt.Fprintf(w, reports[i]+" ("+reports[i+1]+") <a href='#'>Remove barcode</a> <a href='#'>Discard reports</a><br>")
	}
}

// Clears blockedIPs array every 6 hours
func clearBlockedIPs() {
	for {
		blockedIPs = []string{}
		time.Sleep(time.Hour * 6)
	}
}

func isIPBlocked(ipAddr string) bool {
	for _, ip := range blockedIPs {
		if ip == ipAddr {
			return true
		}
	}
	return false
}

func getSecondsToMidnight() string {
	t := time.Now()
	return strconv.Itoa((24 * 60 * 60) - (60*60*t.Hour() + 60*t.Minute() + t.Second()))
}

func isValidUuid(uuid string) bool {
	return len(uuid) == 32
}

func sendTooManyRequests(w http.ResponseWriter) {
	result := ResponseError{
		Result:       "error",
		ErrorMessage: "Too many requests",
	}
	response, _ := json.Marshal(result)
	http.Error(w, string(response), http.StatusTooManyRequests)
}

func sendResultOK(w http.ResponseWriter, response []byte) {
	fmt.Fprintf(w, string(response))
}

func sendGenericResultOK(w http.ResponseWriter) {
	fmt.Fprintf(w, GENERIC_RESPONSE_OK)
}

func sendBarcodeNotFound(w http.ResponseWriter) {
	result := ResponseError{
		Result:       "error",
		ErrorMessage: "Barcode not found",
	}
	response, _ := json.Marshal(result)
	fmt.Fprintf(w, string(response))
}

func sendBadRequest(w http.ResponseWriter) {
	result := ResponseError{
		Result:       "error",
		ErrorMessage: "Bad request",
	}
	response, _ := json.Marshal(result)
	http.Error(w, string(response), http.StatusTooManyRequests)
}
