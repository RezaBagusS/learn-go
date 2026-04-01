package helper

import (
	"bytes"
	"compress/zlib"
	"io"
)

// CompressData mengompresi byte array menggunakan zlib
func CompressData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	w.Close()
	return b.Bytes(), nil
}

// DecompressData mengembalikan data terkompresi ke bentuk aslinya
func DecompressData(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
