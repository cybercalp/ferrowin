use tauri::Manager;
use tauri::State;
use std::time::Duration;
use rusqlite::params;

pub mod auth;
pub mod db;
pub mod sync;
pub mod signature;
pub mod catalog_sync;


pub struct DbState {
    pub db_path: String,
    pub backend_url: String,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ReceiptLineItem {
    pub codigo: String,
    pub nombre: String,
    pub quantity: f64,
    pub unit_price: f64,
    pub discount_percent: f64,
    pub line_total: f64,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ReceiptPayment {
    pub metodo_pago: String,
    pub amount: f64,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ReceiptData {
    pub terminal_id: String,
    pub invoice_number: String,
    pub created_at: String,
    pub items: Vec<ReceiptLineItem>,
    pub payments: Vec<ReceiptPayment>,
    pub subtotal: f64,
    pub tax_total: f64,
    pub discount_total: f64,
    pub total: f64,
    pub firma_registro: Option<String>,
}

use crate::signature::{Firmador, FirmaSimulada};

pub fn save_offline_sale_impl(
    mut sale: db::OfflineSale,
    items: Vec<db::OfflineSaleItem>,
    db_path: &str,
) -> Result<(), String> {
    let mut conn = rusqlite::Connection::open(db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let tx = conn.transaction()
        .map_err(|e| format!("Failed to start transaction: {}", e))?;

    // Query last signature
    let last_signature = db::get_ultimo_registro_encadenado(&tx)
        .map_err(|e| format!("Failed to query last signature: {}", e))?;

    // Extract prefix
    let prefijo = sale.invoice_number.split('-').next().unwrap_or("TPV");

    // Generate signature
    let firmador = FirmaSimulada;
    let hash_anterior_opt = last_signature.as_deref();
    let current_signature = firmador
        .firmar_registro(
            prefijo,
            sale.sequence_number,
            sale.total,
            &sale.created_at,
            hash_anterior_opt,
        )
        .map_err(|e| format!("Failed to generate signature: {}", e))?;

    let now_iso = chrono::Utc::now().to_rfc3339();

    // Assign verifactu fields
    sale.firma_registro = Some(current_signature.clone());
    sale.hash_anterior = last_signature;
    sale.datos_encadenamiento = Some(format!(
        "Veri*factu-chained;alg=sha256;ts={}",
        now_iso
    ));

    // Insert sale and items
    db::insert_offline_sale(&tx, &sale)
        .map_err(|e| format!("Failed to insert sale: {}", e))?;

    for item in items {
        db::insert_offline_sale_item(&tx, &item)
            .map_err(|e| format!("Failed to insert sale item: {}", e))?;
        db::decrement_stock_cache(&tx, &item.item_id, item.quantity, &now_iso)
            .map_err(|e| format!("Failed to decrement stock cache: {}", e))?;
    }

    // Update last signature
    db::upsert_ultimo_registro_encadenado(&tx, &sale.id, &current_signature)
        .map_err(|e| format!("Failed to update last signature: {}", e))?;

    // Record the event in registro_sucesos
    let event_id = uuid::Uuid::new_v4().to_string();
    db::insert_registro_suceso(
        &tx,
        &event_id,
        "ALTA_FACTURA",
        &format!("Factura emitida offline: {}", sale.invoice_number),
    )
    .map_err(|e| format!("Failed to record audit event: {}", e))?;

    tx.commit().map_err(|e| format!("Failed to commit transaction: {}", e))?;
    Ok(())
}

/// Voids a completed sale: marks status='VOIDED', restores stock,
/// records ANULACION event with chain linkage. Does NOT update
/// ultimo_registro_encadenado per chain integrity rules.
pub fn void_sale_impl(sale_id: &str, reason: &str, db_path: &str) -> Result<(), String> {
    let mut conn = rusqlite::Connection::open(db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let tx = conn.transaction()
        .map_err(|e| format!("Failed to start transaction: {}", e))?;

    // 1. Read the sale and verify it exists and is COMPLETED
    let sale = db::get_sale_by_id(&tx, sale_id)
        .map_err(|e| format!("Failed to query sale: {}", e))?
        .ok_or_else(|| format!("Sale not found: {}", sale_id))?;

    if sale.status != "COMPLETED" {
        return Err(format!(
            "Sale {} is not COMPLETED (current status: {})",
            sale_id, sale.status
        ));
    }

    // 2. Read the voided sale's OWN hash_anterior (preceding valid sale's signature)
    let hash_anterior_ref = sale.hash_anterior.clone();

    let voided_at = chrono::Utc::now().to_rfc3339();

    // 3. Update status to VOIDED
    db::update_sale_status_to_voided(&tx, sale_id, reason, &voided_at)
        .map_err(|e| format!("Failed to void sale: {}", e))?;

    // 4. Restore stock for each item
    let items = db::get_sale_items(&tx, sale_id)
        .map_err(|e| format!("Failed to get sale items: {}", e))?;

    for item in &items {
        db::increment_stock_cache(&tx, &item.item_id, item.quantity)
            .map_err(|e| format!("Failed to restore stock for {}: {}", item.item_id, e))?;
    }

    // 5. Record ANULACION in registro_sucesos with chain metadata
    let event_id = uuid::Uuid::new_v4().to_string();
    let chain_ref = hash_anterior_ref.as_deref().unwrap_or("(none)");
    db::insert_registro_suceso(
        &tx,
        &event_id,
        "ANULACION",
        &format!(
            "Void of sale {} ({}): {} items, hash_anterior={}",
            sale_id, sale.invoice_number, items.len(), chain_ref
        ),
    )
    .map_err(|e| format!("Failed to record ANULACION event: {}", e))?;

    // 6. Do NOT update ultimo_registro_encadenado
    tx.commit().map_err(|e| format!("Failed to commit transaction: {}", e))?;
    Ok(())
}

#[tauri::command]
fn save_offline_sale(
    sale: db::OfflineSale,
    items: Vec<db::OfflineSaleItem>,
    state: State<'_, DbState>,
) -> Result<(), String> {
    save_offline_sale_impl(sale, items, &state.db_path)
}

#[tauri::command]
fn save_offline_closure(
    closure: db::OfflineBoxClosure,
    state: State<'_, DbState>,
) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::insert_offline_box_closure(&conn, &closure)
        .map_err(|e| format!("Failed to insert closure: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn get_stock(
    item_id: String,
    auth: State<'_, crate::auth::AuthState>,
    state: State<'_, DbState>,
) -> Result<Option<f64>, String> {
    let health_client = reqwest::Client::builder()
        .timeout(Duration::from_secs(3))
        .build()
        .map_err(|e| format!("Failed to build HTTP client: {}", e))?;

    let online = health_client
        .head(format!("{}/api/v1/health", state.backend_url))
        .send()
        .await
        .map(|r| r.status().is_success())
        .unwrap_or(false);

    if online {
        let authorized_client = crate::auth::get_authorized_client(&auth)
            .map_err(|e| format!("Failed to build authorized client: {}", e))?;
        let url = format!("{}/api/v1/inventory/stock/{}", state.backend_url, item_id);
        let res = authorized_client.get(&url).send().await;
        match res {
            Ok(r) if r.status().is_success() => {
                #[derive(serde::Deserialize)]
                struct StockResponse {
                    item_id: String,
                    stock: f64,
                }
                if let Ok(data) = r.json::<StockResponse>().await {
                    let conn = rusqlite::Connection::open(&state.db_path)
                        .map_err(|e| format!("Failed to open DB: {}", e))?;
                    
                    let now_iso = chrono::Utc::now().to_rfc3339();
                    
                    db::upsert_stock_cache(&conn, &data.item_id, data.stock, &now_iso)
                        .map_err(|e| format!("Failed to update stock cache: {}", e))?;
                    
                    return Ok(Some(data.stock));
                }
            }
            _ => {
                eprintln!("[get_stock] Central query failed, falling back to local cache.");
            }
        }
    }

    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    
    let cached = db::get_cached_stock(&conn, &item_id)
        .map_err(|e| format!("Failed to query cached stock: {}", e))?;
    
    Ok(cached)
}

#[tauri::command]
async fn sync_catalog(auth: State<'_, crate::auth::AuthState>, state: State<'_, DbState>) -> Result<(), String> {
    let token = crate::auth::get_token(&auth);
    catalog_sync::sync_catalog_delta(&state.db_path, &state.backend_url, token).await
}

#[tauri::command]
fn search_products(
    query: String,
    state: State<'_, DbState>,
) -> Result<Vec<db::POSProduct>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::search_products(&conn, &query)
        .map_err(|e| format!("Failed to search products: {}", e))
}

#[tauri::command]
fn get_product_by_code(
    code: String,
    state: State<'_, DbState>,
) -> Result<Option<db::POSProduct>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_product_by_code(&conn, &code)
        .map_err(|e| format!("Failed to get product: {}", e))
}

#[tauri::command]
fn get_next_sequence(
    prefix: String,
    state: State<'_, DbState>,
) -> Result<i64, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_next_sequence(&conn, &prefix)
        .map_err(|e| format!("Failed to get next sequence: {}", e))
}

#[tauri::command]
fn get_today_sales(
    state: State<'_, DbState>,
) -> Result<Vec<db::OfflineSale>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_today_sales(&conn)
        .map_err(|e| format!("Failed to get today's sales: {}", e))
}

#[tauri::command]
fn reset_barcode_buffer() -> Result<(), String> {
    // Frontend-managed barcode buffer; this command signals the frontend
    // to clear its accumulated input. No backend state needed.
    Ok(())
}

// ---------------------------------------------------------------------------
// Phase 2: Void + backend refinements
// ---------------------------------------------------------------------------

#[tauri::command]
fn void_sale(sale_id: String, reason: String, state: State<'_, DbState>) -> Result<(), String> {
    void_sale_impl(&sale_id, &reason, &state.db_path)
}

#[tauri::command]
fn registrar_apertura(amount: f64, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let id = uuid::Uuid::new_v4().to_string();
    let opened_at = chrono::Utc::now().to_rfc3339();
    db::insert_caja_apertura(&conn, &id, amount, &opened_at)
        .map_err(|e| format!("Failed to record cash register opening: {}", e))
}

#[tauri::command]
fn registrar_ingreso_caja(concepto: String, amount: f64, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let id = uuid::Uuid::new_v4().to_string();
    let created_at = chrono::Utc::now().to_rfc3339();
    db::insert_caja_movimiento(&conn, &id, "INGRESO", &concepto, amount, &created_at)
        .map_err(|e| format!("Failed to record cash income: {}", e))
}

#[tauri::command]
fn registrar_retiro_caja(concepto: String, amount: f64, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let id = uuid::Uuid::new_v4().to_string();
    let created_at = chrono::Utc::now().to_rfc3339();
    db::insert_caja_movimiento(&conn, &id, "RETIRO", &concepto, amount, &created_at)
        .map_err(|e| format!("Failed to record cash withdrawal: {}", e))
}

#[tauri::command]
fn registrar_cobro_pago(
    sale_id: String,
    metodo_pago: String,
    amount: f64,
    state: State<'_, DbState>,
) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let id = uuid::Uuid::new_v4().to_string();
    let created_at = chrono::Utc::now().to_rfc3339();
    db::insert_offline_sale_payment(&conn, &id, &sale_id, &metodo_pago, amount, &created_at)
        .map_err(|e| format!("Failed to record payment: {}", e))
}

// ---------------------------------------------------------------------------
// Phase 3 (backport): Terminal health
// ---------------------------------------------------------------------------

#[tauri::command]
fn get_terminal_health(state: State<'_, DbState>) -> Result<db::TerminalHealth, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;

    let db_size_bytes = db::get_db_size(&conn).unwrap_or(0);
    let pending_sales_count = db::get_pending_sales(&conn)
        .map(|s| s.len() as i64)
        .unwrap_or(0);
    let pending_closures_count = db::get_pending_closures(&conn)
        .map(|c| c.len() as i64)
        .unwrap_or(0);

    Ok(db::TerminalHealth {
        terminal_id: "default".to_string(),
        db_size_bytes,
        pending_sales_count,
        pending_closures_count,
        online_status: false,
        app_version: env!("CARGO_PKG_VERSION").to_string(),
    })
}

// ---------------------------------------------------------------------------
// Phase 5: Client dossier & entity commands
// ---------------------------------------------------------------------------

#[tauri::command]
fn get_clientes(state: State<'_, DbState>) -> Result<Vec<db::Cliente>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_all_clientes(&conn)
        .map_err(|e| format!("Failed to get clientes: {}", e))
}

#[tauri::command]
fn get_direcciones(entidad_id: String, state: State<'_, DbState>) -> Result<Vec<db::Direccion>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_direcciones_by_entidad(&conn, &entidad_id)
        .map_err(|e| format!("Failed to get direcciones: {}", e))
}

#[tauri::command]
fn get_contactos(entidad_id: String, state: State<'_, DbState>) -> Result<Vec<db::Contacto>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_contactos_by_entidad(&conn, &entidad_id)
        .map_err(|e| format!("Failed to get contactos: {}", e))
}

#[tauri::command]
fn get_notas(entidad_id: String, state: State<'_, DbState>) -> Result<Vec<db::Nota>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_notas_by_entidad(&conn, &entidad_id)
        .map_err(|e| format!("Failed to get notas: {}", e))
}

#[tauri::command]
fn save_direccion(direccion: db::Direccion, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::upsert_direccion(&conn, &direccion)
        .map_err(|e| format!("Failed to save direccion: {}", e))
}

#[tauri::command]
fn save_contacto(contacto: db::Contacto, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::upsert_contacto(&conn, &contacto)
        .map_err(|e| format!("Failed to save contacto: {}", e))
}

#[tauri::command]
fn save_nota(nota: db::Nota, state: State<'_, DbState>) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::upsert_nota(&conn, &nota)
        .map_err(|e| format!("Failed to save nota: {}", e))
}

#[tauri::command]
fn get_cliente_dossier(client_id: String, state: State<'_, DbState>) -> Result<db::ClientDossier, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_cliente_dossier(&conn, &client_id)
        .map_err(|e| format!("Failed to get cliente dossier: {}", e))
}

#[tauri::command]
fn registrar_cobro(
    id: String,
    cliente_id: String,
    factura_id: Option<String>,
    importe: f64,
    metodo_pago: String,
    tipo_cobro: String,
    state: State<'_, DbState>,
) -> Result<(), String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    let created_at = chrono::Utc::now().to_rfc3339();
    db::insert_cobro(&conn, &id, &cliente_id, factura_id.as_deref(), importe, &metodo_pago, &tipo_cobro, &created_at)
        .map_err(|e| format!("Failed to registrar cobro: {}", e))
}

#[tauri::command]
fn get_families(state: State<'_, DbState>) -> Result<Vec<String>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_distinct_families(&conn)
        .map_err(|e| format!("Failed to get families: {}", e))
}

#[tauri::command]
fn get_products_by_family(familia: Option<String>, state: State<'_, DbState>) -> Result<Vec<db::POSProduct>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::get_products_by_family(&conn, familia.as_deref())
        .map_err(|e| format!("Failed to get products by family: {}", e))
}

#[tauri::command]
fn search_clients(query: String, state: State<'_, DbState>) -> Result<Vec<db::CustomerInfo>, String> {
    let conn = rusqlite::Connection::open(&state.db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;
    db::search_clientes(&conn, &query)
        .map_err(|e| format!("Failed to search clients: {}", e))
}

// ---------------------------------------------------------------------------
// Phase 4 (backport): PDF receipt generation
// ---------------------------------------------------------------------------

#[tauri::command]
fn generate_receipt_pdf(sale_id: String, state: State<'_, DbState>) -> Result<String, String> {
    generate_receipt_pdf_impl(&sale_id, &state.db_path)
}

fn generate_receipt_pdf_impl(sale_id: &str, db_path: &str) -> Result<String, String> {
    let conn = rusqlite::Connection::open(db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;

    let sale = db::get_sale_by_id(&conn, sale_id)
        .map_err(|e| format!("Failed to query sale: {}", e))?
        .ok_or_else(|| format!("Sale not found: {}", sale_id))?;

    let items = db::get_sale_items(&conn, sale_id)
        .map_err(|e| format!("Failed to query items: {}", e))?;

    // Query payments for this sale
    let mut stmt = conn
        .prepare("SELECT metodo_pago, amount FROM offline_sale_payments WHERE sale_id = ?1")
        .map_err(|e| format!("Failed to prepare payment query: {}", e))?;
    let payments: Vec<ReceiptPayment> = stmt
        .query_map(params![sale_id], |row| {
            Ok(ReceiptPayment {
                metodo_pago: row.get(0)?,
                amount: row.get(1)?,
            })
        })
        .map_err(|e| format!("Failed to query payments: {}", e))?
        .filter_map(|r| r.ok())
        .collect();

    let receipt = ReceiptData {
        terminal_id: sale.terminal_id.clone(),
        invoice_number: sale.invoice_number.clone(),
        created_at: sale.created_at.clone(),
        items: items
            .into_iter()
            .map(|i| ReceiptLineItem {
                codigo: i.item_id.clone(),
                nombre: i.item_id,
                quantity: i.quantity,
                unit_price: i.unit_price,
                discount_percent: i.discount_percent,
                line_total: i.quantity * i.unit_price * (1.0 - i.discount_percent / 100.0),
            })
            .collect(),
        payments,
        subtotal: sale.subtotal,
        tax_total: sale.tax_total,
        discount_total: sale.discount_total,
        total: sale.total,
        firma_registro: sale.firma_registro.clone(),
    };

    let pdf_bytes = render_receipt_pdf(&receipt)?;
    use base64::Engine as _;
    let b64 = base64::engine::general_purpose::STANDARD.encode(&pdf_bytes);
    Ok(b64)
}

fn render_receipt_pdf(data: &ReceiptData) -> Result<Vec<u8>, String> {
    use printpdf::*;

    let line_h: f32 = 4.0;
    let line_count = 6.0f32 + data.items.len() as f32 * 2.0 + 6.0f32 + data.payments.len() as f32 + 6.0f32;
    let page_h = Mm(10.0 + line_count * line_h);
    let page_w = Mm(80.0); // 80mm thermal receipt width

    let (doc, page_idx, layer_idx) = PdfDocument::new(
        "Receipt",
        page_w,
        page_h,
        "Layer 1",
    );

    let font = doc.add_builtin_font(BuiltinFont::Helvetica)
        .map_err(|e| format!("Failed to load font: {}", e))?;
    let font_bold = doc.add_builtin_font(BuiltinFont::HelveticaBold)
        .map_err(|e| format!("Failed to load bold font: {}", e))?;

    let layer = doc.get_page(page_idx).get_layer(layer_idx);

    let mut y: f32 = page_h.0 - 10.0;
    let left: f32 = 5.0;
    let center: f32 = 28.0;

    // Helper: draw a line of text
    macro_rules! draw_text {
        ($txt:expr, $font:expr, $size:expr, $x:expr) => {
            layer.use_text($txt, $size, Mm($x), Mm(y), &$font);
            y -= $size * 0.45;
        };
    }

    // Header
    draw_text!("TIENDA FERROWIN", font_bold, 14.0, center);
    y -= 2.0;
    draw_text!(&format!("Terminal: {}", data.terminal_id), font, 8.0, center);
    draw_text!(&format!("Factura: {}", data.invoice_number), font, 8.0, center);
    draw_text!(&format!("Fecha: {}", &data.created_at[..19].replace("T", " ")), font, 8.0, center);
    y -= 3.0;

    // Separator
    draw_text!("=================================", font, 7.0, left);
    draw_text!("CODIGO   ARTICULO           QTY   TOTAL", font_bold, 7.0, left);
    draw_text!("---------------------------------", font, 7.0, left);

    // Items
    for item in &data.items {
        let codigo = if item.codigo.len() > 7 { &item.codigo[..7] } else { &item.codigo };
        let nombre = if item.nombre.len() > 16 {
            format!("{}..", &item.nombre[..14])
        } else {
            format!("{:<16}", item.nombre)
        };
        draw_text!(
            &format!("{} {} x{}  ${:.2}", codigo, nombre, item.quantity, item.line_total),
            font, 7.0, left
        );
        if item.discount_percent > 0.0 {
            draw_text!(&format!("    ({}% descuento)", item.discount_percent), font, 6.0, left);
        }
    }

    // Totals
    y -= 1.0;
    draw_text!("=================================", font, 7.0, left);
    draw_text!(&format!("SUB TOTAL:          ${:.2}", data.subtotal), font, 8.0, left);
    draw_text!(&format!("DSCTO:              ${:.2}", data.discount_total), font, 8.0, left);
    draw_text!(&format!("IVA:                ${:.2}", data.tax_total), font, 8.0, left);
    y -= 1.0;
    draw_text!(&format!("TOTAL:              ${:.2}", data.total), font_bold, 10.0, left);
    y -= 2.0;

    // Payments
    if !data.payments.is_empty() {
        draw_text!("--- PAGOS ---", font_bold, 8.0, center);
        for p in &data.payments {
            draw_text!(&format!("{}: ${:.2}", p.metodo_pago, p.amount), font, 8.0, left);
        }
    }

    // Footer
    draw_text!("---------------------------------", font, 7.0, left);
    if let Some(ref sig) = data.firma_registro {
        let short_sig = if sig.len() > 16 { &sig[..16] } else { sig };
        draw_text!(&format!("Chain: {}...", short_sig), font, 6.0, left);
    }
    draw_text!("Gracias por su compra!", font, 7.0, center);

    // Serialize to bytes
    let mut buf = Vec::new();
    let mut writer = std::io::BufWriter::new(&mut buf);
    doc.save(&mut writer)
        .map_err(|e| format!("Failed to save PDF: {}", e))?;
    drop(writer);

    Ok(buf)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![
            save_offline_sale,
            save_offline_closure,
            get_stock,
            sync_catalog,
            search_products,
            get_product_by_code,
            get_next_sequence,
            get_today_sales,
            reset_barcode_buffer,
            void_sale,
            registrar_apertura,
            registrar_ingreso_caja,
            registrar_retiro_caja,
            registrar_cobro_pago,
            get_terminal_health,
            generate_receipt_pdf,
            get_clientes,
            get_direcciones,
            get_contactos,
            get_notas,
            save_direccion,
            save_contacto,
            save_nota,
            get_cliente_dossier,
            registrar_cobro,
            get_families,
            get_products_by_family,
            search_clients,
            auth::login,
            auth::set_auth_state,
            auth::clear_auth,
            auth::get_auth_token,
        ])

        .setup(|app| {
            let app_handle = app.handle().clone();

            // Resolve the database path in the app's data directory.
            let data_dir = app
                .path()
                .app_data_dir()
                .expect("failed to resolve app data dir");
            std::fs::create_dir_all(&data_dir).ok();
            let db_path = data_dir.join("tpv_offline.db");
            let db_path_str = db_path
                .to_str()
                .expect("invalid db path")
                .to_string();

            // Initialize the database (schema migration on startup).
            db::init_db(&db_path_str).expect("failed to initialize local SQLite database");

            // Backend URL — can be overridden via env var.
            let backend_url = std::env::var("FERROWIN_BACKEND_URL")
                .unwrap_or_else(|_| "http://localhost:8080".to_string());

            // Start the background sync loop.
            sync::start_sync_loop(app_handle, db_path_str.clone(), backend_url.clone());

            // Manage DbState
            let db_state = DbState {
                db_path: db_path_str,
                backend_url,
            };
            app.manage(db_state);

            // Manage AuthState (initially empty — restored from localStorage by frontend)
            app.manage(auth::AuthState::new());

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::{init_db, get_pending_sales, get_ultimo_registro_encadenado, get_sale_by_id, insert_offline_sale, insert_offline_sale_item, insert_caja_apertura, insert_caja_movimiento, insert_offline_sale_payment, get_db_size, OfflineSale, OfflineSaleItem};

    fn get_temp_db_path() -> (String, std::path::PathBuf) {
        let uuid = uuid::Uuid::new_v4().to_string();
        let mut path = std::env::temp_dir();
        path.push(format!("ferrowin_lib_test_{}.db", uuid));
        let path_str = path.to_str().unwrap().to_string();
        (path_str, path)
    }

    #[test]
    fn test_offline_chaining_integrity() {
        let (db_path_str, db_path) = get_temp_db_path();
        
        // Initialize DB
        init_db(&db_path_str).unwrap();

        // 1. Create first sale
        let sale1 = OfflineSale {
            id: "sale-001".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 100.0,
            created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-1".into(),
            invoice_number: "TPV-0001".into(),
            sequence_number: 1,
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
        let items1 = vec![OfflineSaleItem {
            id: "item-1".into(),
            offline_sale_id: "sale-001".into(),
            item_id: "prod-1".into(),
            quantity: 1.0,
            unit_price: 100.0,
            discount_percent: 0.0,
        }];

        // Save first sale
        save_offline_sale_impl(sale1, items1, &db_path_str).unwrap();

        // 2. Create second sale
        let sale2 = OfflineSale {
            id: "sale-002".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 150.0,
            created_at: "2026-06-05T10:05:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-2".into(),
            invoice_number: "TPV-0002".into(),
            sequence_number: 2,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 150.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        let items2 = vec![OfflineSaleItem {
            id: "item-2".into(),
            offline_sale_id: "sale-002".into(),
            item_id: "prod-2".into(),
            quantity: 1.0,
            unit_price: 150.0,
            discount_percent: 0.0,
        }];

        // Save second sale
        save_offline_sale_impl(sale2, items2, &db_path_str).unwrap();

        // Verify chaining in SQLite
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        
        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 2);

        let p1 = pending.iter().find(|s| s.id == "sale-001").unwrap();
        let p2 = pending.iter().find(|s| s.id == "sale-002").unwrap();

        // Sale 1 should have no hash_anterior
        assert!(p1.hash_anterior.is_none());
        assert!(p1.firma_registro.is_some());

        // Sale 2 should have hash_anterior matching Sale 1's signature
        assert_eq!(p2.hash_anterior, p1.firma_registro);
        assert!(p2.firma_registro.is_some());

        // Verify the latest chained signature matches Sale 2's signature
        let last_sig = get_ultimo_registro_encadenado(&conn).unwrap();
        assert_eq!(last_sig, p2.firma_registro);

        // Cleanup
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_save_offline_sale_decrements_stock_cache() {
        let (db_path_str, db_path) = get_temp_db_path();
        
        // Initialize DB
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();

        // Scenario A: Pre-cached product
        let item_id_cached = "prod-cached";
        crate::db::upsert_stock_cache(&conn, item_id_cached, 10.0, "2026-06-05T10:00:00Z").unwrap();

        // Scenario B: New product (not in cache)
        let item_id_new = "prod-new";

        // Drop connection so save_offline_sale_impl can open its own connection/transaction
        drop(conn);

        // Record a sale for both items
        let sale = OfflineSale {
            id: "sale-003".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 25.0,
            created_at: "2026-06-05T10:10:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-3".into(),
            invoice_number: "TPV-0003".into(),
            sequence_number: 3,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 25.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        let items = vec![
            OfflineSaleItem {
                id: "item-3".into(),
                offline_sale_id: "sale-003".into(),
                item_id: item_id_cached.into(),
                quantity: 3.0,
                unit_price: 5.0,
                discount_percent: 0.0,
            },
            OfflineSaleItem {
                id: "item-4".into(),
                offline_sale_id: "sale-003".into(),
                item_id: item_id_new.into(),
                quantity: 2.0,
                unit_price: 5.0,
                discount_percent: 0.0,
            },
        ];

        save_offline_sale_impl(sale, items, &db_path_str).unwrap();

        // Verify stock cache levels
        let conn2 = rusqlite::Connection::open(&db_path_str).unwrap();
        let stock_cached = crate::db::get_cached_stock(&conn2, item_id_cached).unwrap();
        let stock_new = crate::db::get_cached_stock(&conn2, item_id_new).unwrap();

        assert_eq!(stock_cached, Some(7.0));
        assert_eq!(stock_new, Some(-2.0));

        // Cleanup
        drop(conn2);
        let _ = std::fs::remove_file(db_path);
    }

    // -----------------------------------------------------------------------
    // Phase 2: Void chain integrity tests
    // -----------------------------------------------------------------------

    #[test]
    fn test_void_chain_integrity() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        // Setup stock
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-1", 10.0, "2026-06-05T10:00:00Z").unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-2", 10.0, "2026-06-05T10:00:00Z").unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-3", 10.0, "2026-06-05T10:00:00Z").unwrap();
        drop(conn);

        // Sale 1 (first in chain)
        let sale1 = OfflineSale {
            id: "sale-001".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 100.0, created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-1".into(),
            invoice_number: "TPV-0001".into(), sequence_number: 1,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 100.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        let items1 = vec![OfflineSaleItem {
            id: "item-1".into(), offline_sale_id: "sale-001".into(),
            item_id: "prod-1".into(), quantity: 1.0, unit_price: 100.0, discount_percent: 0.0,
        }];
        save_offline_sale_impl(sale1, items1, &db_path_str).unwrap();

        // Sale 2 (middle — to be voided)
        let sale2 = OfflineSale {
            id: "sale-002".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 200.0, created_at: "2026-06-05T10:05:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-2".into(),
            invoice_number: "TPV-0002".into(), sequence_number: 2,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 200.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        let items2 = vec![OfflineSaleItem {
            id: "item-2".into(), offline_sale_id: "sale-002".into(),
            item_id: "prod-2".into(), quantity: 1.0, unit_price: 200.0, discount_percent: 0.0,
        }];
        save_offline_sale_impl(sale2, items2, &db_path_str).unwrap();

        // Sale 3 (chain tail)
        let sale3 = OfflineSale {
            id: "sale-003".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 300.0, created_at: "2026-06-05T10:10:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-3".into(),
            invoice_number: "TPV-0003".into(), sequence_number: 3,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 300.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        let items3 = vec![OfflineSaleItem {
            id: "item-3".into(), offline_sale_id: "sale-003".into(),
            item_id: "prod-3".into(), quantity: 1.0, unit_price: 300.0, discount_percent: 0.0,
        }];
        save_offline_sale_impl(sale3, items3, &db_path_str).unwrap();

        // Capture chain state before void
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        let s1_before = get_sale_by_id(&conn, "sale-001").unwrap().unwrap();
        let s2_before = get_sale_by_id(&conn, "sale-002").unwrap().unwrap();
        let s3_before = get_sale_by_id(&conn, "sale-003").unwrap().unwrap();
        let ultimo_before = get_ultimo_registro_encadenado(&conn).unwrap();
        assert_eq!(ultimo_before, s3_before.firma_registro);
        drop(conn);

        // Void the middle sale (sale-002)
        void_sale_impl("sale-002", "Prueba de anulacion", &db_path_str).unwrap();

        // Verify voided sale status
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        let s2_after = get_sale_by_id(&conn, "sale-002").unwrap().unwrap();
        assert_eq!(s2_after.status, "VOIDED");
        assert_eq!(s2_after.void_reason, Some("Prueba de anulacion".into()));
        assert!(s2_after.voided_at.is_some());

        // Verify voided sale's hash_anterior is UNCHANGED
        assert_eq!(s2_after.hash_anterior, s2_before.hash_anterior);
        assert_eq!(s2_after.hash_anterior, s1_before.firma_registro);

        // Verify chain tail (sale-003) is UNCHANGED
        let s3_after = get_sale_by_id(&conn, "sale-003").unwrap().unwrap();
        assert_eq!(s3_after.hash_anterior, s3_before.hash_anterior);
        assert_eq!(s3_after.firma_registro, s3_before.firma_registro);

        // Verify ultimo_registro_encadenado is UNCHANGED (still points to sale-003)
        let ultimo_after = get_ultimo_registro_encadenado(&conn).unwrap();
        assert_eq!(ultimo_after, ultimo_before);
        assert_eq!(ultimo_after, s3_before.firma_registro);

        // Verify stock was restored for sale-002's items
        let stock_prod2 = crate::db::get_cached_stock(&conn, "prod-2").unwrap();
        assert_eq!(stock_prod2, Some(10.0)); // restored from 9 to 10

        // Verify other stock is unchanged
        let stock_prod1 = crate::db::get_cached_stock(&conn, "prod-1").unwrap();
        assert_eq!(stock_prod1, Some(9.0)); // sale-001 consumed 1
        let stock_prod3 = crate::db::get_cached_stock(&conn, "prod-3").unwrap();
        assert_eq!(stock_prod3, Some(9.0)); // sale-003 consumed 1

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_void_rejects_already_voided() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-1", 10.0, "2026-06-05T10:00:00Z").unwrap();
        drop(conn);

        let sale = OfflineSale {
            id: "sale-001".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 100.0, created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-1".into(),
            invoice_number: "TPV-0001".into(), sequence_number: 1,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 100.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        let items = vec![OfflineSaleItem {
            id: "item-1".into(), offline_sale_id: "sale-001".into(),
            item_id: "prod-1".into(), quantity: 1.0, unit_price: 100.0, discount_percent: 0.0,
        }];
        save_offline_sale_impl(sale, items, &db_path_str).unwrap();

        // First void — should succeed
        void_sale_impl("sale-001", "First void", &db_path_str).unwrap();

        // Second void — should fail (already VOIDED)
        let err = void_sale_impl("sale-001", "Second void", &db_path_str).unwrap_err();
        assert!(err.contains("not COMPLETED"));
        assert!(err.contains("VOIDED"));

        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_void_rejects_non_existent_sale() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let err = void_sale_impl("nonexistent", "Test", &db_path_str).unwrap_err();
        assert!(err.contains("not found"));

        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_void_restores_stock() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-1", 50.0, "2026-06-05T10:00:00Z").unwrap();
        crate::db::upsert_stock_cache(&conn, "prod-2", 30.0, "2026-06-05T10:00:00Z").unwrap();
        drop(conn);

        // Create a sale with 2 items
        let sale = OfflineSale {
            id: "sale-001".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 25.0, created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-1".into(),
            invoice_number: "TPV-0001".into(), sequence_number: 1,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 25.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        let items = vec![
            OfflineSaleItem {
                id: "item-1".into(), offline_sale_id: "sale-001".into(),
                item_id: "prod-1".into(), quantity: 3.0, unit_price: 5.0, discount_percent: 0.0,
            },
            OfflineSaleItem {
                id: "item-2".into(), offline_sale_id: "sale-001".into(),
                item_id: "prod-2".into(), quantity: 2.0, unit_price: 5.0, discount_percent: 0.0,
            },
        ];
        save_offline_sale_impl(sale, items, &db_path_str).unwrap();

        // Verify stock decreased
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        assert_eq!(crate::db::get_cached_stock(&conn, "prod-1").unwrap(), Some(47.0));
        assert_eq!(crate::db::get_cached_stock(&conn, "prod-2").unwrap(), Some(28.0));
        drop(conn);

        // Void the sale
        void_sale_impl("sale-001", "Stock test", &db_path_str).unwrap();

        // Verify stock restored
        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        assert_eq!(crate::db::get_cached_stock(&conn, "prod-1").unwrap(), Some(50.0));
        assert_eq!(crate::db::get_cached_stock(&conn, "prod-2").unwrap(), Some(30.0));

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    // -----------------------------------------------------------------------
    // Phase 2: Daily operations commands tests
    // -----------------------------------------------------------------------

    #[test]
    fn test_insert_caja_apertura() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        insert_caja_apertura(&conn, "apt-001", 500.0, "2026-06-05T08:00:00Z").unwrap();

        let count: i64 = conn.query_row(
            "SELECT COUNT(*) FROM caja_aperturas WHERE id = 'apt-001'",
            [], |r| r.get(0),
        ).unwrap();
        assert_eq!(count, 1);

        let amount: f64 = conn.query_row(
            "SELECT amount FROM caja_aperturas WHERE id = 'apt-001'",
            [], |r| r.get(0),
        ).unwrap();
        assert!((amount - 500.0).abs() < f64::EPSILON);

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_insert_caja_movimiento_ingreso_retiro() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();

        // INGRESO
        insert_caja_movimiento(&conn, "mov-001", "INGRESO", "Pago proveedor", 1000.0, "2026-06-05T10:00:00Z").unwrap();
        let count: i64 = conn.query_row(
            "SELECT COUNT(*) FROM caja_movimientos WHERE id = 'mov-001' AND tipo = 'INGRESO'",
            [], |r| r.get(0),
        ).unwrap();
        assert_eq!(count, 1);

        // RETIRO
        insert_caja_movimiento(&conn, "mov-002", "RETIRO", "Gasto menor", 50.0, "2026-06-05T11:00:00Z").unwrap();
        let count: i64 = conn.query_row(
            "SELECT COUNT(*) FROM caja_movimientos WHERE id = 'mov-002' AND tipo = 'RETIRO'",
            [], |r| r.get(0),
        ).unwrap();
        assert_eq!(count, 1);

        // Verify CHECK constraint rejects invalid tipo
        let err = conn.execute(
            "INSERT INTO caja_movimientos (id, tipo, concepto, amount, created_at) VALUES ('mov-003', 'INVALIDO', 'Test', 10.0, '2026-06-05T12:00:00Z')",
            [],
        );
        assert!(err.is_err());

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_insert_offline_sale_payment() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();

        // First insert the sale
        let sale = OfflineSale {
            id: "sale-pay".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 100.0, created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-pay".into(),
            invoice_number: "TPV-0100".into(), sequence_number: 100,
            firma_registro: None, hash_anterior: None, datos_encadenamiento: None,
            subtotal: 100.0, tax_total: 0.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();

        // Insert payment
        insert_offline_sale_payment(&conn, "pay-001", "sale-pay", "Efectivo", 100.0, "2026-06-05T10:01:00Z").unwrap();

        let count: i64 = conn.query_row(
            "SELECT COUNT(*) FROM offline_sale_payments WHERE sale_id = 'sale-pay'",
            [], |r| r.get(0),
        ).unwrap();
        assert_eq!(count, 1);

        // Insert second payment (split payment)
        insert_offline_sale_payment(&conn, "pay-002", "sale-pay", "Tarjeta", 50.0, "2026-06-05T10:02:00Z").unwrap();

        let count: i64 = conn.query_row(
            "SELECT COUNT(*) FROM offline_sale_payments WHERE sale_id = 'sale-pay'",
            [], |r| r.get(0),
        ).unwrap();
        assert_eq!(count, 2);

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_get_db_size_returns_positive() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();
        let size = get_db_size(&conn).unwrap();
        assert!(size > 0, "DB size should be positive, got {}", size);

        drop(conn);
        let _ = std::fs::remove_file(db_path);
    }

    #[test]
    fn test_generate_receipt_pdf_returns_base64() {
        let (db_path_str, db_path) = get_temp_db_path();
        init_db(&db_path_str).unwrap();

        let conn = rusqlite::Connection::open(&db_path_str).unwrap();

        // Insert a sale
        let sale = OfflineSale {
            id: "sale-receipt".into(), terminal_id: "term-1".into(), customer_id: None,
            total: 121.0, created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(), idempotency_key: "key-receipt".into(),
            invoice_number: "TPV-0200".into(), sequence_number: 200,
            firma_registro: Some("abc123def456".into()), hash_anterior: Some("prevhash".into()),
            datos_encadenamiento: Some("chaindata".into()),
            subtotal: 100.0, tax_total: 21.0, discount_total: 0.0,
            status: "COMPLETED".into(), void_reason: None, voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();

        // Add an item
        let item = OfflineSaleItem {
            id: "item-receipt".into(), offline_sale_id: "sale-receipt".into(),
            item_id: "prod-1".into(), quantity: 2.0, unit_price: 50.0, discount_percent: 0.0,
        };
        insert_offline_sale_item(&conn, &item).unwrap();

        // Add a payment
        insert_offline_sale_payment(&conn, "pay-receipt", "sale-receipt", "EFECTIVO", 121.0, "2026-06-05T10:01:00Z").unwrap();

        drop(conn);

        // Generate PDF
        let b64 = generate_receipt_pdf_impl("sale-receipt", &db_path_str).unwrap();
        assert!(!b64.is_empty(), "Base64 output should not be empty");

        // Verify it's valid base64
        use base64::Engine as _;
        let decoded = base64::engine::general_purpose::STANDARD.decode(&b64);
        assert!(decoded.is_ok(), "Output should be valid base64");
        let pdf_bytes = decoded.unwrap();
        assert!(pdf_bytes.len() > 100, "PDF should be larger than 100 bytes");

        // Verify PDF header
        assert_eq!(&pdf_bytes[..5], b"%PDF-", "Should start with PDF magic bytes");

        let _ = std::fs::remove_file(db_path);
    }
}

