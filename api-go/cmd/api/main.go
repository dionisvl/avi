package main

import (
	"github.com/dionisvl/avi/api-go/internal/app"
)

// @title           avi Marketplace API
// @version         1.0
// @description     Demo classifieds marketplace API (C2C listings: items, categories, favorites, chat, listing promotion).
// @description     Not affiliated with Avito, OLX, or any other classifieds platform.
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	a := app.New()
	if err := a.Run(); err != nil {
		panic(err)
	}
}
