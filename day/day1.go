package day

import (
	"belajar-go/matematika"
	"fmt"
)

func Day1() {
	fmt.Println("Hi, This is my first GO Program!")

	// hasilKurang := matematika.kurang(5, 1) ** Func tidak ditemukan karena func diawali huruf kecil (private)
	hitung, err := matematika.Hitung(5, -1, matematika.BagiOp)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Hasil :", hitung)
	}

	fmt.Println("\n=== Fitur Looping ===")
	matematika.CetakDeret(5)

	fmt.Println("\n=== OBJECT DST ===")
	profileSaya := matematika.Profile{
		UserId:   1,
		Username: "Testing",
		Email:    "emailTesting@cimbniaga.com",
	}

	profileSaya.CekProfile()
	profileSaya.ChangeEmail("changeEmail@cimbniaga.co.id")
	profileSaya.CekProfile()

	matematika.ProsesUpdateEmail(&profileSaya, "changeAgain@cimbniaga.co.id")
	profileSaya.CekProfile()

	matematika.ProsesUpdateEmail(&profileSaya, "invalid@niaga.co.id")
	profileSaya.CekProfile()

}
