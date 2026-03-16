package baseCodeAPI

import (
	"belajar-go/baseCodeAPI/handler"
	"belajar-go/baseCodeAPI/server"
	"net/http"
)

func BaseAPI() {

	mux := http.NewServeMux()

	aboutHandler := handler.NewAboutHandler(mux)
	aboutHandler.MapRoutes()

	// "POST /user"
	// mux.HandleFunc(server.NewAPIPath(http.MethodPost, "/user"), func(w http.ResponseWriter, r *http.Request) {
	// 	w.WriteHeader(http.StatusOK)
	// 	json.NewEncoder(w).Encode(dto.BaseResponse{
	// 		ResponseCode: "200",
	// 		ResponseDesc: "Success from /user",
	// 	})
	// })

	http.ListenAndServe(":8080",
		server.ApplicationMiddlewareResponse(
			server.HandleRouteNotFound(mux),
		),
	)
}
