package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, globalConfig.WebserverRedirect, http.StatusSeeOther)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}
func handleAmount(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, strconv.Itoa(storedBarcodes))
}

func handleGetBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	requests := logNewRequest(r, uuid, false)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 {
		storedNames := getBarcode(barcode, true)
		if len(storedNames) > 0 {
			response := ResponseBarcodeFound{
				Result:     "OK",
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
	requests := logNewRequest(r, uuid, false)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
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
	requests := logNewRequest(r, uuid, true)
	if requests > globalConfig.ApiDailyCallsUpload {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
		sendBadRequest(w)
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
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	name := r.Header.Get("name")
	requests := logNewRequest(r, uuid, false)
	if requests > globalConfig.ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
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

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><title>Barcode Buddy Federation Admin</title><h2>Barcode Buddy Federation Admin</h2><br>Total barcodes: "+strconv.Itoa(getTotalBarcodes())+"<br>")
	fmt.Fprintf(w, "Unique users: "+strconv.Itoa(getTotalUsers())+"<br>")
	fmt.Fprintf(w, "Blocked IPs: "+strconv.Itoa(len(blockedIPs))+"<br><br>")
	fmt.Fprintf(w, "Total votes: "+strconv.Itoa(getTotalVotes())+"<br>")
	fmt.Fprintf(w, "Total reports: "+strconv.Itoa(getTotalReports())+"<br><br>")
	fmt.Fprintf(w, "<h3>Reports</h3>")

	reports := getReportList()
	length := len(reports)
	for i := 0; i <= length-1; i = i + 2 {
		fmt.Fprintf(w, reports[i]+" ("+reports[i+1]+") <a href='#'>Remove barcode</a> <a href='#'>Discard reports</a><br>")
	}

	fmt.Fprintf(w, "<br><h4>Top 25 barcodes</h4><br>")

	topBarcodes := getMostPopularBarcodes()
	length = len(topBarcodes)
	for i := 0; i <= length-1; i = i + 2 {
		barcode := topBarcodes[i]
		hits := topBarcodes[i+1]
		names := getBarcode(barcode, false)
		fmt.Fprintf(w, barcode+" ("+hits+"):")
		if len(names) > 0 {
			for j, name := range names {
				fmt.Fprintf(w, " "+name)
				if j < len(names)-1 {
					fmt.Fprintf(w, ",")
				}
			}
		}
		fmt.Fprintf(w, "<br>")
	}
}
