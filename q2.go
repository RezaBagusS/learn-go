package main

import (
	"fmt"
	"strings"
)

func cardValidation(inputCardNumber string) string {
	prefixSwitch := []string{"4903", "4905", "4911", "4936", "564182", "633110", "6333", "6759"}
	prefixUnionPay := []string{"62"}

	if len(inputCardNumber) < 16 || len(inputCardNumber) > 19 {
		return "Nomor kartu tidak valid (Number length less than 16 & more than 19)"
	}

	for i := 0; i < len(prefixSwitch); i++ {
		if strings.HasPrefix(inputCardNumber, prefixSwitch[i]) && len(inputCardNumber) != 17 {
			return "Switch"
		}
	}

	for i := 0; i < len(prefixUnionPay); i++ {

		if strings.HasPrefix(inputCardNumber, prefixUnionPay[i]) {
			return "China UnionPay"
		}
	}

	return "Type Card Not Found"
}

func Q2() {

	bulkInput := []string{}
	bulkResult := []string{}

	fmt.Println("Masukkan daftar nomor kartu (Enter) & untuk berhenti input 0 + Enter: ")

	var inputCardNumber string

	for {
		fmt.Scanln(&inputCardNumber)
		// fmt.Println(inputCardNumber)
		if inputCardNumber == "0" {
			break
		}

		bulkInput = append(bulkInput, inputCardNumber)

	}

	// fmt.Println(bulkInput)

	for i := 0; i < len(bulkInput); i++ {
		typeCard := cardValidation(bulkInput[i])

		bulkResult = append(bulkResult, typeCard)

		// fmt.Println(bulkResult)

		switch bulkResult[i] {
		case "China UnionPay", "Switch":
			fmt.Printf("Nomor %s adalah %s \n", bulkInput[i], bulkResult[i])
		default:
			fmt.Println("Jenis kartu tidak dikenali.")
		}
	}

}
