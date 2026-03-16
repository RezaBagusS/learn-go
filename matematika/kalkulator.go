package matematika

import (
	"errors"
	"fmt"
	"strings"
)

// Huruf Depan Kapital = Public (Exported)
func tambah(a int, b int) int {
	return a + b
}

// Huruf Depan Kecil = Private (Unexported)
func kurang(a int, b int) int {
	return a - b
}

func bagi(a int, b int) int {
	return a / b
}

func kali(a int, b int) int {
	return a * b
}

// --- 1. CONSTANTS & IOTA ---
// Menggunakan IOTA untuk membuat daftar kode operasi secara otomatis
// TambahOp = 0, KurangOp = 1, KaliOp = 2, BagiOp = 3
// Fitur ini hanya bisa digunakan di dalam blok const (...).

// Constant biasa (nilai yang tidak pernah berubah)
const BatasMaksimal = 100

const (
	TambahOp = iota // Baris ke-0: iota bernilai 0 (inisialisasi)
	KurangOp        // Baris ke-1: otomatis bernilai 1
	KaliOp          // Baris ke-2: otomatis bernilai 2
	BagiOp          // Baris ke-3: otomatis bernilai 3
) // BISA UNTUK ENUM

// --- 2. FUNCTIONS, OPERATORS, SWITCH, & ERROR HANDLING ---
// Fungsi Hitung menerima 2 angka dan 1 kode operasi, lalu mengembalikan (hasil int, error)
func Hitung(firstNumber int, secondNumber int, operateCode int) (int, error) {
	switch operateCode {
	case TambahOp:
		return tambah(firstNumber, secondNumber), nil // 'nil' berarti tidak ada error
	case KurangOp:
		return kurang(firstNumber, secondNumber), nil // 'nil' berarti tidak ada error
	case KaliOp:
		return kali(firstNumber, secondNumber), nil // 'nil' berarti tidak ada error
	case BagiOp:

		if secondNumber <= 0 {
			return 0, errors.New("Error: Pembagi tidak boleh bernilai 0 atau dibawah 0")
		}

		return bagi(firstNumber, secondNumber), nil // 'nil' berarti tidak ada error
	default:
		return 0, errors.New("Error: Operate code not found!")
	}
}

// --- 3. LOOPING ---
// Fungsi untuk mencetak deret angka menggunakan For Loop
func CetakDeret(jumlah int) {
	for i := 0; i < jumlah; i++ {
		fmt.Printf("%d ", i)
	}

	fmt.Println()
}

// --- 4. Pointer Concepts ---
// & (Ampersand): Digunakan untuk mengambil alamat memori dari sebuah variabel.
// * (Asterisk): Digunakan untuk membaca atau mengubah nilai yang ada di alamat memori tersebut.

// --- 5. Struct ---
type Profile struct {
	UserId   int
	Username string
	Email    string
}

// --- 6. Methods: Value vs Pointer Receivers ---
// Value Receiver: Menggunakan copy (salinan) dari objek.
// Digunakan jika Anda hanya ingin membaca data, tanpa mengubahnya
func (p Profile) CekProfile() {
	result := fmt.Sprintf("Akun %s memiliki email: %s\n", p.Username, p.Email)

	fmt.Println(result)
}

// Pointer Receiver: Menggunakan alamat memori objek asli.
// Digunakan jika Anda ingin mengubah data di dalam objek tersebut
// 2. POINTER RECEIVER (Mengubah data asli)
// Perhatikan (a *AkunKeuangan) menggunakan bintang (*)
func (p *Profile) ChangeEmail(email string) error {

	if !strings.Contains(email, "@cimb") {
		return fmt.Errorf("Error: Input email harus menggunakan email CIMB")
	}
	p.Email = email

	return nil
}

// --- 7. Interfaces as Contracts ---
type Akun interface {
	CekProfile()
	ChangeEmail(email string) error
}

// --- 8. Dependency Injection ---
func ProsesUpdateEmail(p Akun, email string) {
	fmt.Printf("\nMemproses updating akun %s...\n", email)

	err := p.ChangeEmail(email)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Profile berhasil diupdate")

}
