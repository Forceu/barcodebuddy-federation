package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var blockedIPs []string
var storedBarcodes = 0

/** Prevent brute force. With too many invalid password attempts, admin access is disabled*/
var remainingLoginTries = 10

func startWebserver() {
	go updateBarcodeCount()
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/amount", handleAmount)
	http.HandleFunc("/get", handleGetBarcode)
	http.HandleFunc("/vote", handleVote)
	http.HandleFunc("/report", handleReport)
	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/admin", basicAuth(handleAdmin, "Admin"))
	fmt.Println("Starting webserver on " + globalConfig.WebserverPort)
	log.Fatal(http.ListenAndServe(globalConfig.WebserverPort, nil))
}

func updateBarcodeCount() {
	for {
		getTotalBarcodes()
		time.Sleep(6 * time.Hour)
	}
}

func basicAuth(handler http.HandlerFunc, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ipAddr := getIpAddress(r)
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
			w.Write([]byte("You are not authorised to access the application.\n"))
			remainingLoginTries--
			fmt.Println("Invalid login from " + ipAddr)
			return
		}
		remainingLoginTries = 10
		handler(w, r)
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

func getIpAddress(r *http.Request) string {
	//Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	//Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "undefined-ip"
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}
	return "undefined-ip"
}
