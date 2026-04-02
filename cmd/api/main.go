package main

import (
	"mikhailjbs/chat-service/internal/app"
	_ "mikhailjbs/chat-service/docs" // required for swaggo
)

// @title Katgen Chat Service API
// @version 1.0
// @description This is the real-time chat and contacts service.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@katgen.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8081
// @BasePath /
func main() {
	app.Run()
}
