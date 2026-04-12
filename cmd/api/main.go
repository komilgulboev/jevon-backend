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

	// ── Repositories ─────────────────────────────────────────
	userRepo           := repository.NewUserRepo(database)
	projectRepo        := repository.NewProjectRepo(database)
	taskRepo           := repository.NewTaskRepo(database)
	dashRepo           := repository.NewDashboardRepo(database)
	pipelineRepo       := repository.NewPipelineRepo(database)
	orderRepo          := repository.NewOrderRepo(database)
	estimateRepo       := repository.NewEstimateRepo(database)
	detailEstimateRepo := repository.NewDetailEstimateRepo(database)
	warehouseRepo      := repository.NewWarehouseRepo(database)
	clientBalanceRepo  := repository.NewClientBalanceRepo(database)
	expenseRepo        := repository.NewWorkshopExpenseRepo(database)
	assigneeRepo       := repository.NewStageAssigneeRepo(database)
	outgoingInvoiceRepo := repository.NewOutgoingInvoiceRepo(database)


	// ── Handlers ─────────────────────────────────────────────
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
	clientBalanceH  := handlers.NewClientBalanceHandler(clientBalanceRepo)
	expenseH        := handlers.NewWorkshopExpenseHandler(expenseRepo)
	assigneeH       := handlers.NewStageAssigneeHandler(assigneeRepo)
	outgoingInvoiceH   := handlers.NewOutgoingInvoiceHandler(outgoingInvoiceRepo)
	// ── Router ───────────────────────────────────────────────
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

	// ════════════════════════════════════════════════════════════
	// PUBLIC — без авторизации
	// ════════════════════════════════════════════════════════════
	api.POST("/auth/login",   authH.Login)
	api.POST("/auth/refresh", authH.Refresh)
	api.POST("/auth/logout",  authH.Logout)

	// ── Protected — все роуты ниже требуют JWT токен ──────────
	p := api.Group("/")
	p.Use(middleware.RequireAuth(authSvc))

	// ════════════════════════════════════════════════════════════
	// ДАШБОРД — /dashboard
	// ════════════════════════════════════════════════════════════
	p.GET("/dashboard/stats", dashH.Stats)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: СОТРУДНИКИ — /employees
	// Страница: src/views/employees/Employees.js
	// Роли: admin, supervisor
	// ════════════════════════════════════════════════════════════

	// Вспомогательные — /roles, /users/assignable доступны всем авторизованным
	// ВАЖНО: статические пути /assignable и /avatar должны быть ДО параметрического /:id
	p.GET("/roles",                     usersH.RoleList)
	p.GET("/users/assignable",          usersH.Assignable)        // для модала назначения — все роли
	p.POST("/users/avatar",             uploadH.UploadAvatar)     // загрузка аватара — все роли
	p.GET("/users",                     middleware.RequireRole("admin", "supervisor"), usersH.List)
	p.GET("/users/:id",                 middleware.RequireRole("admin", "supervisor"), usersH.Get)
	p.POST("/users",                    middleware.RequireRole("admin"),               usersH.Create)
	p.PATCH("/users/:id",               middleware.RequireRole("admin"),               usersH.Update)
	p.PATCH("/users/:id/toggle-active", middleware.RequireRole("admin"),               usersH.ToggleActive)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: ТАБЕЛЬ — /employees/timesheet
	// Страница: src/views/employees/Timesheet.js
	// Роли: admin, supervisor
	// ════════════════════════════════════════════════════════════
	p.GET("/timesheets",            middleware.RequireRole("admin", "supervisor"), assigneeH.TimesheetList)
	p.GET("/timesheets/summary",    middleware.RequireRole("admin", "supervisor"), assigneeH.TimesheetSummary)
	p.POST("/timesheets",           middleware.RequireRole("admin", "supervisor"), assigneeH.TimesheetCreate)
	p.DELETE("/timesheets/:id",     middleware.RequireRole("admin", "supervisor"), assigneeH.TimesheetDelete)
	p.POST("/timesheets/auto-fill", middleware.RequireRole("admin", "supervisor"), assigneeH.TimesheetAutoFill)

	// Выплаты зарплат (вкладка "Выплаты" в Табеле)
	p.GET("/salary-payments",        middleware.RequireRole("admin", "supervisor"), assigneeH.SalaryPaymentList)
	p.POST("/salary-payments",       middleware.RequireRole("admin", "supervisor"), assigneeH.SalaryPaymentCreate)
	p.DELETE("/salary-payments/:id", middleware.RequireRole("admin", "supervisor"), assigneeH.SalaryPaymentDelete)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: ПРОЕКТЫ — /projects
	// Страница: src/views/projects/Projects.js + ProjectDetail.js
	// Роли: admin, supervisor, manager (список); все (этапы назначенных)
	// ════════════════════════════════════════════════════════════
	p.GET("/projects",                middleware.RequireRole("admin", "supervisor", "manager"), projH.List)
	p.POST("/projects",               middleware.RequireRole("admin", "supervisor", "manager"), projH.Create)
	p.PATCH("/projects/:project_id",  middleware.RequireRole("admin", "supervisor", "manager"), projH.Update)
	p.DELETE("/projects/:project_id", middleware.RequireRole("admin"),                          projH.Delete)

	p.GET("/catalog/operations", pipelineH.CatalogList)

	// Этапы проекта + мультиназначение (кнопка "Назначить" в ProjectDetail)
	p.GET("/projects/:project_id/stages",                     pipelineH.StagesList)
	p.GET("/projects/:project_id/stages/:stage_id",           pipelineH.StageGet)
	p.PATCH("/projects/:project_id/stages/:stage_id",         pipelineH.StageUpdate)
	p.POST("/projects/:project_id/stages/:stage_id/complete", pipelineH.StageComplete)
	p.GET("/projects/:project_id/stages/:stage_id/assignees", assigneeH.ProjectStageAssignees)
	p.PUT("/projects/:project_id/stages/:stage_id/assignees",
		middleware.RequireRole("admin", "supervisor", "manager"), assigneeH.ProjectStageSync)

	// Операции проекта
	p.GET("/projects/:project_id/operations",                  pipelineH.OperationsByProject)
	p.POST("/projects/:project_id/operations",                 pipelineH.OperationCreate)
	p.GET("/projects/:project_id/stages/:stage_id/operations", pipelineH.OperationsList)
	p.PATCH("/projects/:project_id/operations/:operation_id",  pipelineH.OperationUpdate)
	p.DELETE("/projects/:project_id/operations/:operation_id",
		middleware.RequireRole("admin", "supervisor"), pipelineH.OperationDelete)

	// Материалы проекта
	p.GET("/projects/:project_id/materials",                                          pipelineH.MaterialsByProject)
	p.GET("/projects/:project_id/operations/:operation_id/materials",                 pipelineH.MaterialsList)
	p.POST("/projects/:project_id/operations/:operation_id/materials",                pipelineH.MaterialCreate)
	p.DELETE("/projects/:project_id/operations/:operation_id/materials/:material_id",
		middleware.RequireRole("admin", "supervisor"), pipelineH.MaterialDelete)

	// Файлы проекта
	p.GET("/projects/:project_id/stages/:stage_id/files",   pipelineH.FilesList)
	p.POST("/projects/:project_id/stages/:stage_id/upload", uploadH.UploadStageFiles)
	p.POST("/projects/:project_id/stages/:stage_id/files",  pipelineH.FileCreate)
	p.DELETE("/projects/:project_id/stages/:stage_id/files/:file_id",
		middleware.RequireRole("admin", "supervisor", "designer"), pipelineH.FileDelete)

	p.GET("/projects/:project_id/history", pipelineH.History)
	p.DELETE("/files", middleware.RequireRole("admin", "supervisor", "designer"), uploadH.DeleteFile)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: ЗАДАЧИ — /tasks
	// Страница: src/views/tasks/Tasks.js
	// Роли: все (список своих); admin/supervisor (создание/удаление)
	// ════════════════════════════════════════════════════════════
	p.GET("/tasks",              tasksH.List)
	p.POST("/tasks",             middleware.RequireRole("admin", "supervisor"), tasksH.Create)
	p.PATCH("/tasks/:id",        tasksH.Update)
	p.PATCH("/tasks/:id/status", tasksH.UpdateStatus)
	p.DELETE("/tasks/:id",       middleware.RequireRole("admin", "supervisor"), tasksH.Delete)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: КЛИЕНТЫ — /clients
	// Страница: src/views/clients/Clients.js
	// Роли: admin, supervisor, manager (скрыто от master/warehouse)
	// ════════════════════════════════════════════════════════════
	p.GET("/clients",       orderH.ClientList)
	p.POST("/clients",      middleware.RequireRole("admin", "supervisor", "manager"), orderH.ClientCreate)
	p.PATCH("/clients/:id", middleware.RequireRole("admin", "supervisor", "manager"), orderH.ClientUpdate)

	// Баланс клиентов (вкладка "Долг" в Клиентах)
	p.GET("/clients/debt",                        clientBalanceH.DebtList)
	p.GET("/clients/:id/orders",                  clientBalanceH.ClientOrders)
	p.GET("/clients/:id/payments",                clientBalanceH.PaymentHistory)
	p.POST("/clients/:id/payments",               middleware.RequireRole("admin", "supervisor", "manager"), clientBalanceH.PaymentCreate)
	p.DELETE("/clients/:id/payments/:payment_id", middleware.RequireRole("admin", "supervisor", "manager"), clientBalanceH.PaymentDelete)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: ЗАКАЗЫ — /orders
	// Страница: src/views/orders/Orders.js + OrderDetail.js
	// Роли: все видят назначенные; admin/supervisor/manager — все
	// ════════════════════════════════════════════════════════════

	// Прайслист и каталог (используется в смете заказа)
	p.GET("/price-list",        orderH.PriceList)
	p.PATCH("/price-list/:id",  middleware.RequireRole("admin", "supervisor"), orderH.PriceUpdate)
	p.GET("/materials/catalog", orderH.MaterialsCatalog)

	// Каталог сметы (вкладка "Смета" в OrderDetail)
	p.GET("/estimate/catalog",        estimateH.CatalogList)
	p.GET("/estimate/catalog/flat",   estimateH.CatalogFlat)
	p.POST("/estimate/catalog",       middleware.RequireRole("admin", "supervisor"), estimateH.CatalogCreate)
	p.PATCH("/estimate/catalog/:id",  middleware.RequireRole("admin", "supervisor"), estimateH.CatalogUpdate)
	p.DELETE("/estimate/catalog/:id", middleware.RequireRole("admin", "supervisor"), estimateH.CatalogDelete)
	p.GET("/estimate/colors",         estimateH.ColorList)

	// Список и CRUD заказов
	p.GET("/orders",              orderH.OrderList)
	p.GET("/orders/stats",        orderH.OrderStats)
	p.GET("/orders/labels",       orderH.Labels)
	p.POST("/orders",             middleware.RequireRole("admin", "supervisor", "manager"), orderH.OrderCreate)
	p.GET("/orders/:order_id",    orderH.OrderGet)
	p.PATCH("/orders/:order_id",  middleware.RequireRole("admin", "supervisor", "manager"), orderH.OrderUpdate)
	p.DELETE("/orders/:order_id", middleware.RequireRole("admin", "supervisor"),            orderH.OrderCancel)

	// Этапы заказа + мультиназначение (кнопка "Назначить" в OrderDetail)
	p.GET("/orders/:order_id/stages",                     orderH.StagesList)
	p.PATCH("/orders/:order_id/stages/:stage_id",         orderH.StageUpdate)
	p.POST("/orders/:order_id/stages/:stage_id/complete", orderH.StageComplete)
	p.GET("/orders/:order_id/stages/:stage_id/assignees", assigneeH.OrderStageAssignees)
	p.PUT("/orders/:order_id/stages/:stage_id/assignees",
		middleware.RequireRole("admin", "supervisor", "manager"), assigneeH.OrderStageSync)

	// Файлы заказа (вкладка "Файлы" в OrderDetail)
	p.GET("/orders/:order_id/stages/:stage_id/files",   pipelineH.FilesList)
	p.POST("/orders/:order_id/stages/:stage_id/upload", uploadH.UploadStageFiles)
	p.POST("/orders/:order_id/stages/:stage_id/files",  pipelineH.FileCreate)
	p.DELETE("/orders/:order_id/stages/:stage_id/files/:file_id",
		middleware.RequireRole("admin", "supervisor", "designer"), pipelineH.FileDelete)

	// Оплаты заказа (вкладка "Оплата" в OrderDetail — только admin)
	p.GET("/orders/:order_id/payments",  orderH.PaymentsList)
	p.POST("/orders/:order_id/payments", middleware.RequireRole("admin", "supervisor", "manager"), orderH.PaymentCreate)

	// Комментарии заказа (вкладка "Комментарии" в OrderDetail)
	p.GET("/orders/:order_id/comments",  orderH.CommentsList)
	p.POST("/orders/:order_id/comments", orderH.CommentCreate)

	// История заказа (вкладка "История" в OrderDetail)
	p.GET("/orders/:order_id/history", orderH.History)

	// Материалы заказа (вкладка "Материалы" в OrderDetail)
	p.GET("/orders/:order_id/materials",  orderH.MaterialsList)
	p.POST("/orders/:order_id/materials", orderH.MaterialCreate)
	p.DELETE("/orders/:order_id/materials/:material_id",
		middleware.RequireRole("admin", "supervisor", "manager"), orderH.MaterialDelete)

	// Смета заказа (вкладка "Смета" в OrderDetail)
	p.GET("/orders/:order_id/estimate",  estimateH.EstimateGet)
	p.POST("/orders/:order_id/estimate", estimateH.EstimateSave)

	// Детальная смета (вкладка "Смета" — расширенная)
	p.GET("/orders/:order_id/detail-estimate",                  detailEstimateH.GetEstimate)
	p.POST("/orders/:order_id/detail-estimate",                 detailEstimateH.SaveSection)
	p.DELETE("/orders/:order_id/detail-estimate/:service_type", detailEstimateH.DeleteSection)

	// Расходы заказа (вкладка "Расходы" в OrderDetail — только workshop тип)
	p.GET("/orders/:order_id/expenses",    orderH.ExpensesList)
	p.POST("/orders/:order_id/expenses",   middleware.RequireRole("admin", "supervisor", "manager"), orderH.ExpenseCreate)
	p.DELETE("/orders/:order_id/expenses/:expense_id",
		middleware.RequireRole("admin", "supervisor", "manager"), orderH.ExpenseDelete)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: РАСХОДЫ — /expenses
	// Страница: src/views/expenses/Expenses.js
	// Роли: admin, supervisor, manager (скрыто от master/warehouse)
	// Категории: Зарплата (выбор сотрудника + лимит аванса),
	//            Проекты (выбор незавершённого проекта)
	// ════════════════════════════════════════════════════════════
	p.GET("/expenses",            expenseH.List)
	p.GET("/expenses/categories", expenseH.Categories)
	p.POST("/expenses",           middleware.RequireRole("admin", "supervisor", "manager"), expenseH.Create)
	p.PATCH("/expenses/:id",      middleware.RequireRole("admin", "supervisor", "manager"), expenseH.Update)
	p.DELETE("/expenses/:id",     middleware.RequireRole("admin", "supervisor"),            expenseH.Delete)

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: ОТЧЁТЫ — /reports
	// Страница: src/views/reports/Reports.js
	// Роли: admin, supervisor
	// (использует /orders/stats, /expenses, /timesheets/summary)
	// ════════════════════════════════════════════════════════════
	// Отдельных роутов нет — Reports.js вызывает существующие API

	// ════════════════════════════════════════════════════════════
	// МЕНЮ: СКЛАД — /warehouse
	// Страница: src/views/warehouse/WarehouseItems.js + Suppliers.js + Receipts.js
	// Роли: admin, supervisor, manager, warehouse (скрыто от master)
	// ════════════════════════════════════════════════════════════

	// Справочники склада
	p.GET("/warehouse/units",      warehouseH.UnitList)
	p.GET("/warehouse/categories", warehouseH.CategoryList)

	// Номенклатура (/warehouse/items)
	p.GET("/warehouse/items",        warehouseH.ItemList)
	p.GET("/warehouse/items/:id",    warehouseH.ItemGet)
	p.POST("/warehouse/items",       middleware.RequireRole("admin", "supervisor"), warehouseH.ItemCreate)
	p.PUT("/warehouse/items/:id",    middleware.RequireRole("admin", "supervisor"), warehouseH.ItemUpdate)
	p.DELETE("/warehouse/items/:id", middleware.RequireRole("admin", "supervisor"), warehouseH.ItemDelete)

	// Поставщики (/warehouse/suppliers)
	p.GET("/warehouse/suppliers",        warehouseH.SupplierList)
	p.GET("/warehouse/suppliers/:id",    warehouseH.SupplierGet)
	p.POST("/warehouse/suppliers",       middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierCreate)
	p.PATCH("/warehouse/suppliers/:id",  middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierUpdate)
	p.DELETE("/warehouse/suppliers/:id", middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierDelete)

	// Платежи поставщикам
	p.GET("/warehouse/suppliers/:id/payments",
		middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentHistory)
	p.POST("/warehouse/suppliers/:id/payments",
		middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentCreate)
	p.DELETE("/warehouse/suppliers/:id/payments/:payment_id",
		middleware.RequireRole("admin", "supervisor"), warehouseH.SupplierPaymentDelete)

	// Приходные накладные (/warehouse/receipts)
	
	p.GET("/warehouse/receipts",              warehouseH.ReceiptList)
	p.GET("/warehouse/receipts/next-number",  warehouseH.ReceiptNextNumber)
	p.GET("/warehouse/receipts/:id",          warehouseH.ReceiptGet)
	p.POST("/warehouse/receipts",       middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptCreate)
	p.PATCH("/warehouse/receipts/:id",  middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptUpdate)
	p.DELETE("/warehouse/receipts/:id", middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptDelete)
	p.POST("/warehouse/receipts/:id/items",
		middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptItemAdd)
	p.DELETE("/warehouse/receipts/:id/items/:item_id",
		middleware.RequireRole("admin", "supervisor"), warehouseH.ReceiptItemDelete)

	// Оплаты накладных
	p.GET("/warehouse/receipts/:id/payments",
		middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentList)
	p.POST("/warehouse/receipts/:id/payments",
		middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentCreate)
	p.DELETE("/warehouse/receipts/:id/payments/:payment_id",
		middleware.RequireRole("admin", "supervisor"), warehouseH.PaymentDelete)
	p.GET("/warehouse/outgoing-invoices",           outgoingInvoiceH.List)
	p.POST("/warehouse/outgoing-invoices",          middleware.RequireRole("admin","supervisor","manager","seller"), outgoingInvoiceH.Create)
	p.GET("/warehouse/outgoing-invoices/:id",       outgoingInvoiceH.Get)
	p.POST("/warehouse/outgoing-invoices/:id/confirm", middleware.RequireRole("admin","supervisor","manager","seller"), outgoingInvoiceH.Confirm)
	p.POST("/warehouse/outgoing-invoices/:id/cancel",  middleware.RequireRole("admin","supervisor"), outgoingInvoiceH.Cancel)
	
	setupStatic(r)

	log.Printf("🚀 Server running on :%s", cfg.Server.Port)
	log.Printf("📖 Swagger UI: http://localhost:%s/swagger/index.html", cfg.Server.Port)
	r.Run(":" + cfg.Server.Port)
}