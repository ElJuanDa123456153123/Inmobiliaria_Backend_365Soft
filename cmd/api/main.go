package main

import (
	"log"
	"os"
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	adminhandlers "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers/admin"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/jobs"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno desde archivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No se encontró archivo .env, usando variables de entorno del sistema")
	}

	// Cargar configuración
	cfg := config.LoadConfig()

	// Ejecutar migraciones al iniciar
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Error ejecutando migraciones: %v", err)
	}

	// Crear app de Fiber
	app := fiber.New(fiber.Config{
		AppName: "Inmobiliaria API v1.0",
	})

	// 🔒 Middleware profesional
	app.Use(middleware.RecoverMiddleware())       // Recupera de panics
	app.Use(middleware.RequestIDMiddleware())     // Añade ID único
	app.Use(middleware.LoggingMiddleware())       // Logs estructurados
	app.Use(middleware.SecurityHeadersMiddleware()) // Headers de seguridad
	app.Use(middleware.ErrorHandlerMiddleware())  // Manejo centralizado de errores

	// 🚀 Redis disponible para rate limiting específico (login)
	// NOTA: Rate limiting SOLO se aplica a endpoints específicos, no globalmente
	propertyHandler, err := handlers.NewPropertyHandler()
	if err == nil && propertyHandler.Redis != nil {
		log.Println("✅ Redis conectado - Rate limiting disponible para endpoints específicos")
	} else {
		log.Println("⚠️  Rate limiting no disponible (Redis no conectado)")
	}

	// CORS (personalizado)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001,http://localhost:3002",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, Content-Type",
		MaxAge:           86400, // 24 horas
		// Permitir preflight requests
		Next:             nil,
	}))

	// Manejar explícitamente preflight OPTIONS requests
	app.Options("/*", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", c.Get("Origin"))
		c.Set("Access-Control-Allow-Credentials", "true")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Set("Access-Control-Max-Age", "86400")
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Inicializar handler de autenticación
	authHandler, err := handlers.NewAuthHandler()
	if err != nil {
		log.Fatalf("❌ Error inicializando auth handler: %v", err)
	}

	// Inicializar handler de Google Auth
	googleAuthHandler, err := handlers.NewGoogleAuthHandler()
	if err != nil {
		log.Printf("⚠️  Google Auth no disponible: %v", err)
		// No es fatal, continuar sin Google Auth
	}

	// Inicializar handlers de administración
	userHandler, err := adminhandlers.NewUserHandler()
	if err != nil {
		log.Printf("⚠️  User admin handler no disponible: %v", err)
	}

	advisorHandler, err := adminhandlers.NewAdvisorHandler()
	if err != nil {
		log.Printf("⚠️  Advisor admin handler no disponible: %v", err)
	}

	adminPropertyHandler, err := adminhandlers.NewPropertyHandler()
	if err != nil {
		log.Printf("⚠️  Property admin handler no disponible: %v", err)
	}

	analyticsHandler, err := adminhandlers.NewAnalyticsHandler()
	if err != nil {
		log.Printf("⚠️  Analytics admin handler no disponible: %v", err)
	}

	// Inicializar handler de reportes financieros
	reportsHandler, err := adminhandlers.NewReportsHandler()
	if err != nil {
		log.Printf("⚠️  Reports handler no disponible: %v", err)
	}

	// Inicializar handler de planes
	planHandler, err := handlers.NewPlanHandler()
	if err != nil {
		log.Printf("⚠️  Plan handler no disponible: %v", err)
	}

	// NOTA: propertyHandler ya fue inicializado arriba para rate limiting

	// Inicializar handler de imágenes
	imageHandler, err := handlers.NewImageHandler()
	if err != nil {
		log.Printf("⚠️  Image handler no disponible: %v", err)
		// No es fatal, continuar sin image handler
	}

	// Inicializar handler de favoritos
	favoriteHandler, err := handlers.NewFavoriteHandler()
	if err != nil {
		log.Printf("⚠️  Favorite handler no disponible: %v", err)
	}

	// Inicializar handler de búsquedas guardadas
	savedSearchHandler, err := handlers.NewSavedSearchHandler()
	if err != nil {
		log.Printf("⚠️  Saved search handler no disponible: %v", err)
	}

	// Inicializar handler de historial de vistas
	viewHistoryHandler, err := handlers.NewViewHistoryHandler()
	if err != nil {
		log.Printf("⚠️  View history handler no disponible: %v", err)
	}

	// Inicializar handler de pagos QR
	paymentHandler, err := handlers.NewPaymentHandler()
	if err != nil {
		log.Printf("⚠️  Payment handler no disponible: %v", err)
		// No es fatal, continuar sin payment handler
	} else {
		log.Println("✅ Payment handler (Banco MC4 QR) inicializado")
	}

	// Inicializar handler de comprobantes de pago
	proofHandler, err := handlers.NewProofHandler()
	if err != nil {
		log.Printf("⚠️  Proof handler no disponible: %v", err)
	} else {
		log.Println("✅ Proof handler (comprobantes de pago) inicializado")
	}

	// Inicializar handler de WebSocket (para actualizaciones en tiempo real)
	var wsHandler *handlers.WebSocketHandler
	if propertyHandler != nil && propertyHandler.Redis != nil {
		wsHandler = handlers.NewWebSocketHandler(propertyHandler.Redis)
		log.Println("✅ WebSocket handler inicializado para actualizaciones en tiempo real")
	}

	// Inicializar handler de asesores (para dashboard de asesores)
	advisorDashboardHandler, err := handlers.NewAdvisorHandler()
	if err != nil {
		log.Printf("⚠️  Advisor dashboard handler no disponible: %v", err)
	} else {
		log.Println("✅ Advisor dashboard handler inicializado")
	}

	// Inicializar handler de CRM
	crmHandler, err := handlers.NewCrmHandler()
	if err != nil {
		log.Printf("⚠️  CRM handler no disponible: %v", err)
	} else {
		log.Println("✅ CRM handler inicializado")
	}

	// Inicializar handler de Lead Scoring
	leadScoringHandler, err := handlers.NewLeadScoringHandler()
	if err != nil {
		log.Printf("⚠️  Lead Scoring handler no disponible: %v", err)
	} else {
		log.Println("✅ Lead Scoring handler inicializado")
	}

	// Inicializar handler de documentos
	documentHandler, err := handlers.NewDocumentHandler(cfg)
	if err != nil {
		log.Printf("⚠️  Document handler no disponible: %v", err)
	} else {
		log.Println("✅ Document handler inicializado")
	}

	// Inicializar handler de Motor de Recomendaciones
	recommendationHandler, err := handlers.NewRecommendationHandler()
	if err != nil {
		log.Printf("⚠️  Recommendation handler no disponible: %v", err)
	} else {
		log.Println("✅ Motor de Recomendaciones inicializado")
	}

	// Rutas de salud
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API is running",
		})
	})

	app.Get("/api", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Bienvenido a la API de Inmobiliaria Bolivia",
			"version": "1.0.0",
		})
	})

	// Servir archivos estáticos de uploads (comprobantes de pago, etc.)
	app.Static("/uploads", "./uploads")

	// API Routes
	api := app.Group("/api")

	// Rutas públicas de autenticación
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Ruta de login estricta para administradores
	auth.Post("/admin/login", authHandler.AdminLogin)

	// Rutas de Google OAuth (PÚBLICAS - sin autenticación)
	if googleAuthHandler != nil {
		app.Get("/api/auth/google", googleAuthHandler.GoogleLogin)
		app.Get("/api/auth/google/callback", googleAuthHandler.GoogleCallback)
	}

	// Rutas protegidas de autenticación (aplicamos middleware individualmente)
	api.Get("/auth/me", middleware.AuthMiddleware(), authHandler.GetMe)
	api.Put("/auth/profile", middleware.AuthMiddleware(), authHandler.UpdateProfile) // Actualizar perfil
	api.Put("/auth/datos-publicos", middleware.AuthMiddleware(), authHandler.UpdateDatosPublicos) // Actualizar datos públicos

	// Ruta de completar perfil de Google (protegida - requiere token)
	if googleAuthHandler != nil {
		api.Post("/auth/google/complete-profile", middleware.AuthMiddleware(), googleAuthHandler.CompleteGoogleProfile)
	}

	// Rutas de administración compartidas (admin + asesor)
	adminShared := api.Group("/admin")
	adminShared.Use(middleware.AuthMiddleware())
	adminShared.Use(middleware.AdminOrAdvisorMiddleware()) // Permite admin y asesores

	// Ruta de Analytics Dashboard (admin y asesores pueden ver)
	if analyticsHandler != nil {
		adminShared.Get("/analytics", analyticsHandler.GetAnalytics)
	}

	// Rutas de Reportes Financieros (admin y asesores pueden ver, pero con filtros)
	if reportsHandler != nil {
		adminShared.Get("/reports/overview", reportsHandler.GetFinancialOverview)
		adminShared.Get("/reports/income", reportsHandler.GetIncomeReport)
		adminShared.Get("/reports/subscriptions", reportsHandler.GetSubscriptionMetrics)
	}

	// Rutas de administración de propiedades (admin y asesores pueden ver)
	// IMPORTANTE: El orden de las rutas importa. Las rutas MÁS ESPECÍFICAS deben ir ANTES.
	if adminPropertyHandler != nil {
		// Rutas básicas (sin parámetros)
		adminShared.Get("/properties", adminPropertyHandler.ListProperties)
		adminShared.Post("/properties", adminPropertyHandler.CreateProperty)

		// Rutas ESPECÍFICAS (van antes que /:id para evitar conflictos)
		adminShared.Get("/properties/stats", adminPropertyHandler.GetPropertiesStats)

		// Rutas con parámetro :id
		adminShared.Get("/properties/:id", adminPropertyHandler.GetProperty)
		adminShared.Put("/properties/:id", adminPropertyHandler.UpdateProperty)
		adminShared.Delete("/properties/:id", adminPropertyHandler.DeleteProperty)

		// Rutas con :id y sub-rutas (van después)
		adminShared.Patch("/properties/:id/status", adminPropertyHandler.UpdatePropertyStatus)
		adminShared.Patch("/properties/:id/featured", adminPropertyHandler.ToggleFeatured)
		adminShared.Patch("/properties/:id/advisor", adminPropertyHandler.AssignAdvisor)
	}

	// Rutas de administración exclusivas (SOLO admin)
	adminExclusive := api.Group("/admin")
	adminExclusive.Use(middleware.AuthMiddleware())
	adminExclusive.Use(middleware.AdminMiddleware()) // Solo permite admin

	// Rutas de gestión de usuarios (solo admin)
	if userHandler != nil {
		adminExclusive.Get("/users", userHandler.GetUsers)
		adminExclusive.Post("/users", userHandler.CreateUser)
		adminExclusive.Put("/users/:id", userHandler.UpdateUser)
		adminExclusive.Patch("/users/:id/toggle-active", userHandler.ToggleUserActive)
		adminExclusive.Delete("/users/:id", userHandler.DeleteUser)
	}

	// Ruta de migración de datos (solo admin)
	generalAdminHandler, err := handlers.NewAdminHandler()
	if err != nil {
		log.Printf("⚠️  Admin handler general no disponible: %v", err)
	} else {
		adminExclusive.Post("/migrate-data", generalAdminHandler.MigrateData)
	}

	// Gestión de Planes (solo admin)
	if planHandler != nil {
		adminExclusive.Get("/planes", planHandler.GetPlans)
		adminExclusive.Post("/planes", planHandler.CreatePlan)
		adminExclusive.Put("/planes/:id", planHandler.UpdatePlan)
		adminExclusive.Delete("/planes/:id", planHandler.DeletePlan)
	}

	// Rutas Públicas de Planes (solo activos)
	if planHandler != nil {
		api.Get("/planes", planHandler.GetActivePlans)
		api.Get("/planes/:id", planHandler.GetPlan)
	}

	// Revisión de comprobantes de pago (verificación de admin dentro del handler)
	// NOTA: Estas rutas están fuera de adminExclusive para usar AuthMiddleware + verificación manual
	if proofHandler != nil {
		api.Get("/admin/payments/pending", middleware.AuthMiddleware(), proofHandler.GetPendingProofs)
		api.Get("/admin/payments/pending/count", middleware.AuthMiddleware(), proofHandler.GetPendingProofsCount)
		api.Post("/admin/payments/:id/review", middleware.AuthMiddleware(), proofHandler.ReviewProof)
	}

	// Rutas de Pagos QR (protegidas - requieren autenticación)
	if planHandler != nil {
		api.Get("/planes", planHandler.GetActivePlans)
		api.Get("/planes/:id", planHandler.GetPlan)
	}

	// Rutas de Pagos QR (protegidas - requieren autenticación)
	if paymentHandler != nil {
		// Crear pago QR (protegido)
		api.Post("/payments/create", middleware.AuthMiddleware(), paymentHandler.CreatePayment)

		// Verificar estado de pago (protegido)
		api.Get("/payments/:id/status", middleware.AuthMiddleware(), paymentHandler.GetPaymentStatus)

		// Webhook del banco (PÚBLICO - sin autenticación, el banco llama aquí)
		api.Post("/payments/webhook", paymentHandler.WebhookPayment)

		// Cancelar pago (protegido)
		api.Delete("/payments/:id", middleware.AuthMiddleware(), paymentHandler.CancelPayment)

		// Mis pagos (protegido)
		api.Get("/payments", middleware.AuthMiddleware(), paymentHandler.GetMyPayments)

		// Mi suscripción (protegido)
		api.Get("/payments/subscription", middleware.AuthMiddleware(), paymentHandler.GetMySubscription)

		// Mis suscripciones (protegido) - Obtener todas las suscripciones
		api.Get("/payments/subscriptions", middleware.AuthMiddleware(), paymentHandler.GetMySubscriptions)

		// Seleccionar suscripción (protegido)
		api.Post("/payments/subscriptions/:id/select", middleware.AuthMiddleware(), paymentHandler.SelectSubscription)

		// SIMULAR pago exitoso (SOLO PARA DESARROLLO/PRUEBAS)
		api.Post("/payments/:id/simulate-success", middleware.AuthMiddleware(), paymentHandler.SimulatePaymentSuccess)

		// Subir comprobante de pago (protegido)
		if proofHandler != nil {
			api.Post("/payments/proof", middleware.AuthMiddleware(), proofHandler.UploadProof)
		}
	}

	// Rutas de gestión de asesores (solo admin)
	if advisorHandler != nil {
		adminExclusive.Get("/advisors", advisorHandler.GetAdvisors)
		adminExclusive.Get("/advisors/:id", advisorHandler.GetAdvisor)
		adminExclusive.Post("/advisors", advisorHandler.CreateAdvisor)
		adminExclusive.Put("/advisors/:id", advisorHandler.UpdateAdvisor)
		adminExclusive.Delete("/advisors/:id", advisorHandler.DeleteAdvisor)
		adminExclusive.Patch("/advisors/:id/toggle-active", advisorHandler.ToggleAdvisorActive)
	}

	// Rutas de asesores asignados a propiedades (solo admin)
	if advisorHandler != nil {
		// Nuevas rutas para múltiples asesores por propiedad
		adminExclusive.Get("/properties/:id/advisors", advisorHandler.GetPropertyAdvisors)
		adminExclusive.Post("/properties/:id/advisors", advisorHandler.AssignAdvisorToProperty)
		adminExclusive.Delete("/properties/:id/advisors/:advisorId", advisorHandler.RemoveAdvisorFromProperty)
	}

	// Rutas de imágenes (DEBEN IR ANTES que /properties/:id para evitar conflictos)
	if imageHandler != nil {
		// Rutas protegidas para subir imágenes
		api.Post("/imagenes/upload", middleware.AuthMiddleware(), imageHandler.UploadImage)
		api.Post("/imagenes/upload-base64", middleware.AuthMiddleware(), imageHandler.UploadImagesFromBase64)

		// Ruta pública para servir imágenes (ESTA PRIMERO por tener :filename)
		api.Get("/imagenes/:filename", imageHandler.ServeImage)

		// Rutas para avatares (separadas de las imágenes de propiedades)
		api.Post("/avatars/upload", middleware.AuthMiddleware(), imageHandler.UploadAvatar)
		api.Get("/avatars/:filename", imageHandler.ServeAvatar)
	}

	// Rutas de propiedades
	if propertyHandler != nil {
		// IMPORTANTE: En Fiber, el orden de las rutas importa.
		// Las rutas MÁS ESPECÍFICAS deben ir ANTES que las rutas con parámetros.

		// Rutas ESPECÍFICAS (protegidas) - van primero
		api.Get("/properties/my", middleware.AuthMiddleware(), propertyHandler.GetUserProperties)         // Mis propiedades (requiere login)
		api.Patch("/properties/:id/status", middleware.AuthMiddleware(), propertyHandler.UpdatePropertyStatus) // Actualizar estado (requiere login)

		// Rutas PÚBLICAS (van después)
		api.Get("/properties", propertyHandler.ListProperties)              // Listar propiedades (público)
		api.Get("/properties/search", propertyHandler.SearchProperties)    // Búsqueda full-text (público)
		api.Get("/properties/:id", propertyHandler.GetProperty)            // Ver detalle de propiedad (público)

		// Rutas protegidas restantes
		api.Post("/properties", middleware.AuthMiddleware(), middleware.SubscriptionMiddleware(), propertyHandler.CreateProperty)              // Crear propiedad (requiere login + suscripción)
		api.Delete("/properties/:id", middleware.AuthMiddleware(), propertyHandler.DeleteProperty)        // Eliminar propiedad (requiere login)
	}

	// Rutas de favoritos
	if favoriteHandler != nil {
		api.Post("/favorites", middleware.AuthMiddleware(), favoriteHandler.AddFavorite)
		api.Get("/favorites", middleware.AuthMiddleware(), favoriteHandler.GetFavorites)
		api.Delete("/favorites/:id", middleware.AuthMiddleware(), favoriteHandler.RemoveFavorite)
		api.Get("/favorites/check/:propertyId", middleware.AuthMiddleware(), favoriteHandler.CheckFavorite)
	}

	// Rutas de Motor de Recomendaciones (protegidas - requieren autenticación)
	if recommendationHandler != nil {
		api.Get("/recommendations", middleware.AuthMiddleware(), recommendationHandler.GetRecommendations)
		api.Get("/recommendations/user/:userId", middleware.AuthMiddleware(), recommendationHandler.GetRecommendationsById)
	}

	// Rutas de búsquedas guardadas y notificaciones
	if savedSearchHandler != nil {
		api.Post("/saved-searches", middleware.AuthMiddleware(), savedSearchHandler.CreateSavedSearch)
		api.Get("/saved-searches", middleware.AuthMiddleware(), savedSearchHandler.GetSavedSearches)
		api.Delete("/saved-searches/:id", middleware.AuthMiddleware(), savedSearchHandler.DeleteSavedSearch)
		api.Patch("/saved-searches/:id/toggle", middleware.AuthMiddleware(), savedSearchHandler.ToggleSavedSearch)
		api.Get("/saved-searches/:id/matches", middleware.AuthMiddleware(), savedSearchHandler.GetSavedSearchMatches)

		api.Get("/notifications", middleware.AuthMiddleware(), savedSearchHandler.GetNotifications)
		api.Patch("/notifications/read", middleware.AuthMiddleware(), savedSearchHandler.MarkNotificationsAsRead)
		api.Patch("/notifications/:id/read", middleware.AuthMiddleware(), savedSearchHandler.MarkOneNotificationAsRead)
	}

	// Rutas de documentos
	if documentHandler != nil {
		api.Post("/documents", middleware.AuthMiddleware(), documentHandler.UploadDocument)
		api.Get("/documents", middleware.AuthMiddleware(), documentHandler.GetDocuments)
		api.Get("/documents/public", documentHandler.GetPublicDocuments)
		api.Get("/documents/property/:id", documentHandler.GetPropertyDocuments)
		api.Get("/documents/all", middleware.AuthMiddleware(), middleware.AdminMiddleware(), documentHandler.GetAllDocuments) // Admin only
		api.Delete("/documents/:id", middleware.AuthMiddleware(), documentHandler.DeleteDocument)
	}

	// Rutas de historial de vistas
	if viewHistoryHandler != nil {
		api.Post("/view-history", middleware.AuthMiddleware(), viewHistoryHandler.RecordView)
		api.Get("/view-history", middleware.AuthMiddleware(), viewHistoryHandler.GetViewHistory)
		api.Delete("/view-history", middleware.AuthMiddleware(), viewHistoryHandler.ClearViewHistory)
		api.Delete("/view-history/:id", middleware.AuthMiddleware(), viewHistoryHandler.DeleteViewHistoryItem)
	}

	// Rutas de dashboard de asesores (protegidas - requieren autenticación de asesor)
	if advisorDashboardHandler != nil {
		advisor := api.Group("/advisor")
		advisor.Use(middleware.AuthMiddleware())     // Requiere autenticación
		advisor.Use(middleware.AdvisorMiddleware()) // Requiere rol de asesor

		// Obtener propiedades del asesor (con filtro opcional de estado)
		advisor.Get("/properties", advisorDashboardHandler.GetProperties)

		// Activar propiedad (solo el asesor asignado puede hacerlo)
		advisor.Patch("/properties/:id/activate", advisorDashboardHandler.ActivateProperty)

		// Obtener estadísticas del asesor
		advisor.Get("/stats", advisorDashboardHandler.GetStats)
	}

	// Rutas de CRM (protegidas - requieren autenticación de admin o asesor)
	if crmHandler != nil {
		crm := api.Group("/crm")
		crm.Use(middleware.AuthMiddleware())              // Requiere autenticación
		crm.Use(middleware.AdminOrAdvisorMiddleware())    // Requiere rol de admin o asesor

		// Clientes
		crm.Post("/clients", crmHandler.CreateClient)                      // Crear cliente
		crm.Get("/clients", crmHandler.ListClients)                        // Listar clientes
		crm.Get("/clients/stats", crmHandler.GetFunnelStats)                // Estadísticas embudo
		crm.Get("/clients/upcoming", crmHandler.GetUpcomingContacts)        // Próximos contactos
		crm.Get("/clients/:id", crmHandler.GetClient)                       // Ver cliente
		crm.Put("/clients/:id", crmHandler.UpdateClient)                    // Actualizar cliente
		crm.Delete("/clients/:id", crmHandler.DeleteClient)                // Eliminar cliente

		// Interacciones
		crm.Post("/clients/:id/interactions", crmHandler.CreateInteraction)  // Crear interacción
		crm.Get("/clients/:id/interactions", crmHandler.GetClientInteractions) // Listar interacciones
	}

	// Rutas de Lead Scoring (protegidas - requieren autenticación de admin o asesor)
	if leadScoringHandler != nil {
		crm := api.Group("/crm")
		crm.Use(middleware.AuthMiddleware())              // Requiere autenticación
		crm.Use(middleware.AdminOrAdvisorMiddleware())    // Requiere rol de admin o asesor

		// Lead Scoring de clientes
		crm.Get("/clients/:id/score", leadScoringHandler.GetLeadScore)                    // Obtener score de cliente
		crm.Post("/clients/:id/score/recalculate", leadScoringHandler.CalculateLeadScore) // Recalcular score

		// Lead Scoring general
		crm.Get("/leads/scores", leadScoringHandler.GetAllLeadScores)          // Todos los scores
		crm.Get("/leads/top", leadScoringHandler.GetTopLeads)                  // Top leads
		crm.Get("/leads/by-level/:level", leadScoringHandler.GetLeadsByLevel)  // Leads por nivel
		crm.Get("/leads/scores/stats", leadScoringHandler.GetScoreStats)       // Estadísticas de scoring
		crm.Post("/leads/scores/recalculate-all", leadScoringHandler.RecalculateAllScores) // Recalcular todos
	}

	// 🚀 WebSocket para actualizaciones en tiempo real
	if wsHandler != nil {
		// WebSocket upgrade endpoint
		app.Get("/ws/properties", websocket.New(wsHandler.HandlePropertyUpdates))
	}

	// 🔒 Middleware NotFound (debe ir al final)
	app.Use(middleware.NotFoundMiddleware())

	// Iniciar servidor
	log.Println("🚀 Servidor corriendo en http://localhost:3001")
	log.Println("📊 Endpoints disponibles:")
	log.Println("  POST   /api/auth/register        - Registro de usuario")
	log.Println("  POST   /api/auth/login           - Login de usuario")
	log.Println("  GET    /api/auth/me              - Obtener usuario actual (requiere token)")
	log.Println("  PUT    /api/auth/profile         - Actualizar perfil (requiere token)")
	log.Println("  PUT    /api/auth/datos-publicos  - Actualizar datos públicos (requiere token)")
	log.Println("  GET    /api/properties           - Listar propiedades (público)")
	log.Println("  GET    /api/properties/search    - Búsqueda full-text (público)")
	log.Println("  GET    /api/properties/:id       - Obtener propiedad (público)")
	log.Println("  POST   /api/properties           - Crear propiedad (requiere token)")
	log.Println("  GET    /api/properties/my        - Mis propiedades (requiere token)")
	log.Println("  PATCH  /api/properties/:id/status - Actualizar estado (requiere token)")
	log.Println("  DELETE /api/properties/:id       - Eliminar propiedad (requiere token)")
	if wsHandler != nil {
		log.Println("  WS     /ws/properties          - Actualizaciones en tiempo real (WebSocket)")
	}

	// 🧹 Scheduler de limpieza de imágenes (DESHABILITADO - Ahora se eliminan inmediatamente)
	// El job está comentado porque las imágenes se eliminan automáticamente al eliminar la propiedad
	/*
	retentionDays := 30 // Días de retención de imágenes
	intervalHours := 24 // Ejecutar cada 24 horas

	// Leer desde variables de entorno si existen
	if rd := os.Getenv("IMAGE_CLEANUP_RETENTION_DAYS"); rd != "" {
		if val, err := strconv.Atoi(rd); err == nil && val > 0 {
			retentionDays = val
		}
	}
	if ih := os.Getenv("IMAGE_CLEANUP_INTERVAL_HOURS"); ih != "" {
		if val, err := strconv.Atoi(ih); err == nil && val > 0 {
			intervalHours = val
		}
	}

	// Iniciar job de limpieza en background
	cleanupJob, err := jobs.NewCleanupJob()
	if err != nil {
		log.Printf("⚠️  Error iniciando job de limpieza: %v\n", err)
	} else {
		go cleanupJob.StartCleanupScheduler(retentionDays, intervalHours)
		log.Printf("🧹 Job de limpieza iniciado: Retención=%d días, Intervalo=%d horas\n", retentionDays, intervalHours)
	}
	*/

	// 🔍 Iniciar Worker de Búsquedas Guardadas en background
	savedSearchWorker, err := jobs.NewSavedSearchWorker()
	if err != nil {
		log.Printf("⚠️  Error iniciando worker de búsquedas guardadas: %v\n", err)
	} else {
		// Ejecutar verificación cada 1 minuto en desarrollo local para feedback inmediato
		go savedSearchWorker.StartWorkerScheduler(1)
		log.Println("🔔 Worker de Búsquedas Guardadas iniciado (Intervalo = 1 minuto)")
	}

	// ⏰ Scheduler de eliminación de propiedades expiradas
	cleanupJob, err := jobs.NewCleanupJob()
	if err != nil {
		log.Printf("⚠️  Error iniciando job de expiración: %v\n", err)
	} else {
		// Verificar cada 1 minuto (ajustable con variable de entorno)
		checkInterval := 1 // minutos
		if ci := os.Getenv("EXPIRATION_CHECK_INTERVAL_MINUTES"); ci != "" {
			if val, err := strconv.Atoi(ci); err == nil && val > 0 {
				checkInterval = val
			}
		}
		go cleanupJob.StartExpirationScheduler(checkInterval)
		log.Printf("⏰ Job de expiración de propiedades iniciado: Verificando cada %d minuto(s)\n", checkInterval)
	}

	// 📊 Scheduler de Lead Scoring (recálculo automático de scores)
	leadScoringJob, err := jobs.NewLeadScoringJob()
	if err != nil {
		log.Printf("⚠️  Error iniciando job de lead scoring: %v\n", err)
	} else {
		// Ejecutar cada 24 horas (ajustable con variable de entorno)
		go leadScoringJob.StartSchedulerFromEnv()
		log.Println("📊 Job de Lead Scoring iniciado: Recalculando scores cada 24 horas")
	}

	log.Fatal(app.Listen(":3001"))
}
