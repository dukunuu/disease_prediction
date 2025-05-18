package main

import (
	"context"
	"log"

	"github.com/dukunuu/munkhjin-diplom/backend/config"
	"github.com/dukunuu/munkhjin-diplom/backend/db"
	"github.com/dukunuu/munkhjin-diplom/backend/server"

	_ "github.com/dukunuu/munkhjin-diplom/backend/docs"
)

// @title           Patient API Service
// @version         1.0
// @description     API Service for managing patients, symptoms, and diseases.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @schemes   http https
func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Could not load config: %v", err);
	}

	db, err := db.Init(cfg.DB_Url, ctx)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err);
	}

	srv := server.Init(db, cfg.Model_Url)

	err = srv.Start(cfg.Port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
