package edeka

import (
	"BarcodeServer/internal/redis"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const importUrl = "https://gc-lb.heig.net/eanexport?key="

type edekaItem struct {
	Brand    string     `json:"brand"`
	Name     string     `json:"name"`
	Barcodes [][]string `json:"EAN"`
}

func itemsToBarcodes(response []edekaItem) redis.GrocyBarcodes {
	var result []redis.Barcode
	for _, product := range response {
		var name string
		if product.Brand != "" {
			name = product.Brand + " " + product.Name
		} else {
			name = product.Name
		}
		// It would save resources to create a slice with the length determined, for better
		// readability and robustness append has been used instead however
		for _, barcode := range product.Barcodes[0] {
			result = append(result, redis.Barcode{
				Barcode: barcode,
				Name:    name,
			})
		}
	}
	return redis.GrocyBarcodes{Barcodes: result}
}

func getBarcodesFromApi(apikey string) (error, []edekaItem) {
	apiClient := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, importUrl+apikey, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("User-Agent", "Barcode Buddy Federation")

	res, getErr := apiClient.Do(req)
	if getErr != nil {
		return err, nil
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return err, nil
	}
	if string(body) == "Wrong API Key" {
		return errors.New("incorrect api key"), nil
	}

	var response []edekaItem
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err, nil
	}
	return nil, response
}

func StartPeriodicSync(apikey string) {
	RunImport(apikey)
	time.Sleep(24 * time.Hour)
	go StartPeriodicSync(apikey)
}

func RunImport(apikey string) {
	if apikey == "" {
		return
	}
	err, response := getBarcodesFromApi(apikey)
	if err != nil {
		log.Println("Unable to sync Edeka barcodes: " + err.Error())
		return
	}
	log.Println("Edeka Import: Total products " + strconv.Itoa(len(response)))
	barcodes := itemsToBarcodes(response)
	log.Println("Edeka Import: Total barcodes " + strconv.Itoa(len(barcodes.Barcodes)))
	redis.AddGrocyBarcodes(barcodes, "edeka")
}
