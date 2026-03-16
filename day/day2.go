package day

import (
	"errors"
	"fmt"
)

type Person struct {
	Name  string
	Email string
}

type Notifier interface {
	SendNotification() error
	DeleteNotification() error
}



type MockNotifier struct {
	title string
	msg   string
}

type EmailNotifier struct {
	*MockNotifier
	email string
}

type SMSNotifier struct {
	*MockNotifier
	phone string
}

type NotificationService struct {
	notifier Notifier
}

func (e EmailNotifier) SendNotification() error {
	if len(e.msg) == 0 {
		return errors.New("Pesan Email tidak boleh kosong ... ")
	}

	fmt.Println("Sending Email ....")
	fmt.Println("Email Msg: ", e.msg)
	return nil
}

func (s SMSNotifier) SendNotification() error {
	if len(s.msg) == 0 {
		return errors.New("Pesan SMS tidak boleh kosong ... ")
	}

	fmt.Println("Sending SMS ....")
	fmt.Println("SMS Msg: ", s.msg)
	return nil
}

func (e EmailNotifier) DeleteNotification() error {
	fmt.Println("Menghapus Email ...")
	return nil
}

func (s SMSNotifier) DeleteNotification() error {
	fmt.Println("Menghapus SMS ...")
	return nil
}

// Getter (Value Receiver)
func (p Person) getProfile() Person {
	return p
}

// Setter (Pointer Receiver)
func (p *Person) setNameProfile(newName string) {
	p.Name = newName
}

// Error Custom Validation
func errorValidation(err error) {
	if err != nil {
		fmt.Println("Error : ", err)
	}
}

// Notify Func
func notify(n Notifier) {
	err := n.SendNotification()

	errorValidation(err)

	err = n.DeleteNotification()

	errorValidation(err)
}

func NewNotificationService(n Notifier) *NotificationService {
	return &NotificationService{
		notifier: n,
	}
}

func (ns *NotificationService) NotifyUser() {
	ns.notifier.SendNotification()
}

func Day2() {
	poin := 10           // initiate variable poin
	pointerPoin := &poin // mengirim alamat variabel poin

	*pointerPoin = 50

	fmt.Println("Poin : ", poin)
	fmt.Println("PointerPoin : ", pointerPoin)

	fmt.Println("============= Methods (Value vs Pointer) ==========")

	// Value Receiver
	profileData := Person{
		Name:  "INI NAMA SAYA",
		Email: "INI EMAIL SAYA",
	}

	fmt.Println("Name : ", profileData.Name)
	fmt.Println("Email : ", profileData.Email)

	profileData.setNameProfile("Nama Baru Saya")

	fmt.Println("New Name : ", profileData.Name)

	fmt.Println("============= Interfaces ============")

	mockData := MockNotifier{"Promo", "Promo meriah 11.11"}

	emailNotify := EmailNotifier{
		MockNotifier: &mockData,
		email:        "info@cimbniaga.co.id",
	}

	smsNotify := SMSNotifier{
		MockNotifier: &mockData,
		phone:        "0811128323",
	}

	emailNotify.SendNotification()
	smsNotify.SendNotification()

	emailNotify.DeleteNotification()
	smsNotify.DeleteNotification()

	fmt.Println("============= Interfaces ============")

	service1 := NewNotificationService(emailNotify)
	service1.NotifyUser()
}
