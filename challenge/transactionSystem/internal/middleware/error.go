package middleware

import (
	"belajar-go/challenge/transactionSystem/dto"
	"fmt"
	"net/http"
)

type responseInterceptor struct {
	http.ResponseWriter
	statusCode int
}

// Override fungsi WriteHeader untuk menangkap status code
func (ri *responseInterceptor) WriteHeader(code int) {
	ri.statusCode = code

	if code == http.StatusNotFound || code == http.StatusMethodNotAllowed {
		return
	}
	ri.ResponseWriter.WriteHeader(code)
}

// Override fungsi Write untuk mengabaikan body/pesan text bawaan dari Go ("404 page not found")
func (ri *responseInterceptor) Write(b []byte) (int, error) {
	if ri.statusCode == http.StatusNotFound || ri.statusCode == http.StatusMethodNotAllowed {
		return len(b), nil
	}
	return ri.ResponseWriter.Write(b)
}

func ErrorHandling(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interceptor := &responseInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default asumsi sukses
		}

		fmt.Printf("Tracking Request [middleware]: %s %s\n", r.Method, r.URL.Path)

		// Biarkan ServeMux memproses routing menggunakan interceptor
		mux.ServeHTTP(interceptor, r)

		// 3. Cek hasil dari proses ServeMux
		if interceptor.statusCode == http.StatusNotFound {
			dto.WriteError(w, http.StatusNotFound, "Route tidak ditemukan!")
			return
		}

		if interceptor.statusCode == http.StatusMethodNotAllowed {
			dto.WriteError(w, http.StatusMethodNotAllowed, "Method tidak ditemukan!")
			return
		}
	}
}
