package middleware

import (
	"my-microservices/transaction-service/internal/dto"
	"fmt"
	"net/http"
	"strconv"
)

type responseInterceptor struct {
	http.ResponseWriter
	statusCode   int
	routeMatched bool
	headerSent   bool
}

func (ri *responseInterceptor) WriteHeader(code int) {
	ri.statusCode = code
	ri.routeMatched = true

	if code == http.StatusMethodNotAllowed {
		return
	}

	ri.headerSent = true
	ri.ResponseWriter.WriteHeader(code)
}

func (ri *responseInterceptor) Write(b []byte) (int, error) {
	ri.routeMatched = true

	if ri.statusCode == http.StatusMethodNotAllowed {
		return len(b), nil
	}

	return ri.ResponseWriter.Write(b)
}

func ErrorHandling(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interceptor := &responseInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			routeMatched:   false,
		}

		fmt.Printf("Tracking Request [middleware]: %s %s\n", r.Method, r.URL.Path)

		mux.ServeHTTP(interceptor, r)

		if !interceptor.routeMatched {
			dto.WriteError(w, http.StatusNotFound, strconv.Itoa(http.StatusNotFound), "Route tidak ditemukan!")
			return
		}

		if interceptor.statusCode == http.StatusMethodNotAllowed {
			dto.WriteError(w, http.StatusMethodNotAllowed, strconv.Itoa(http.StatusMethodNotAllowed), "Method tidak diizinkan!")
			return
		}
	}
}
