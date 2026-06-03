package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/oroitz-lago-ramos/minecraft-panel/internal/auth"
	"github.com/oroitz-lago-ramos/minecraft-panel/internal/handlers"
	"github.com/oroitz-lago-ramos/minecraft-panel/internal/minecraft"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	// Charger les variables d'environnement
	godotenv.Load()

	// 1. Connexion MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("❌ MongoDB ne répond pas: %v", err)
	}
	log.Println("✅ MongoDB connecté")

	db := client.Database("minecraft_panel")

	// 2. Services
	authService := auth.NewService(db)
	authService.EnsureAdminExists()

	// Minecraft
	mcServer := minecraft.NewServer(
		os.Getenv("RCON_ADDR"),
		os.Getenv("RCON_PASSWORD"),
	)

	// 3. Handlers
	authHandler := auth.NewHandler(authService)
	serverHandler := handlers.NewServerHandler(mcServer)
	wsHandler := handlers.NewWSHandler(mcServer)

	// 4. Gin + CORS
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "https://mc.oroitzlagoramos.com")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 5. Routes publiques
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", auth.RateLimitLogin(), authHandler.Login)
		authRoutes.POST("/logout", authHandler.Logout)
	}

	// 6. Routes protégées
	api := r.Group("/api")
	api.Use(auth.AuthMiddleware())
	{
		// Auth
		api.GET("/auth/me", authHandler.Me)

		// Stats système
		api.GET("/stats", handlers.GetStats)

		// Commandes Minecraft
		api.GET("/server/status", serverHandler.Status)
		api.GET("/server/uptime", serverHandler.Uptime)
		api.POST("/server/start", auth.AdminOnly(), serverHandler.Start)
		api.POST("/server/stop", auth.AdminOnly(), serverHandler.Stop)
		api.GET("/server/players", serverHandler.Players)
		api.POST("/server/command", auth.AdminOnly(), serverHandler.SendCommand)
		api.GET("/server/worlds", serverHandler.GetWorlds)
		api.POST("/server/worlds/switch", auth.AdminOnly(), serverHandler.SwitchWorld)
		api.POST("/server/worlds/upload", auth.AdminOnly(), serverHandler.UploadWorld)
		api.DELETE("/server/worlds/:name", auth.AdminOnly(), serverHandler.DeleteWorld)
		api.GET("/server/worlds/:name/backup", auth.AdminOnly(), serverHandler.BackupWorld)
		api.GET("/ws/console", wsHandler.Console)
	}

	// 7. Lancer
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Serveur sur :%s", port)
	r.Run(":" + port)
}
