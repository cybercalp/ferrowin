package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
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

	billingRepo := billingadapters.NewSQLInvoicingSeriesRepository(db, isSQLite)
	billingServ := billingdomain.NewBillingService(billingRepo)

	secUserRepo := securityadapters.NewSQLUserRepository(db, isSQLite)
	authService := securitydomain.NewAuthService(secUserRepo)

	salesRepo := salesadapters.NewSQLSalesRepository(db, isSQLite)
	salesService := salesdomain.NewSalesService(salesRepo, invService, authService, billingServ)

	// Initialize controllers
	syncController := syncadapters.NewSalesSyncController(db, isSQLite, invService, tracker)
	catalogController := syncadapters.NewCatalogSyncController(db, isSQLite, tracker)
	purchaseController := purchasesadapters.NewPurchaseController(purchaseService)
	salesController := salesadapters.NewSalesController(salesService)

	// Set up HTTP routing
	mux := http.NewServeMux()
	mux.Handle("/api/v1/health", syncController)
	mux.Handle("/api/v1/sync/sales", syncController)
	mux.Handle("/api/v1/sync/closures", syncController)
	mux.Handle("/api/v1/sync/events", syncController)
	mux.Handle("/api/v1/sync/voids", syncController)
	mux.Handle("/api/v1/inventory/stock/", syncController)
	mux.Handle("/api/v1/catalog/sync", catalogController)
	mux.Handle("/api/v1/catalog/clients/dossier", catalogController)
	mux.Handle("/api/v1/sync/payments", catalogController)

	// Purchases routing
	mux.Handle("/api/v1/purchases/companies", purchaseController)
	mux.Handle("/api/v1/purchases/warehouses", purchaseController)
	mux.Handle("/api/v1/purchases/suppliers", purchaseController)
	mux.Handle("/api/v1/purchases/orders", purchaseController)
	mux.Handle("/api/v1/purchases/orders/approve/", purchaseController)
	mux.Handle("/api/v1/purchases/receipts", purchaseController)
	mux.Handle("/api/v1/purchases/receipts/process/", purchaseController)

	// Sales routing
	mux.Handle("/api/v1/sales/quotes", salesController)
	mux.Handle("/api/v1/sales/quotes/convert", salesController)
	mux.Handle("/api/v1/sales/orders", salesController)
	mux.Handle("/api/v1/sales/orders/convert", salesController)
	mux.Handle("/api/v1/sales/delivery-notes/process", salesController)
	mux.Handle("/api/v1/sales/delivery-notes/convert", salesController)

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
