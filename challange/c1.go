package challange

import (
	"errors"
	"fmt"
)

type Product struct {
	id    string
	name  string
	price float64
}

type ProductRepository interface {
	Save(p Product) error
}

type DatabaseRepo struct {
	product ProductRepository
}

func Save(p Product) error {
	fmt.Printf("Menyimpan data %s ... \nl", p.name)
	return nil
}

func (p *Product) ApplyDiscount(percentage float64) error {
	if percentage <= 0.0 {
		return errors.New("Nilai persentase tidak boleh kurang dari 0.0")
	}

	disc := p.price * percentage

	p.price -= disc
	return nil
}
