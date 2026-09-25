package main

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/helper"
	"BarcodeServer/internal/redis"
	"encoding/csv"
	"fmt"
	"github.com/gocarina/gocsv"
	"io"
	"os"
)

type csvInput struct {
	Barcode string `csv:"barcode"`
	Name    string `csv:"name"`
}

func main() {
	configuration.Load()
	redis.Connect()
	if !helper.FileExists("barcodes.csv") {
		fmt.Println("barcodes.csv not found for importing")
		os.Exit(1)
	}

	in, err := os.Open("barcodes.csv")
	if err != nil {
		panic(err)
	}
	defer in.Close()

	var barcodes []*csvInput

	// Use pipe as delimiter instead of comma
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = '|'
		return r
	})

	if err := gocsv.UnmarshalFile(in, &barcodes); err != nil {
		panic(err)
	}
	var redisBarcodes []redis.Barcode
	for _, barcode := range barcodes {
		redisBarcode := redis.Barcode{
			Barcode: barcode.Barcode,
			Name:    barcode.Name,
		}
		redisBarcodes = append(redisBarcodes, redisBarcode)
		// lookup := redis.GetBarcode(barcode.Barcode, false)
		// if len(lookup) == 0 {
		// 	newBarcodes++
		// }

	}
	redis.AddGrocyBarcodes(redis.GrocyBarcodes{Barcodes: redisBarcodes}, "importer")
	fmt.Printf("Imported %d barcodes.\n", len(barcodes))
}
