package webserver

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/helper"
	"BarcodeServer/internal/redis"
	sessionmanager "BarcodeServer/internal/webserver/sessions"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
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
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
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
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
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
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	if requests > configuration.Get().ApiDailyCallsUpload {
		sendTooManyRequests(w)
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
	if !isValidUuid(uuid) {
		sendBadRequest(w)
		return
	}
	requests := redis.LogNewRequest(r, uuid, false)
	if requests > configuration.Get().ApiDailyCalls {
		sendTooManyRequests(w)
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

func handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	sessionmanager.LogoutSession(w, r)
	redirect(w, r, "login")
}
func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	err := r.ParseForm()
	if err != nil {
		log.Panicln(err)
		return
	}
	username := r.Form.Get("username")
	password := r.Form.Get("password")
	isInvalidLogin := false

	if username != "" && password != "" {
		if helper.SecureStringEqual(username, configuration.Get().AdminUser) && helper.SecureStringEqual(password, configuration.Get().AdminPassword) {
			sessionmanager.CreateSession(w, nil)
			redirect(w, r, "admin")
			return
		} else {
			time.Sleep(2 * time.Second)
			isInvalidLogin = true
		}
	}
	err = templateFolder.ExecuteTemplate(w, "login", loginVariables{IsFailedLogin: isInvalidLogin, User: username})
	if err != nil {
		log.Panicln(err)
	}
}

type loginVariables struct {
	IsFailedLogin bool
	User          string
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "private")
	if !sessionmanager.IsValidSession(w, r) {
		time.Sleep(1 * time.Second)
		redirect(w, r, "login")
		return
	}
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
			for _, report := range reports {
				if report.Id == id {
					redis.ProcessReport(report, false)
					redirect(w, r, "admin")
					return
				}
			}
		} else {
			fmt.Println("Invalid ID for report provided.")
		}
	}
	if reportIdDismiss != nil {
		id, err := strconv.Atoi(reportIdDismiss[0])
		if err == nil {
			reports := redis.GetReportList()
			for _, report := range reports {
				if report.Id == id {
					redis.ProcessReport(report, true)
					redirect(w, r, "admin")
					return
				}
			}
		} else {
			fmt.Println("Invalid ID for report provided.")
		}
	}

	view := adminView{
		TotalBarcodes: redis.GetTotalBarcodes(),
		Users:         redis.GetTotalUsers(),
		UsersActive:   redis.GetTotalActiveUsers(),
		RamUsage:      redis.GetRamUsage(),
		TotalVotes:    redis.GetTotalVotes(),
		TotalReports:  redis.GetTotalReports(),
		Reports:       redis.GetReportList(),
		TopBarcodes:   redis.GetMostPopularBarcodes(),
	}

	totalRam, freeRam, err := helper.GetRamInfo()
	if err == nil {
		view.FreeRam = "Free RAM: " + freeRam + " of " + totalRam
	}
	err = templateFolder.ExecuteTemplate(w, "admin", view)
	if err != nil {
		log.Panicln(err)
	}

}

type adminView struct {
	TotalBarcodes int
	Users         int
	UsersActive   int
	TotalVotes    int
	TotalReports  int
	RamUsage      string
	FreeRam       string
	Reports       []redis.Report
	TopBarcodes   []redis.TopBarcode
}

func serveCsv(w http.ResponseWriter, r *http.Request, data [][]string) {
	w.Header().Set("Content-Disposition", "attachment; filename=exportBarcodes.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	writer := csv.NewWriter(w)
	_ = writer.WriteAll(data)
}
