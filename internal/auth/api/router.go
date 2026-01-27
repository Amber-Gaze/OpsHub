package api

import "github.com/fasthttp/router"

func RegisterRoutes(r *router.Router) {
	r.POST("/login", Login)
	r.POST("/authorize", Authorize)
}
