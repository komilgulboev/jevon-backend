// @title           Jevon CRM API
// @version         1.0
// @description     Система управления мебельным цехом
// @host            localhost:8181
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "jevon/docs"
	"jevon/internal/auth"
	"jevon/internal/config"
	"jevon/internal/db"
	"jevon/internal/handlers"
	"jevon/internal/middleware"
	"jevon/internal/repository"
	"jevon/internal/storage"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.Server.GinMode)

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database, cfg.DB.Name); err != nil {
		log.Fatalf("❌ Migrations failed: %v", err)
	}

	minioSvc, err := storage.NewMinIOService(cfg.MinIO)
	if err != nil {
		log.Printf("⚠️  MinIO connection failed: %v (file uploads disabled)", err)
		minioSvc = nil
	} else {
		log.Printf("✅ MinIO connected — %s", cfg.MinIO.Endpoint)
	}

	authSvc := auth.NewService(cfg.JWT)

	// ── Repositories ─────────────────────────────────────
	userRepo           := repository.NewUserRepo(database)
	projectRepo        := repository.NewProjectRepo(database)
	taskRepo           := repository.NewTaskRepo(database)
	dashRepo           := repository.NewDashboardRepo(database)
	pipelineRepo       := repository.NewPipelineRepo(database)
	orderRepo          := repository.NewOrderRepo(database)
	estimateRepo       := repository.NewEstimateRepo(database)
	detailEstimateRepo := repository.NewDetailEstimateRepo(database)
	warehouseRepo      := repository.NewWarehouseRepo(database)

	// ── Handlers ─────────────────────────────────────────
	authH           := handlers.NewAuthHandler(userRepo, authSvc)
	usersH          := handlers.NewUsersHandler(userRepo)
	dashH           := handlers.NewDashboardHandler(dashRepo)
	projH           := handlers.NewProjectsHandler(projectRepo)
	tasksH          := handlers.NewTasksHandler(taskRepo)
	pipelineH       := handlers.NewPipelineHandler(pipelineRepo)
	uploadH         := handlers.NewUploadHandler(minioSvc, pipelineRepo)
	uploadH.SetOrderRepo(orderRepo)
	orderH          := handlers.NewOrderHandler(orderRepo)
	estimateH       := handlers.NewEstimateHandler(estimateRepo)
	detailEstimateH := handlers.NewDetailEstimateHandler(detailEstimateRepo)
	warehouseH      := handlers.NewWarehouseHandler(warehouseRepo)

	// ── Router ───────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")

	// ── Public ───────────────────────────────────────────
	api.POST("/auth/login",   authH.Login)
	api.POST("/auth/refresh", authH.Refresh)
	api.POST("/auth/logout",  authH.Logout)

	// ── Protected ────────────────────────────────────────
	p := api.Group("/")
	p.Use(middleware.RequireAuth(authSvc))

	// Dashboard
	p.GET("/dashboard/stats", dashH.Stats)

	// Users
	p.GET("/users",                     middleware.RequireRole("admin", "supervisor"), usersH.List)
	p.GET("/users/:id",                 middleware.RequireRole("admin", "supervisor"), usersH.Get)
	p.POST("/users",                    middleware.RequireRole("admin"),               usersH.Create)
	p.PATCH("/users/:id/toggle-active", middleware.RequireRole("admin"),               usersH.ToggleActive)
	p.POST("/users/avatar",             uploadH.UploadAvatar)

	// Projects
	p.GET("/projects",                middleware.RequireRole("admin", "supervisor", "manager"), projH.List)
	p.POST("/projects",               middleware.RequireRole("admin", "supervisor", "manager"), projH.Create)
	p.PATCH("/projects/:project_id",  middleware.RequireRole("admin", "supervisor", "manager"), projH.Update)
	p.DELETE("/projects/:project_id", middleware.RequireRole("admin"),                          projH.Delete)

	// Tasks
	p.GET("/tasks",              tasksH.List)
	p.POST("/tasks",             middleware.RequireRole("admin", "supervisor"), tasksH.Create)
	p.PATCH("/tasks/:id",        tasksH.Update)
	p.PATCH("/tasks/:id/status", tasksH.UpdateStatus)
	p.DELETE("/tasks/:id",       middleware.RequireRole("admin", "supervisor"), tasksH.Delete)

	// Каталог операций
	p.GET("/catalog/operations", pipelineH.CatalogList)

	// Этапы проекта
	p.GET("/projects/:project_id/stages",                     pipelineH.StagesList)
	p.GET("/projects/:project_id/stages/:stage_id",           pipelineH.StageGet)
	p.PATCH("/projects/:project_id/stages/:stage_id",         pipelineH.StageUpdate)
	p.POST("/projects/:project_id/stages/:stage_id/complete", pipelineH.StageComplete)

	// Операции
	p.GET("/projects/:project_id/operations",                  pipelineH.OperationsByProject)
	p.POST("/projects/:project_id/operations",                 pipelineH.OperationCreate)
	p.GET("/projects/:project_id/stages/:stage_id/operations", pipelineH.OperationsList)
	p.PATCH("/projects/:project_id/operations/:operation_id",  pipelineH.OperationUpdate)
	p.DELETE("/projects/:project_id/operations/:operation_id",
		middleware.RequireRole("admin", "supervisor"), pipelineH.OperationDelete)

	// Материалы проектов (pipeline)
	p.GET("/projects/:project_id/materials",                                          pipelineH.MaterialsByProject)
	p.GET("/projects/:project_id/operations/:operation_id/materials",                 pipelineH.MaterialsList)
	p.POST("/projects/:project_id/operations/:operation_id/materials",                pipelineH.MaterialCreate)
	p.DELETE("/projects/:project_id/operations/:operation_id/materials/:material_id",
		middleware.RequireRole("admin", "supervisor"), pipelineH.MaterialDelete)

	// Файлы этапов проекта
	p.GET("/projects/:project_id/stages/:stage_id/files",   pipelineH.FilesList)
	p.POST("/projects/:project_id/stages/:stage_id/upload", uploadH.UploadStageFiles)
	p.POST("/projects/:project_id/stages/:stage_id/files",  pipelineH.FileCreate)
	p.DELETE("/projects/:project_id/stages/:stage_id/files/:file_id",
		middleware.RequireRole("admin", "supervisor", "designer"), pipelineH.FileDelete)

	// История проекта
	p.GET("/projects/:project_id/history", pipelineH.History)

	// Удаление файла из MinIO
	p.DELETE("/files", middleware.RequireRole("admin", "supervisor", "designer"), uploadH.DeleteFile)

	// ════════════════════════════════════════════════════
	// МОДУЛЬ ЗАКАЗОВ
	// ════════════════════════════════════════════════════

	// Клиенты
	p.GET("/clients",       orderH.ClientList)
	p.POST("/clients",      middleware.RequireRole("admin", "supervisor", "manager"), orderH.ClientCreate)
	p.PATCH("/clients/:id", middleware.RequireRole("admin", "supervisor", "manager"), orderH.ClientUpdate)

	// Прайслист
	p.GET("/price-list",       orderH.PriceList)
	p.PATCH("/price-list/:id", middleware.RequireRole("admin", "supervisor"), orderH.PriceUpdate)

	// Каталог материалов
	p.GET("/materials/catalog", orderH.MaterialsCatalog)

	// Каталог услуг сметы
	p.GET("/estimate/catalog",        estimateH.CatalogList)
	p.GET("/estimate/catalog/flat",   estimateH.CatalogFlat)
	p.POST("/estimate/catalog",       middleware.RequireRole("admin", "supervisor"), estimateH.CatalogCreate)
	p.PATCH("/estimate/catalog/:id",  middleware.RequireRole("admin", "supervisor"), estimateH.CatalogUpdate)
	p.DELETE("/estimate/catalog/:id", middleware.RequireRole("admin", "supervisor"), estimateH.CatalogDelete)
	p.GET("/estimate/colors",         estimateH.ColorList)

	// Заказы
	p.GET("/orders",              orderH.OrderList)
	p.GET("/orders/stats",        orderH.OrderStats)
	p.GET("/orders/labels",       orderH.Labels)
	p.POST("/orders",             middleware.RequireRole("admin", "supervisor", "manager"), orderH.OrderCreate)
	p.GET("/orders/:order_id",    orderH.OrderGet)
	p.PATCH("/orders/:order_id",  middleware.RequireRole("admin", "supervisor", "manager"), orderH.OrderUpdate)
	p.DELETE("/orders/:order_id", middleware.RequireRole("admin", "supervisor"), orderH.OrderCancel)

	// Этапы заказа
	p.GET("/orders/:order_id/stages",                     orderH.StagesList)
	p.PATCH("/orders/:order_id/stages/:stage_id",         orderH.StageUpdate)
	p.POST("/orders/:order_id/stages/:stage_id/complete", orderH.StageComplete)

	// Файлы этапов заказа
	p.GET("/orders/:order_id/stages/:stage_id/files",   pipelineH.FilesList)
	p.POST("/orders/:order_id/stages/:stage_id/upload", uploadH.UploadStageFiles)
	p.POST("/orders/:order_id/stages/:stage_id/files",  pipelineH.FileCreate)
	p.DELETE("/orders/:order_id/stages/:stage_id/files/:file_id",
		middleware.RequireRole("admin", "supervisor", "designer"), pipelineH.FileDelete)

	// Оплаты
	p.GET("/orders/:order_id/payments",  orderH.PaymentsList)
	p.POST("/orders/:order_id/payments", middleware.RequireRole("admin", "supervisor", "manager"), orderH.PaymentCreate)

	// Комментарии
	p.GET("/orders/:order_id/comments",  orderH.CommentsList)
	p.POST("/orders/:order_id/comments", orderH.CommentCreate)

	// История заказа
	p.GET("/orders/:order_id/history", orderH.History)

	// Материалы заказа (накладная)
	p.GET("/orders/:order_id/materials",                 orderH.MaterialsList)
	p.POST("/orders/:order_id/materials",                orderH.MaterialCreate)
	p.DELETE("/orders/:order_id/materials/:material_id",
		middleware.RequireRole("admin", "supervisor", "manager"), orderH.MaterialDelete)

	// Смета заказа (распил — EstimateTable)
	p.GET("/orders/:order_id/estimate",  estimateH.EstimateGet)
	p.POST("/orders/:order_id/estimate", estimateH.EstimateSave)

	// Расходы заказа цеха
	p.GET("/orders/:order_id/expenses",                orderH.ExpensesList)
	p.POST("/orders/:order_id/expenses",               middleware.RequireRole("admin", "supervisor", "manager"), orderH.ExpenseCreate)
	p.DELETE("/orders/:order_id/expenses/:expense_id", middleware.RequireRole("admin", "supervisor", "manager"), orderH.ExpenseDelete)

	// Детализированная смета (ЧПУ, Покраска, Мягкая мебель, Распил)
	p.GET("/orders/:order_id/detail-estimate",                  detailEstimateH.GetEstimate)
	p.POST("/orders/:order_id/detail-estimate",                 detailEstimateH.SaveSection)
	p.DELETE("/orders/:order_id/detail-estimate/:service_type", detailEstimateH.DeleteSection)

	// ════════════════════════════════════════════════════
	// МОДУЛЬ СКЛАДА
	// ════════════════════════════════════════════════════

	// Единицы измерения
	p.GET("/warehouse/units", warehouseH.UnitList)

	// Категории номенклатуры
	p.GET("/warehouse/categories", warehouseH.CategoryList)

	// Номенклатура
	p.GET("/warehouse/items",        warehouseH.ItemList)
	p.GET("/warehouse/items/:id",    warehouseH.ItemGet)
	p.POST("/warehouse/items",       middleware.RequireRole("admin", "supervisor"), warehouseH.ItemCreate)
	p.PUT("/warehouse/items/:id",    middleware.RequireRole("admin", "supervisor"), warehouseH.ItemUpdate)
	p.DELETE("/warehouse/items/:id", middleware.RequireRole("admin", "supervisor"), warehouseH.ItemDelete)

	// Поставщики
	p.GET("/warehouse/suppliers",        warehouseH.SupplierList)
	p.GET("/warehouse/suppliers/:id",    warehouseH.SupplierGet)
	p.POST("/warehouse/suppliers",       middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierCreate)
	p.PATCH("/warehouse/suppliers/:id",  middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierUpdate)
	p.DELETE("/warehouse/suppliers/:id", middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierDelete)

	// Приходные накладные
	p.GET("/warehouse/receipts",                       warehouseH.ReceiptList)
	p.GET("/warehouse/receipts/:id",                   warehouseH.ReceiptGet)
	p.POST("/warehouse/receipts",                      middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptCreate)
	p.PATCH("/warehouse/receipts/:id",                 middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptUpdate)
	p.DELETE("/warehouse/receipts/:id",                middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptDelete)
	p.POST("/warehouse/receipts/:id/items",            middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptItemAdd)
	p.DELETE("/warehouse/receipts/:id/items/:item_id", middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptItemDelete)

	// Платежи по накладным
	p.GET("/warehouse/receipts/:id/payments",                middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentList)
	p.POST("/warehouse/receipts/:id/payments",               middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentCreate)
	p.DELETE("/warehouse/receipts/:id/payments/:payment_id", middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentDelete)

	 
	// Платежи поставщику (общий расчёт)
	p.GET("/warehouse/suppliers/:id/payments",                middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentHistory)
	p.POST("/warehouse/suppliers/:id/payments",               middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentCreate)
	p.DELETE("/warehouse/suppliers/:id/payments/:payment_id", middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentDelete)
	
	log.Printf("🚀 Server running on :%s", cfg.Server.Port)
	log.Printf("📖 Swagger UI: http://localhost:%s/swagger/index.html", cfg.Server.Port)
	r.Run(":" + cfg.Server.Port)
}	