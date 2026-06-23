use tauri::Emitter;
use serde::Serialize;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use crate::db;

#[derive(Clone, Serialize)]
struct SyncStatusPayload {
    online: bool,
    pending_sync_count: usize,
}

/// Closure payload sent to the Go backend.
#[derive(serde::Serialize)]
struct SyncClosurePayload {
    id: String,
    opened_at: String,
    closed_at: String,
    cash_reported: f64,
    card_reported: f64,
    sales_total: f64,
}

/// Sale payload sent to the Go backend.
#[derive(serde::Serialize)]
struct SyncSalePayload {
    id: String,
    invoice_number: String,
    sequence_number: i64,
    created_at: String,
    total: f64,
    items: Vec<SyncSaleItemPayload>,
    firma_registro: Option<String>,
    hash_anterior: Option<String>,
    datos_encadenamiento: Option<String>,
    // Phase 5 — extended financial fields
    subtotal: f64,
    tax_total: f64,
    discount_total: f64,
    status: String,
    void_reason: Option<String>,
    voided_at: Option<String>,
    payments: Vec<SyncPayment>,
}

/// Sale item payload embedded in a sale sync request.
#[derive(serde::Serialize)]
struct SyncSaleItemPayload {
    item_id: String,
    quantity: f64,
    unit_price: f64,
}

/// Payment payload embedded in a sale sync request.
#[derive(serde::Serialize, Debug)]
struct SyncPayment {
    metodo_pago: String,
    amount: f64,
}

/// Wrapper sent to `/api/v1/sync/sales`.
#[derive(serde::Serialize)]
struct SyncSalesRequest {
    sales: Vec<SyncSalePayload>,
}

#[derive(serde::Serialize)]
struct SyncEventPayload {
    id: String,
    fecha_hora: String,
    tipo_evento: String,
    detalles: String,
}

#[derive(serde::Serialize)]
struct SyncEventsRequest {
    events: Vec<SyncEventPayload>,
}

/// Spawns the background sync loop. Runs every 30 seconds, checks
/// connectivity, emits `sync-status-changed`, and syncs pending records.
pub fn start_sync_loop(
    app_handle: tauri::AppHandle,
    db_path: String,
    backend_url: String,
    online_flag: Arc<AtomicBool>,
) {
    let backend_url = Arc::new(backend_url);
    let db_path = Arc::new(db_path);

    tauri::async_runtime::spawn(async move {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .expect("failed to build HTTP client");

        loop {
            // Network check: lightweight HEAD request.
            let online = client
                .head(format!("{}/api/v1/health", backend_url))
                .send()
                .await
                .map(|r| r.status().is_success())
                .unwrap_or(false);

            // Update shared online flag for get_terminal_health.
            online_flag.store(online, Ordering::Relaxed);
            eprintln!("[sync] health check: online={}", online);

            // Count pending records for the status event.
            let pending_count = count_pending(&db_path);

            // Emit status event to the frontend.
            let _ = app_handle.emit(
                "sync-status-changed",
                SyncStatusPayload {
                    online,
                    pending_sync_count: pending_count,
                },
            );

            if online {
                sync_pending_sales(&db_path, &backend_url, &client).await;
                sync_pending_closures(&db_path, &backend_url, &client).await;
                sync_pending_events(&db_path, &backend_url, &client).await;
                sync_pending_voids(&db_path, &backend_url, &client).await;
            }

            tokio::time::sleep(Duration::from_secs(30)).await;
        }
    });
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn count_pending(db_path: &str) -> usize {
    let Ok(conn) = rusqlite::Connection::open(db_path) else {
        return 0;
    };
    let sales: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM offline_sales WHERE sync_status = 'PENDING'",
            [],
            |r| r.get(0),
        )
        .unwrap_or(0);
    let closures: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM offline_box_closures WHERE sync_status = 'PENDING'",
            [],
            |r| r.get(0),
        )
        .unwrap_or(0);
    let events: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM registro_sucesos WHERE estado_sincronizacion = 'PENDING'",
            [],
            |r| r.get(0),
        )
        .unwrap_or(0);
    (sales + closures + events) as usize
}

// ---------------------------------------------------------------------------
// Sales sync
// ---------------------------------------------------------------------------

async fn sync_pending_sales(db_path: &str, backend_url: &str, client: &reqwest::Client) {
    let conn = match rusqlite::Connection::open(db_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[sync] failed to open db for sales sync: {e}");
            return;
        }
    };

    let sales = match db::get_pending_sales(&conn) {
        Ok(s) => s,
        Err(e) => {
            eprintln!("[sync] failed to query pending sales: {e}");
            return;
        }
    };

    if sales.is_empty() {
        return;
    }

    // Sync each sale individually so partial failures don't block others.
    for sale in &sales {
        // Collect items for this sale.
        let items = match load_sale_items(&conn, &sale.id) {
            Ok(items) => items,
            Err(e) => {
                eprintln!("[sync] failed to load items for sale {}: {e}", sale.id);
                continue;
            }
        };

        // Collect payments for this sale.
        let payments = match load_sale_payments(&conn, &sale.id) {
            Ok(p) => p,
            Err(e) => {
                eprintln!("[sync] failed to load payments for sale {}: {e}", sale.id);
                continue;
            }
        };

        let payload = SyncSalesRequest {
            sales: vec![SyncSalePayload {
                id: sale.id.clone(),
                invoice_number: sale.invoice_number.clone(),
                sequence_number: sale.sequence_number,
                created_at: sale.created_at.clone(),
                total: sale.total,
                items: items
                    .into_iter()
                    .map(|i| SyncSaleItemPayload {
                        item_id: i.item_id,
                        quantity: i.quantity,
                        unit_price: i.unit_price,
                    })
                    .collect(),
                firma_registro: sale.firma_registro.clone(),
                hash_anterior: sale.hash_anterior.clone(),
                datos_encadenamiento: sale.datos_encadenamiento.clone(),
                // Phase 5 extended fields
                subtotal: sale.subtotal,
                tax_total: sale.tax_total,
                discount_total: sale.discount_total,
                status: sale.status.clone(),
                void_reason: sale.void_reason.clone(),
                voided_at: sale.voided_at.clone(),
                payments,
            }],
        };

        let res = client
            .post(format!("{backend_url}/api/v1/sync/sales"))
            .header("Idempotency-Key", &sale.idempotency_key)
            .json(&payload)
            .send()
            .await;

        match res {
            Ok(r) if r.status().is_success() => {
                if let Err(e) = db::delete_synced_sale(&conn, &sale.id) {
                    eprintln!("[sync] failed to delete synced sale {}: {e}", sale.id);
                }
            }
            Ok(r) if r.status() == reqwest::StatusCode::CONFLICT => {
                // HTTP 409 — conflict: do NOT delete the sale, record a CONFLICT event
                let event_id = uuid::Uuid::new_v4().to_string();
                let reason = format!("HTTP 409 conflict syncing sale {} ({})", sale.id, sale.invoice_number);
                if let Err(e) = db::insert_registro_suceso(&conn, &event_id, "CONFLICT", &reason) {
                    eprintln!("[sync] failed to record CONFLICT event for sale {}: {e}", sale.id);
                }
                eprintln!("[sync] conflict syncing sale {} ({}): kept PENDING", sale.id, sale.invoice_number);
            }
            Ok(r) => {
                eprintln!(
                    "[sync] backend rejected sale {} with status {}",
                    sale.id,
                    r.status()
                );
            }
            Err(e) => {
                eprintln!("[sync] network error syncing sale {}: {e}", sale.id);
            }
        }
    }
}

fn load_sale_items(
    conn: &rusqlite::Connection,
    sale_id: &str,
) -> rusqlite::Result<Vec<db::OfflineSaleItem>> {
    let mut stmt = conn.prepare(
        "SELECT id, offline_sale_id, item_id, quantity, unit_price, discount_percent
         FROM offline_sale_items
         WHERE offline_sale_id = ?1",
    )?;

    let rows = stmt.query_map(rusqlite::params![sale_id], |row| {
        Ok(db::OfflineSaleItem {
            id: row.get(0)?,
            offline_sale_id: row.get(1)?,
            item_id: row.get(2)?,
            quantity: row.get(3)?,
            unit_price: row.get(4)?,
            discount_percent: row.get(5)?,
        })
    })?;

    rows.collect()
}

fn load_sale_payments(
    conn: &rusqlite::Connection,
    sale_id: &str,
) -> rusqlite::Result<Vec<SyncPayment>> {
    let mut stmt = conn.prepare(
        "SELECT metodo_pago, amount FROM offline_sale_payments WHERE sale_id = ?1 ORDER BY created_at",
    )?;

    let rows = stmt.query_map(rusqlite::params![sale_id], |row| {
        Ok(SyncPayment {
            metodo_pago: row.get(0)?,
            amount: row.get(1)?,
        })
    })?;

    rows.collect()
}

// ---------------------------------------------------------------------------
// Closures sync
// ---------------------------------------------------------------------------

async fn sync_pending_closures(db_path: &str, backend_url: &str, client: &reqwest::Client) {
    let conn = match rusqlite::Connection::open(db_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[sync] failed to open db for closures sync: {e}");
            return;
        }
    };

    let closures = match db::get_pending_closures(&conn) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[sync] failed to query pending closures: {e}");
            return;
        }
    };

    for closure in &closures {
        let payload = SyncClosurePayload {
            id: closure.id.clone(),
            opened_at: closure.opened_at.clone(),
            closed_at: closure.closed_at.clone(),
            cash_reported: closure.cash_reported,
            card_reported: closure.card_reported,
            sales_total: closure.sales_total,
        };

        let res = client
            .post(format!("{backend_url}/api/v1/sync/closures"))
            .header("Idempotency-Key", &closure.idempotency_key)
            .json(&payload)
            .send()
            .await;

        match res {
            Ok(r) if r.status().is_success() => {
                if let Err(e) = db::delete_synced_closure(&conn, &closure.id) {
                    eprintln!(
                        "[sync] failed to delete synced closure {}: {e}",
                        closure.id
                    );
                }
            }
            Ok(r) => {
                eprintln!(
                    "[sync] backend rejected closure {} with status {}",
                    closure.id,
                    r.status()
                );
            }
            Err(e) => {
                eprintln!("[sync] network error syncing closure {}: {e}", closure.id);
            }
        }
    }
}

async fn sync_pending_events(db_path: &str, backend_url: &str, client: &reqwest::Client) {
    let conn = match rusqlite::Connection::open(db_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[sync] failed to open db for events sync: {e}");
            return;
        }
    };

    let events = match db::get_pending_events(&conn) {
        Ok(evs) => evs,
        Err(e) => {
            eprintln!("[sync] failed to query pending events: {e}");
            return;
        }
    };

    if events.is_empty() {
        return;
    }

    let payload = SyncEventsRequest {
        events: events
            .iter()
            .map(|e| SyncEventPayload {
                id: e.id.clone(),
                fecha_hora: e.fecha_hora.clone(),
                tipo_evento: e.tipo_evento.clone(),
                detalles: e.detalles.clone(),
            })
            .collect(),
    };

    let idempotency_key = events[0].id.clone();

    let res = client
        .post(format!("{backend_url}/api/v1/sync/events"))
        .header("Idempotency-Key", &idempotency_key)
        .json(&payload)
        .send()
        .await;

    match res {
        Ok(r) if r.status().is_success() => {
            for event in &events {
                if let Err(e) = db::delete_synced_event(&conn, &event.id) {
                    eprintln!("[sync] failed to delete synced event {}: {e}", event.id);
                }
            }
        }
        Ok(r) => {
            eprintln!(
                "[sync] backend rejected events batch with status {}",
                r.status()
            );
        }
        Err(e) => {
            eprintln!("[sync] network error syncing events: {e}");
        }
    }
}

// ---------------------------------------------------------------------------
// Void sync
// ---------------------------------------------------------------------------

/// Syncs ANULACION events that are pending — POSTs void details to the Go
/// backend and marks the event as synced (deleted) on success.
async fn sync_pending_voids(db_path: &str, backend_url: &str, client: &reqwest::Client) {
    let conn = match rusqlite::Connection::open(db_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[sync] failed to open db for void sync: {e}");
            return;
        }
    };

    let all_events = match db::get_pending_events(&conn) {
        Ok(evs) => evs,
        Err(e) => {
            eprintln!("[sync] failed to query pending events for void sync: {e}");
            return;
        }
    };

    let anulaciones: Vec<_> = all_events
        .into_iter()
        .filter(|e| e.tipo_evento == "ANULACION")
        .collect();

    if anulaciones.is_empty() {
        return;
    }

    #[derive(serde::Serialize)]
    struct SyncVoidPayload {
        sale_id: String,
        reason: Option<String>,
        firma_registro: Option<String>,
        hash_anterior: Option<String>,
    }

    for event in &anulaciones {
        // Parse sale_id from detalles format:
        // "Void of sale {sale_id} ({invoice_number}): {n} items, hash_anterior={hash}"
        let sale_id = event
            .detalles
            .split("sale ")
            .nth(1)
            .and_then(|s| s.split(" (").next())
            .unwrap_or("");

        if sale_id.is_empty() {
            eprintln!(
                "[sync] failed to parse sale_id from ANULACION event {}: {}",
                event.id, event.detalles
            );
            continue;
        }

        let sale = match db::get_sale_by_id(&conn, sale_id) {
            Ok(Some(s)) => s,
            Ok(None) => {
                eprintln!(
                    "[sync] sale {} not found for void sync (event {}), skipping",
                    sale_id, event.id
                );
                continue;
            }
            Err(e) => {
                eprintln!("[sync] failed to query sale {} for void sync: {e}", sale_id);
                continue;
            }
        };

        let payload = SyncVoidPayload {
            sale_id: sale.id.clone(),
            reason: sale.void_reason.clone(),
            firma_registro: sale.firma_registro.clone(),
            hash_anterior: sale.hash_anterior.clone(),
        };

        let res = client
            .post(format!("{backend_url}/api/v1/sync/voids"))
            .header("Idempotency-Key", &event.id)
            .json(&payload)
            .send()
            .await;

        match res {
            Ok(r) if r.status().is_success() => {
                if let Err(e) = db::delete_synced_event(&conn, &event.id) {
                    eprintln!(
                        "[sync] failed to delete synced ANULACION event {}: {e}",
                        event.id
                    );
                }
            }
            Ok(r) => {
                eprintln!(
                    "[sync] backend rejected void for sale {} with status {}",
                    sale_id,
                    r.status()
                );
            }
            Err(e) => {
                eprintln!("[sync] network error syncing void for sale {}: {e}", sale_id);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::{
        init_db, insert_offline_box_closure, insert_offline_sale, insert_offline_sale_item,
        get_pending_closures, get_pending_sales, OfflineBoxClosure, OfflineSale, OfflineSaleItem,
        insert_registro_suceso, get_pending_events,
    };
    use tokio::net::TcpListener;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    fn get_temp_db_path() -> (String, std::path::PathBuf) {
        let uuid = uuid::Uuid::new_v4().to_string();
        let mut path = std::env::temp_dir();
        path.push(format!("ferrowin_test_{}.db", uuid));
        let path_str = path.to_str().unwrap().to_string();
        (path_str, path)
    }
    
    async fn spawn_mock_server() -> (String, tokio::sync::mpsc::Receiver<String>, tokio::task::JoinHandle<()>) {
        let (tx, rx) = tokio::sync::mpsc::channel(10);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let port = listener.local_addr().unwrap().port();
        let address = format!("http://127.0.0.1:{}", port);
        
        let handle = tokio::spawn(async move {
            while let Ok((mut stream, _)) = listener.accept().await {
                let tx = tx.clone();
                tokio::spawn(async move {
                    let mut buf = vec![0; 4096];
                    if let Ok(n) = stream.read(&mut buf).await {
                        let req_str = String::from_utf8_lossy(&buf[..n]).to_string();
                        let _ = tx.send(req_str).await;
                    }
                    let response = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 25\r\n\r\n{\"status\":\"success\"}";
                    let _ = stream.write_all(response.as_bytes()).await;
                });
            }
        });
        
        (address, rx, handle)
    }

    #[tokio::test]
    async fn test_sync_closures_flow() {
        let (db_path_str, db_path) = get_temp_db_path();
        
        // Initialize schema (verifies that SQLite records are successfully stored/schema initializes)
        let conn = init_db(&db_path_str).unwrap();
        
        let closure = OfflineBoxClosure {
            id: uuid::Uuid::new_v4().to_string(),
            opened_at: "2026-06-05T08:00:00Z".into(),
            closed_at: "2026-06-05T18:00:00Z".into(),
            cash_reported: 120.50,
            card_reported: 80.00,
            sales_total: 200.50,
            sync_status: "PENDING".into(),
            idempotency_key: uuid::Uuid::new_v4().to_string(),
        };
        insert_offline_box_closure(&conn, &closure).unwrap();
        
        // Check stored
        let pending = get_pending_closures(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 1. Offline Test (verifies: when connection is offline, records remain in the DB)
        let client = reqwest::Client::new();
        sync_pending_closures(&db_path_str, "http://127.0.0.1:54321", &client).await;
        
        // Check still in DB
        let pending = get_pending_closures(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 2. Online Test (verifies: when connection is online, records are POSTed and deleted upon 2xx success)
        let (mock_url, mut _rx, server_handle) = spawn_mock_server().await;
        sync_pending_closures(&db_path_str, &mock_url, &client).await;
        
        // Check deleted from DB
        let pending = get_pending_closures(&conn).unwrap();
        assert_eq!(pending.len(), 0);
        
        // Cleanup
        server_handle.abort();
        let _ = std::fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_sales_flow() {
        let (db_path_str, db_path) = get_temp_db_path();
        
        // Initialize schema
        let conn = init_db(&db_path_str).unwrap();
        
        let sale_id = uuid::Uuid::new_v4().to_string();
        let sale = OfflineSale {
            id: sale_id.clone(),
            terminal_id: "term-123".into(),
            customer_id: None,
            total: 150.00,
            created_at: "2026-06-05T12:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: uuid::Uuid::new_v4().to_string(),
            invoice_number: "TPV-9999".into(),
            sequence_number: 99,
            firma_registro: Some("mockfirma".into()),
            hash_anterior: Some("mockprev".into()),
            datos_encadenamiento: Some("mockdatos".into()),
            subtotal: 150.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();
        
        let item = OfflineSaleItem {
            id: uuid::Uuid::new_v4().to_string(),
            offline_sale_id: sale_id.clone(),
            item_id: "product-123".into(),
            quantity: 3.0,
            unit_price: 50.00,
            discount_percent: 0.0,
        };
        insert_offline_sale_item(&conn, &item).unwrap();
        
        // Check stored
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 1. Offline Test
        let client = reqwest::Client::new();
        sync_pending_sales(&db_path_str, "http://127.0.0.1:54321", &client).await;
        
        // Check still in DB
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 2. Online Test
        let (mock_url, mut rx, server_handle) = spawn_mock_server().await;
        sync_pending_sales(&db_path_str, &mock_url, &client).await;
        
        // Verify payload contains chaining metadata
        let req_body = rx.recv().await.expect("Expected payload to be received");
        assert!(req_body.contains("mockfirma"));
        assert!(req_body.contains("mockprev"));
        assert!(req_body.contains("mockdatos"));
        assert!(req_body.contains("TPV-9999"));
        
        // Check deleted from DB
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 0);
        
        // Cleanup
        server_handle.abort();
        let _ = std::fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_events_flow() {
        let (db_path_str, db_path) = get_temp_db_path();
        
        // Initialize schema
        let conn = init_db(&db_path_str).unwrap();
        
        insert_registro_suceso(&conn, "evt-001", "ALTA_FACTURA", "Factura emitida").unwrap();
        
        // Check stored
        let pending = get_pending_events(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 1. Offline Test
        let client = reqwest::Client::new();
        sync_pending_events(&db_path_str, "http://127.0.0.1:54321", &client).await;
        
        // Check still in DB
        let pending = get_pending_events(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        
        // 2. Online Test
        let (mock_url, mut rx, server_handle) = spawn_mock_server().await;
        sync_pending_events(&db_path_str, &mock_url, &client).await;
        
        // Verify payload contains event metadata
        let req_body = rx.recv().await.expect("Expected payload to be received");
        assert!(req_body.contains("ALTA_FACTURA"));
        assert!(req_body.contains("Factura emitida"));
        
        // Check deleted from DB
        let pending = get_pending_events(&conn).unwrap();
        assert_eq!(pending.len(), 0);
        
        // Cleanup
        server_handle.abort();
        let _ = std::fs::remove_file(db_path);
    }

    // -----------------------------------------------------------------------
    // Phase 5 — Extended payload, 409 conflict, void sync
    // -----------------------------------------------------------------------

    /// Helper: spawn a mock TCP server that always returns 409 Conflict.
    async fn spawn_mock_409() -> (String, tokio::task::JoinHandle<()>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let port = listener.local_addr().unwrap().port();
        let address = format!("http://127.0.0.1:{}", port);

        let handle = tokio::spawn(async move {
            if let Ok((mut stream, _)) = listener.accept().await {
                let mut buf = vec![0; 8192];
                let _ = stream.read(&mut buf).await;
                let body = r#"{"error":"duplicate"}"#;
                let response = format!(
                    "HTTP/1.1 409 Conflict\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{}",
                    body.len(),
                    body
                );
                let _ = stream.write_all(response.as_bytes()).await;
            }
        });

        (address, handle)
    }

    #[tokio::test]
    async fn test_sync_sales_409_conflict() {
        // Verifies: HTTP 409 → sale stays PENDING + CONFLICT event recorded
        let (db_path_str, db_path) = get_temp_db_path();
        let conn = init_db(&db_path_str).unwrap();

        let sale_id = uuid::Uuid::new_v4().to_string();
        let sale = OfflineSale {
            id: sale_id.clone(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 100.0,
            created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: uuid::Uuid::new_v4().to_string(),
            invoice_number: "TPV-409".into(),
            sequence_number: 409,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 100.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();

        let item = OfflineSaleItem {
            id: uuid::Uuid::new_v4().to_string(),
            offline_sale_id: sale_id.clone(),
            item_id: "prod-1".into(),
            quantity: 1.0,
            unit_price: 100.0,
            discount_percent: 0.0,
        };
        insert_offline_sale_item(&conn, &item).unwrap();

        let client = reqwest::Client::new();

        // 1. Offline test — sale stays
        sync_pending_sales(&db_path_str, "http://127.0.0.1:54321", &client).await;
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 1, "sale should survive offline attempt");

        // 2. 409 mock test — sale stays PENDING, CONFLICT event recorded
        let (mock_url, server_handle) = spawn_mock_409().await;
        sync_pending_sales(&db_path_str, &mock_url, &client).await;

        // Sale still present with sync_status='PENDING'
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 1, "sale should NOT be deleted on 409");
        assert_eq!(pending[0].id, sale_id);

        // CONFLICT event exists
        let events = get_pending_events(&conn).unwrap();
        let conflicts: Vec<_> = events.iter().filter(|e| e.tipo_evento == "CONFLICT").collect();
        assert_eq!(conflicts.len(), 1, "a CONFLICT event should be recorded");
        assert!(conflicts[0].detalles.contains("409"));

        server_handle.abort();
        let _ = std::fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_voids_flow() {
        // Verifies: void sale → ANULACION event → sync_pending_voids POSTs →
        //          event deleted on success
        let (db_path_str, db_path) = get_temp_db_path();
        let conn = init_db(&db_path_str).unwrap();

        let sale_id = uuid::Uuid::new_v4().to_string();
        let sale = OfflineSale {
            id: sale_id.clone(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 100.0,
            created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: uuid::Uuid::new_v4().to_string(),
            invoice_number: "TPV-VOID".into(),
            sequence_number: 500,
            firma_registro: Some("firma_void_test".into()),
            hash_anterior: Some("prev_void_test".into()),
            datos_encadenamiento: Some("chain".into()),
            subtotal: 100.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();
        drop(conn);

        // Void the sale (creates ANULACION event)
        crate::void_sale_impl(&sale_id, "Test void reason", &db_path_str).unwrap();

        let conn2 = rusqlite::Connection::open(&db_path_str).unwrap();

        // Verify ANULACION event exists
        let events = get_pending_events(&conn2).unwrap();
        let anulaciones: Vec<_> = events.iter().filter(|e| e.tipo_evento == "ANULACION").collect();
        assert_eq!(anulaciones.len(), 1, "ANULACION event should exist");
        assert!(
            anulaciones[0].detalles.contains(&sale_id),
            "detalles should contain sale_id"
        );

        // Sync voids to mock server
        let (mock_url, mut rx, server_handle) = spawn_mock_server().await;
        let client = reqwest::Client::new();
        sync_pending_voids(&db_path_str, &mock_url, &client).await;

        // ANULACION event should be deleted (synced)
        let remaining = get_pending_events(&conn2).unwrap();
        let remaining_anulaciones: Vec<_> = remaining
            .iter()
            .filter(|e| e.tipo_evento == "ANULACION")
            .collect();
        assert_eq!(
            remaining_anulaciones.len(),
            0,
            "ANULACION event should be deleted after successful sync"
        );

        // Verify request payload
        let req_body = rx.recv().await.expect("expected void payload");
        assert!(req_body.contains(&sale_id));
        assert!(req_body.contains("Test void reason"));
        assert!(req_body.contains("firma_void_test"));
        assert!(req_body.contains("prev_void_test"));

        server_handle.abort();
        drop(conn2);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_sale_payload_extended_fields_serialization() {
        // Verifies that SyncSalePayload extended fields serialize correctly
        use super::SyncPayment;

        let payload = SyncSalePayload {
            id: "sale-ser".into(),
            invoice_number: "TPV-SER".into(),
            sequence_number: 1,
            created_at: "2026-06-05T10:00:00Z".into(),
            total: 121.0,
            items: vec![SyncSaleItemPayload {
                item_id: "prod-1".into(),
                quantity: 2.0,
                unit_price: 50.0,
            }],
            firma_registro: Some("sig".into()),
            hash_anterior: Some("prev".into()),
            datos_encadenamiento: Some("chain".into()),
            subtotal: 100.0,
            tax_total: 21.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
            payments: vec![SyncPayment {
                metodo_pago: "EFECTIVO".into(),
                amount: 121.0,
            }],
        };

        let json = serde_json::to_value(&payload).unwrap();
        assert_eq!(json["subtotal"], 100.0);
        assert_eq!(json["tax_total"], 21.0);
        assert_eq!(json["discount_total"], 0.0);
        assert_eq!(json["status"], "COMPLETED");
        assert!(json["void_reason"].is_null());
        assert!(json["voided_at"].is_null());
        assert_eq!(json["payments"][0]["metodo_pago"], "EFECTIVO");
        assert_eq!(json["payments"][0]["amount"], 121.0);
    }

}

