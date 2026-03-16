package main

import (
	"errors"
	"fmt"
)

type ConversionResult struct {
	Currency string
	Amount   float64
}

var ratesMap = map[string]float64{
	"USD": 15000,
	"EUR": 16000,
	"JPY": 140,
	"SGD": 11000,
}

func Convertion(Currency string, Amount float64) (ConversionResult, error) {

	rate, ok := ratesMap[Currency]

	if !ok {
		return ConversionResult{}, errors.New("Error: Currency Type not found!")
	}

	return ConversionResult{
		Currency: Currency,
		Amount:   Amount / rate,
	}, nil

}

func Q1() {

	currSlice := []ConversionResult{}

	var inputPrice int

	fmt.Print("Masukkan jumlah uang dalam Rupiah: ")
	fmt.Scanln(&inputPrice)

	fmt.Printf("Konversi dari %d IDR: \n", inputPrice)

	for currencyKey := range ratesMap {

		result, err := Convertion(currencyKey, float64(inputPrice))

		if err != nil {
			fmt.Println("Gagal mengonversi:", err)
			continue
		}

		currSlice = append(currSlice, result)

		fmt.Printf("%s: %.2f\n", result.Currency, result.Amount)
	}

}
