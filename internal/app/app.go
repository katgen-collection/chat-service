package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"mikhailjbs/chat-service/internal/config"
	"mikhailjbs/chat-service/internal/domain/contacts"
	"mikhailjbs/chat-service/internal/domain/message"
	"mikhailjbs/chat-service/internal/domain/user"
	"mikhailjbs/chat-service/internal/infra/http"
	"mikhailjbs/chat-service/internal/infra/http/handlers"
	"mikhailjbs/chat-service/internal/infra/logger"
	"mikhailjbs/chat-service/internal/infra/middleware"
	"mikhailjbs/chat-service/internal/infra/repository"
	"mikhailjbs/chat-service/internal/infra/event"
	ws "mikhailjbs/chat-service/internal/infra/websocket"

	contactsUseCase "mikhailjbs/chat-service/internal/usecase/contacts"
)

func Run() {
	// 1. Load Config
	cfg := config.Load()

	// 2. Init Logger
	logger.Init()
	logger.Log.Info("Starting Chat Service...")

	// 3. Init Database
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require pool_mode=session",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: cfg.SchemaName + ".",
		},
	})
	if err != nil {
		logger.Log.Fatalf("Failed to connect to database: %v", err)
	}

	if cfg.MongoURI == "" {
		logger.Log.Fatal("Missing MONGO_URI in configuration")
	}
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()
	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logger.Log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		logger.Log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.Log.WithError(err).Warn("Failed to disconnect MongoDB cleanly")
		}
	}()

	// First, ensure the schema exists in the database
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + cfg.SchemaName).Error; err != nil {
		logger.Log.Fatalf("Failed to create schema "+cfg.SchemaName+": %v", err)
	}

	// Auto Migrate (for development simplicity, usually done via migration tools)
	if err := db.AutoMigrate(&contacts.Contact{}, &contacts.ContactRequest{}, &user.UserSnapshot{}); err != nil {
		logger.Log.Fatalf("Failed to migrate database: %v", err)
	}

	// Remove index on email
	db.Exec("DROP INDEX IF EXISTS chat_service.idx_chat_service_contacts_email")
	db.Exec("ALTER TABLE chat_service.contacts DROP CONSTRAINT IF EXISTS idx_chat_service_contacts_email")

	// 4. Init Repository
	contactsRepo := repository.NewContactsRepository(db)
	messageRepo := repository.NewMessageRepository(mongoClient, cfg.MongoDatabase)

	// 5. Init Service (Domain)
	contactsService := contacts.NewService(contactsRepo)
	messageService := message.NewService(messageRepo)

	// 6. Init UseCases
	// createUserUC := usecase.NewCreateUserUseCase(userService)
	getContactsUC := contactsUseCase.NewGetContactsUseCase(contactsService)
	getContactUC := contactsUseCase.NewGetContactUseCase(contactsService)
	updateContactUC := contactsUseCase.NewUpdateContactUseCase(contactsService)
	deleteContactUC := contactsUseCase.NewDeleteContactUseCase(contactsService)

	getContactRequestsUC := contactsUseCase.NewGetContactRequestsUseCase(contactsService)
	getContactRequestUC := contactsUseCase.NewGetContactRequestUseCase(contactsService)
	updateContactRequestUC := contactsUseCase.NewUpdateContactRequestUseCase(contactsService)
	deleteContactRequestUC := contactsUseCase.NewDeleteContactRequestUseCase(contactsService)
	createContactRequestUC := contactsUseCase.NewCreateContactRequestUseCase(contactsService)

	// 7. Init Handlers
	// userHandler := handlers.NewUserHandler(createUserUC, getUsersUC, getUserUC, updateUserUC, deleteUserUC)
	contactHandler := handlers.NewContactHandler(
		getContactUC,
		getContactsUC,
		updateContactUC,
		deleteContactUC,
	)

	wsHub := ws.NewHub(contactsService, messageService, logger.Log)

	contactRequestHandler := handlers.NewContactRequestHandler(
		createContactRequestUC,
		getContactRequestUC,
		getContactRequestsUC,
		updateContactRequestUC,
		deleteContactRequestUC,
		wsHub,
	)

	// Message HTTP handler (history pagination)
	messageHandler := handlers.NewMessageHandler(messageService)

	websocketHandler := handlers.NewWebsocketHandler(wsHub)

	// 8. Init Server
	app := http.NewServer(cfg.CorsAllowedOrigins)

	// Start Redis Subscriber for User Snapshots
	event.StartRedisSubscriber(context.Background(), cfg.RedisURL, db)

	// Init Auth Middleware (Trusts API Gateway Headers)
	authzMiddleware := middleware.NewAuthMiddleware(middleware.Config{
		ContextKey:        middleware.DefaultClaimsContextKey,
	})

	// 9. Register Routes
	//http.RegisterRoutes(app, userHandler, authHandler)
	http.RegisterRoutes(app, contactHandler, contactRequestHandler, messageHandler, websocketHandler, authzMiddleware)

	// 10. Start Server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Log.Infof("Server listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
