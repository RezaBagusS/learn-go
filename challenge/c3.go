package challenge

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type SampleDataStruct struct {
	Bank       string
	NoRekening string
	Saldo      int
}

type TransferDataStruct struct {
	ID           int
	FromRekening string
	ToRekening   string
	Amount       int
}

var SampleData = []SampleDataStruct{
	{
		Bank:       "CIMB",
		NoRekening: "C001",
		Saldo:      300000,
	},
	{
		Bank:       "MANDIRI",
		NoRekening: "M002",
		Saldo:      500000,
	},
	{
		Bank:       "BNI",
		NoRekening: "B003",
		Saldo:      400000,
	},
	{
		Bank:       "BCA",
		NoRekening: "BC04",
		Saldo:      800000,
	},
}

var TransferData = []TransferDataStruct{
	{
		ID:           1,
		FromRekening: "C001",
		ToRekening:   "M002",
		Amount:       300000,
	},
	{
		ID:           2,
		FromRekening: "C001",
		ToRekening:   "B003",
		Amount:       600000,
	},
	{
		ID:           3,
		FromRekening: "M002",
		ToRekening:   "BC04",
		Amount:       200000,
	},
	{
		ID:           4,
		FromRekening: "B003",
		ToRekening:   "C001",
		Amount:       500000,
	},
	{
		ID:           5,
		FromRekening: "BC04",
		ToRekening:   "M002",
		Amount:       700000,
	},
	{
		ID:           6,
		FromRekening: "C001",
		ToRekening:   "M002",
		Amount:       400000,
	},
	{
		ID:           7,
		FromRekening: "B003",
		ToRekening:   "C001",
		Amount:       300000,
	},
}

func transaction(i int, data []SampleDataStruct, mtx *sync.Mutex, ctx context.Context, wg *sync.WaitGroup) {

	randomDuration := time.Duration(rand.Intn(5)+1) * time.Second
	// second := int(randomDuration / time.Second)

	defer wg.Done()

	tx := TransferData[i]
	log.Printf("Processing TX-%d (%ds) | %s -> %s | Rp%d\n", tx.ID, randomDuration, tx.FromRekening, tx.ToRekening, tx.Amount)

	select {
	case <-time.After(randomDuration):

		deadline, ok := ctx.Deadline()

		if ok {

			log.Println(time.Until(deadline), "TX-", tx.ID)
			if time.Until(deadline) < 0 {
				log.Printf("TX-%d TIMEOUT\n", TransferData[i].ID)
				return
			}

		}

		mtx.Lock()
		defer mtx.Unlock()

		if ctx.Err() != nil {
			// return
			ctx.Done()
		}

		var msg string

		senderIdx, receiverIdx := -1, -1
		for idx, val := range data {
			if val.NoRekening == tx.FromRekening {
				senderIdx = idx
			}
			if val.NoRekening == tx.ToRekening {
				receiverIdx = idx
			}
		}

		if senderIdx != -1 && receiverIdx != -1 && data[senderIdx].Saldo >= tx.Amount {

			data[senderIdx].Saldo -= tx.Amount
			data[receiverIdx].Saldo += tx.Amount

			msg = fmt.Sprintf("TX-%d SUCCESS \n Saldo %s sekarang Rp%d\n Saldo %s sekarang Rp%d\n", tx.ID, tx.FromRekening, data[senderIdx].Saldo, tx.ToRekening, data[receiverIdx].Saldo)
		} else {
			msg = fmt.Sprintf("TX-%d FAILED | %s -> %s | Transfer Rp%d | Saldo %s hanya Rp%d \n", tx.ID, tx.FromRekening, tx.ToRekening, tx.Amount, tx.FromRekening, data[senderIdx].Saldo)
		}

		log.Println(msg)
	case <-ctx.Done():
		// 4s Time out
		log.Printf("TX-%d TIMEOUT\n", TransferData[i].ID)
	}
}

func Challenge3() {

	var data []SampleDataStruct = SampleData
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	for i := 0; i < len(TransferData); i++ {
		wg.Add(1)
		go func(i int) {
			ctx, cancel := context.WithTimeout(context.Background(), 4998*time.Millisecond)
			defer cancel()
			transaction(i, data, &mtx, ctx, &wg)
		}(i)
	}

	wg.Wait()

	log.Println("\n===== FINAL BALANCE =====")
	for i := 0; i < len(data); i++ {
		str := fmt.Sprintf("%s (%s) : Rp%d", data[i].Bank, data[i].NoRekening, data[i].Saldo)
		log.Println(str)
	}
}
