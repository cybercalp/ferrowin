package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	inventoryadapters "ferrowin/internal/inventory/adapters"
	inventorydomain "ferrowin/internal/inventory/domain"
	billingadapters "ferrowin/internal/billing/adapters"
	billingdomain "ferrowin/internal/billing/domain"
	purchasesadapters "ferrowin/internal/purchases/adapters"
	purchasesdomain "ferrowin/internal/purchases/domain"
	salesadapters "ferrowin/internal/sales/adapters"
	salesdomain "ferrowin/internal/sales/domain"
	securityadapters "ferrowin/internal/security/adapters"
	securitydomain "ferrowin/internal/security/domain"
	"ferrowin/internal/shared/idempotency"
	syncadapters "ferrowin/internal/sync/adapters"

	catalogadapters "ferrowin/internal/catalog/adapters"
	catalogdomain "ferrowin/internal/catalog/domain"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func main() {
	dbType := os.Getenv("FERROWIN_DB_TYPE")
	if dbType == "" {
		dbType = "postgres"
	}

	dbURL := os.Getenv("FERROWIN_DATABASE_URL")
	if dbURL == "" {
		if dbType == "postgres" {
			dbURL = "postgres://postgres:postgres@localhost:5432/ferrowin?sslmode=disable"
		} else {
			dbURL = "file:ferrowin_dev.db?cache=shared&_pragma=foreign_keys(1)"
		}
	}

	port := os.Getenv("FERROWIN_PORT")
	if port == "" {
		port = ":8080"
	}

	isSQLite := dbType == "sqlite"

	log.Printf("[server] Starting Ferrowin API Server...")
	log.Printf("[server] DB Type: %s", dbType)
	log.Printf("[server] DB Connection URL: %s", dbURL)

	db, err := sql.Open(dbType, dbURL)
	if err != nil {
		log.Fatalf("[server] Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("[server] Database ping failed: %v", err)
	}
	log.Printf("[server] Database connection established successfully.")

	// Initialize idempotency tracker schema
	tracker := idempotency.NewTracker(db, isSQLite)
	if err := tracker.InitSchema(context.Background()); err != nil {
		log.Fatalf("[server] Failed to initialize idempotency schema: %v", err)
	}
	log.Printf("[server] Idempotency tracker schema verified.")

	// Initialize services
	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, isSQLite)
	invService := inventorydomain.NewInventoryService(ledgerRepo)
	purchaseRepo := purchasesadapters.NewSQLPurchaseRepository(db, isSQLite)
	purchaseService := purchasesdomain.NewPurchaseService(purchaseRepo, invService)

	billingTerminalRepo := billingadapters.NewSQLTerminalRepository(db, isSQLite)
	billingSeriesRepo := billingadapters.NewSQLInvoicingSeriesRepository(db, isSQLite)
	billingServ := billingdomain.NewBillingService(billingSeriesRepo)
	billingCtrl := billingadapters.NewBillingController(billingTerminalRepo, billingSeriesRepo)

	secUserRepo := securityadapters.NewSQLUserRepository(db, isSQLite)
	secGroupRepo := securityadapters.NewSQLGroupRepository(db, isSQLite)
	secRoleSetRepo := securityadapters.NewSQLRoleSetRepository(db, isSQLite)
	secRoleRepo := securityadapters.NewSQLRoleRepository(db, isSQLite)

	// JWT configuration
	jwtSecret := os.Getenv("JWT_SECRET")
	envMode := os.Getenv("FERROWIN_ENV")
	if jwtSecret == "" {
		if envMode == "development" {
			jwtSecret = "ferrowin-dev-secret-do-not-use-in-production"
			log.Println("[auth] WARNING: Using dev JWT_SECRET. Set JWT_SECRET env var in production and FERROWIN_ENV=production.")
		} else {
			log.Fatal("[auth] FATAL: JWT_SECRET env var is required. Set JWT_SECRET or use FERROWIN_ENV=development for dev mode.")
		}
	}
	expiryHours := 24
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			expiryHours = h
		}
	}
	jwtCfg := securitydomain.NewJWTConfig(jwtSecret, time.Duration(expiryHours)*time.Hour)

	authService := securitydomain.NewAuthService(secUserRepo, jwtCfg)

	salesRepo := salesadapters.NewSQLSalesRepository(db, isSQLite)
	salesService := salesdomain.NewSalesService(salesRepo, invService, authService, billingServ)

	// Initialize controllers
	authController := securityadapters.NewAuthController(authService)
	rbacCtrl := securityadapters.NewRBACController(secUserRepo, secGroupRepo, secRoleSetRepo, secRoleRepo)
	authMiddleware := securityadapters.NewAuthMiddleware(jwtCfg)
	syncController := syncadapters.NewSalesSyncController(db, isSQLite, invService, tracker, billingServ)
	catalogController := syncadapters.NewCatalogSyncController(db, isSQLite, tracker)

	catalogRepo := catalogadapters.NewSQLCatalogRepository(db, isSQLite)
	catalogSvc := catalogdomain.NewCatalogService(catalogRepo)
	catalogCtrl := catalogadapters.NewCatalogController(catalogSvc)

	purchaseController := purchasesadapters.NewPurchaseController(purchaseService)
	salesController := salesadapters.NewSalesController(salesService)

	// Set up HTTP routing
	mux := http.NewServeMux()
	// Unprotected routes
	mux.Handle("POST /api/v1/auth/login", authController)
	mux.Handle("/api/v1/health", syncController)
	// Protected routes (require JWT auth)
	mux.Handle("/api/v1/sync/sales", authMiddleware.Middleware(syncController))
	mux.Handle("/api/v1/sync/closures", authMiddleware.Middleware(syncController))
	mux.Handle("/api/v1/sync/events", authMiddleware.Middleware(syncController))
	mux.Handle("/api/v1/sync/voids", authMiddleware.Middleware(syncController))
	mux.Handle("/api/v1/inventory/stock/", authMiddleware.Middleware(syncController))
	mux.Handle("/api/v1/catalog/sync", authMiddleware.Middleware(catalogController))
	mux.Handle("/api/v1/catalog/clients/dossier", authMiddleware.Middleware(catalogController))
	mux.Handle("/api/v1/sync/payments", authMiddleware.Middleware(catalogController))

	// Catalog CRUD routes (protected)
	mux.Handle("/api/v1/catalog/tipos-iva", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/tipos-iva/", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/familias", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/familias/", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/productos", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/productos/", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/clientes", authMiddleware.Middleware(catalogCtrl))
	mux.Handle("/api/v1/catalog/clientes/", authMiddleware.Middleware(catalogCtrl))

	// Warehouse transfers wiring
	transferRepo := inventoryadapters.NewSQLTransferRepository(db, isSQLite)
	whValidator := &purchaseWarehouseAdapter{purchaseRepo}
	transferSvc := inventorydomain.NewTransferService(transferRepo, whValidator)
	transferCtrl := inventoryadapters.NewTransferController(transferSvc)

	// Purchases routing (protected)
	mux.Handle("/api/v1/purchases/companies", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/companies/", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/warehouses", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/warehouses/", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/suppliers", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/suppliers/", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/pedidos", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/pedidos/", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/receipts", authMiddleware.Middleware(purchaseController))
	mux.Handle("/api/v1/purchases/receipts/", authMiddleware.Middleware(purchaseController))

	// Transfers routing (protected)
	mux.Handle("/api/v1/inventory/transfers", authMiddleware.Middleware(transferCtrl))
	mux.Handle("/api/v1/inventory/transfers/", authMiddleware.Middleware(transferCtrl))

	// Sales routing (protected)
	mux.Handle("/api/v1/sales/presupuestos", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/presupuestos/convert", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/presupuestos/", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/pedidos", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/pedidos/convert", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/pedidos/", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/albaranes", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/albaranes/process", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/albaranes/convert", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/albaranes/", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/facturas", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/facturas/", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/facturas-rectificativas", authMiddleware.Middleware(salesController))
	mux.Handle("/api/v1/sales/facturas-rectificativas/", authMiddleware.Middleware(salesController))

	// Billing CRUD routes (protected)
	mux.Handle("/api/v1/billing/terminals", authMiddleware.Middleware(billingCtrl))
	mux.Handle("/api/v1/billing/terminals/", authMiddleware.Middleware(billingCtrl))
	mux.Handle("/api/v1/billing/series", authMiddleware.Middleware(billingCtrl))
	mux.Handle("/api/v1/billing/series/", authMiddleware.Middleware(billingCtrl))

	// RBAC CRUD routes (protected)
	mux.Handle("/api/v1/security/users", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/users/", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/groups", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/groups/", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/role-sets", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/role-sets/", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/roles", authMiddleware.Middleware(rbacCtrl))
	mux.Handle("/api/v1/security/roles/", authMiddleware.Middleware(rbacCtrl))

	// Wrap in a simple logger middleware
	loggingMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[http] Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		mux.ServeHTTP(w, r)
		log.Printf("[http] Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})

	log.Printf("[server] Listening on %s...", port)
	if err := http.ListenAndServe(port, loggingMux); err != nil {
		log.Fatalf("[server] Server crash: %v", err)
	}
}

// purchaseWarehouseAdapter adapts purchases.SQLPurchaseRepository.GetWarehouse
// to inventorydomain.WarehouseValidator, avoiding circular imports.
type purchaseWarehouseAdapter struct {
	repo *purchasesadapters.SQLPurchaseRepository
}

func (a *purchaseWarehouseAdapter) GetWarehouse(ctx context.Context, id uuid.UUID) (*inventorydomain.WarehouseView, error) {
	wh, err := a.repo.GetWarehouse(ctx, id)
	if err != nil {
		return nil, err
	}
	return &inventorydomain.WarehouseView{
		ID:        wh.ID,
		EmpresaID: wh.EmpresaID,
	}, nil
}
