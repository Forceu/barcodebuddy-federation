package webserver

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/redis"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

// templateFolderEmbedded is the embedded version of the "templates" folder
//go:embed templates
var templateFolderEmbedded embed.FS

// Variable containing all parsed templates
var templateFolder *template.Template

func Start() {
	initTemplates()
	go updateBarcodeCount()
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/amount", handleAmount)
	http.HandleFunc("/get", handleGetBarcode)
	http.HandleFunc("/vote", handleVote)
	http.HandleFunc("/report", handleReport)
	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/admin", handleAdmin)
	fmt.Println("Starting webserver on " + configuration.Get().WebserverPort)
	srv := &http.Server{
		Addr:         configuration.Get().WebserverPort,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// Initialises the templateFolder variable by scanning through all the templates.
func initTemplates() {
	var err error
	templateFolder, err = template.ParseFS(templateFolderEmbedded, "templates/*.tmpl")
	if err != nil {
		log.Fatal(err)
	}
}

func updateBarcodeCount() {
	for {
		redis.GetTotalBarcodes()
		time.Sleep(6 * time.Hour)
	}
}

type ResponseError struct {
	Result       string `json:"Result"`
	ErrorMessage string `json:"ErrorMessage"`
}

const GENERIC_RESPONSE_OK = "{\"Result\":\"ok\"}"

type ResponseBarcodeFound struct {
	Result     string   `json:"Result"`
	FoundNames []string `json:"FoundNames"`
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

// Sends a redirect HTTP output to the client. Variable url is used to redirect to ./url
func redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, "./"+url, http.StatusTemporaryRedirect)
}
