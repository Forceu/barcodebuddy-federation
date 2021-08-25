package webserver

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/helper"
	"BarcodeServer/internal/redis"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "public, max-age=3600")
	http.Redirect(w, r, configuration.Get().WebserverRedirect, http.StatusSeeOther)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "public, max-age=60")
	fmt.Fprintf(w, "pong")
}
func handleAmount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "public, max-age=1800")
	fmt.Fprintf(w, strconv.Itoa(redis.AmountStoredBarcodes))
}

func handleGetBarcode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 {
		storedNames := redis.GetBarcode(barcode, true)
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
	w.Header().Set("cache-control", "private")
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	name := r.Header.Get("name")
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 && len(name) > 1 {
		redis.VoteName(barcode, name, r)
		sendGenericResultOK(w)
	} else {
		sendBadRequest(w)
		return
	}
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	uuid := r.Header.Get("uuid")
	requests := redis.LogNewRequest(r, uuid, true)
	if requests > configuration.Get().ApiDailyCallsUpload {
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
	var barcodes redis.GrocyBarcodes
	err = json.Unmarshal(body, &barcodes)
	if err != nil {
		sendBadRequest(w)
		return
	}
	if barcodes.Barcodes == nil {
		sendBadRequest(w)
		return
	}
	redis.AddGrocyBarcodes(barcodes, uuid)
	sendGenericResultOK(w)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	barcode := r.Header.Get("barcode")
	uuid := r.Header.Get("uuid")
	name := r.Header.Get("name")
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
		return
	}
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	if len(barcode) > 4 && len(name) > 1 {
		redis.ReportName(barcode, name, r)
		sendGenericResultOK(w)
	} else {
		sendBadRequest(w)
		return
	}
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	reportIdDelete, _ := r.URL.Query()["delete"]
	reportIdDismiss, _ := r.URL.Query()["dismiss"]
	exportButton, _ := r.URL.Query()["export"]

	if exportButton != nil {
		serveCsv(w, r, redis.GetDownloadBarcodesAsCsv())
		return
	}

	if reportIdDelete != nil {
		id, err := strconv.Atoi(reportIdDelete[0])
		if err == nil {
			reports := redis.GetReportList()
			if id >= 0 && id < len(reports) {
				redis.ProcessReport(reports[id], false)
				http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
				return
			}
		} else {
			fmt.Println("Invalid ID for report provided.")
		}
	}
	if reportIdDismiss != nil {
		id, err := strconv.Atoi(reportIdDismiss[0])
		if err == nil {
			reports := redis.GetReportList()
			if id >= 0 && id < len(reports) {
				redis.ProcessReport(reports[id], true)
				http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
				return
			}
		} else {
			fmt.Println("Invalid ID for report provided.")
		}
	}

	fmt.Fprintf(w, "<html><style>body {padding: 15px;background-color: #222222;color: #d9d9d9;}</style><title>Barcode Buddy Federation Admin</title><h2>Barcode Buddy Federation Admin</h2><br>")
	fmt.Fprintf(w, "Total barcodes: "+strconv.Itoa(redis.GetTotalBarcodes())+"<br>")
	fmt.Fprintf(w, "Unique users: "+strconv.Itoa(redis.GetTotalUsers())+"<br><br>")
	fmt.Fprintf(w, "RAM Usage: "+redis.GetRamUsage()+"<br>")
	totalRam, freeRam, err := helper.GetRamInfo()
	if err == nil {
		fmt.Fprintf(w, "Free RAM: "+freeRam+" of "+totalRam+"<br><br>")
	}
	fmt.Fprintf(w, "Blocked IPs: "+strconv.Itoa(len(blockedIPs))+"<br><br>")
	fmt.Fprintf(w, "Total votes: "+strconv.Itoa(redis.GetTotalVotes())+"<br>")
	fmt.Fprintf(w, "Total reports: "+strconv.Itoa(redis.GetTotalReports())+"<br><br>")
	fmt.Fprintf(w, "<a href='/admin?export' style='color: inherit;'>Export barcodes</a><br><br>")
	fmt.Fprintf(w, "<h3>Reports</h3>")

	reports := redis.GetReportList()
	length := len(reports)
	for i := 0; i <= length-1; i = i + 2 {
		fmt.Fprintf(w, reports[i]+" ("+reports[i+1]+") &nbsp;&nbsp;&nbsp;<a href='/admin?delete="+
			strconv.Itoa(i)+"' style='color: inherit;'>Remove barcode</a>&nbsp;&nbsp;<a href='/admin?dismiss="+
			strconv.Itoa(i)+"' style='color: inherit;'>Dismiss reports</a><br>")
	}

	fmt.Fprintf(w, "<br><h4>Top 50 barcodes</h4><br>")

	topBarcodes := redis.GetMostPopularBarcodes()
	length = len(topBarcodes)
	for i := 0; i <= length-1; i = i + 2 {
		barcode := topBarcodes[i]
		hits := topBarcodes[i+1]
		names := redis.GetBarcode(barcode, false)
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

func serveCsv(w http.ResponseWriter, r *http.Request, data [][]string) {
	w.Header().Set("Content-Disposition", "attachment; filename=exportBarcodes.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	writer := csv.NewWriter(w)
	_ = writer.WriteAll(data)
}
