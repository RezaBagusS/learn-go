package middleware

import (
	"belajar-go/projectAPI/dto"
	"fmt"
	"net/http"
	"strings"
)

func ErrorHandling(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)

		if pattern == "" || pattern == "/" {
			dto.WriteError(w, http.StatusNotFound, "Route tidak ditemukan!")
			return
		}

		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) == 2 && parts[0] != r.Method {
			dto.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed!")
			return
		}

		fmt.Printf("Tracking Pattern [middleware]: %s\n", pattern)

		// ✅ Gunakan mux.ServeHTTP agar PathValue tetap terbaca
		mux.ServeHTTP(w, r)
	}
}
