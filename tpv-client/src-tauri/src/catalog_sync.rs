use crate::db;
use rusqlite::Connection;

#[derive(Debug, Clone, Default, serde::Deserialize)]
pub struct EliminadosResponse {
    #[serde(default)]
    pub productos: Option<Vec<String>>,
    #[serde(default)]
    pub familias: Option<Vec<String>>,
    #[serde(default)]
    pub tipos_iva: Option<Vec<String>>,
    /// Kept for forward-compat with the Go backend response.
    /// Client data is NOT saved locally — it was removed in Phase 2.
    #[serde(default)]
    pub clientes: Option<Vec<String>>,
}

#[derive(Debug, Clone, serde::Deserialize)]
pub struct CatalogSyncResponse {
    #[serde(default)]
    pub tipos_iva: Option<Vec<db::TipoIVA>>,
    #[serde(default)]
    pub familias: Option<Vec<db::Familia>>,
    #[serde(default)]
    pub productos: Option<Vec<db::Producto>>,
    #[serde(default)]
    pub clientes: Option<Vec<db::Cliente>>,
    #[serde(default)]
    pub eliminados: Option<EliminadosResponse>,
}

/// Fetches the catalog delta since the last sync timestamp and updates the SQLite DB in a transaction.
/// Accepts an optional bearer token for authenticated requests.
pub async fn sync_catalog_delta(db_path: &str, backend_url: &str, token: Option<String>) -> Result<(), String> {
    let mut conn = Connection::open(db_path)
        .map_err(|e| format!("Failed to open DB: {}", e))?;

    let since_opt = db::get_ultimo_sync_catalogo(&conn)
        .map_err(|e| format!("Failed to get last sync time: {}", e))?;

    let mut headers = reqwest::header::HeaderMap::new();
    if let Some(ref t) = token {
        let bearer = format!("Bearer {}", t);
        if let Ok(mut val) = reqwest::header::HeaderValue::from_str(&bearer) {
            val.set_sensitive(true);
            headers.insert(reqwest::header::AUTHORIZATION, val);
        }
    }

    let client = reqwest::Client::builder()
        .default_headers(headers)
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .map_err(|e| format!("Failed to build reqwest client: {}", e))?;

    let mut req = client.get(format!("{}/api/v1/catalog/sync", backend_url));
    if let Some(since) = &since_opt {
        if !since.is_empty() {
            req = req.query(&[("since", since)]);
        }
    }

    let sync_start_time = chrono::Utc::now().to_rfc3339();

    let resp = req.send().await
        .map_err(|e| format!("Network error sending sync request: {}", e))?;

    if !resp.status().is_success() {
        return Err(format!("Server returned error status: {}", resp.status()));
    }

    let body_text = resp.text().await
        .map_err(|e| format!("Failed to read response body: {}", e))?;

    let sync_data: CatalogSyncResponse = serde_json::from_str(&body_text)
        .map_err(|e| format!("Failed to parse catalog sync response: {}", e))?;

    let tx = conn.transaction()
        .map_err(|e| format!("Failed to begin transaction: {}", e))?;

    // Upsert catalog items
    for item in sync_data.tipos_iva.iter().flatten() {
        db::upsert_tipo_iva(&tx, item)
            .map_err(|e| format!("Failed to upsert tipo_iva: {}", e))?;
    }

    for item in sync_data.familias.iter().flatten() {
        db::upsert_familia(&tx, item)
            .map_err(|e| format!("Failed to upsert familia: {}", e))?;
    }

    for item in sync_data.productos.iter().flatten() {
        db::upsert_producto(&tx, item)
            .map_err(|e| format!("Failed to upsert producto: {}", e))?;
    }

    for item in sync_data.clientes.iter().flatten() {
        db::upsert_cliente(&tx, item)
            .map_err(|e| format!("Failed to upsert cliente: {}", e))?;
    }

    // Process deactivated/deleted items
    for id in sync_data.eliminados.iter().flat_map(|e| e.tipos_iva.iter()).flatten() {
        db::deactivate_tipo_iva(&tx, id)
            .map_err(|e| format!("Failed to deactivate tipo_iva: {}", e))?;
    }

    for id in sync_data.eliminados.iter().flat_map(|e| e.familias.iter()).flatten() {
        db::deactivate_familia(&tx, id)
            .map_err(|e| format!("Failed to deactivate familia: {}", e))?;
    }

    for id in sync_data.eliminados.iter().flat_map(|e| e.productos.iter()).flatten() {
        db::deactivate_producto(&tx, id)
            .map_err(|e| format!("Failed to deactivate producto: {}", e))?;
    }

    // Save metadata sync timestamp
    db::set_ultimo_sync_catalogo(&tx, &sync_start_time)
        .map_err(|e| format!("Failed to update last sync time: {}", e))?;

    // Rebuild FTS index after bulk upserts
    db::reindex_productos_fts(&tx)
        .map_err(|e| format!("Failed to rebuild FTS index: {}", e))?;

    tx.commit().map_err(|e| format!("Failed to commit transaction: {}", e))?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::net::TcpListener;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    fn get_temp_db_path() -> (String, std::path::PathBuf) {
        let uuid = uuid::Uuid::new_v4().to_string();
        let mut path = std::env::temp_dir();
        path.push(format!("ferrowin_catalog_test_{}.db", uuid));
        let path_str = path.to_str().unwrap().to_string();
        (path_str, path)
    }

    async fn spawn_mock_catalog_server(response_body: String) -> (String, tokio::task::JoinHandle<()>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let port = listener.local_addr().unwrap().port();
        let address = format!("http://127.0.0.1:{}", port);
        
        let handle = tokio::spawn(async move {
            if let Ok((mut stream, _)) = listener.accept().await {
                let mut buf = vec![0; 4096];
                let _ = stream.read(&mut buf).await;
                
                let response = format!(
                    "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{}",
                    response_body.len(),
                    response_body
                );
                let _ = stream.write_all(response.as_bytes()).await;
            }
        });
        
        (address, handle)
    }

    #[tokio::test]
    async fn test_sync_catalog_delta_flow() {
        let (db_path_str, db_path) = get_temp_db_path();
        db::init_db(&db_path_str).unwrap();

        let mock_json = r#"{
            "tipos_iva": [
                {"id": "iva-21", "nombre": "IVA 21%", "porcentaje": 21.0, "updated_at": "2026-06-05T12:00:00Z", "activo": true}
            ],
            "familias": [
                {"id": "fam-1", "nombre": "Familia 1", "updated_at": "2026-06-05T12:00:00Z", "activo": true}
            ],
            "productos": [
                {"id": "prod-1", "codigo": "P001", "nombre": "Producto 1", "precio_venta": 10.5, "familia_id": "fam-1", "tipo_iva_id": "iva-21", "updated_at": "2026-06-05T12:00:00Z", "activo": true}
            ],
            "eliminados": {
                "productos": [],
                "clientes": [],
                "familias": [],
                "tipos_iva": []
            }
        }"#.to_string();

        let (mock_url, server_handle) = spawn_mock_catalog_server(mock_json).await;

        sync_catalog_delta(&db_path_str, &mock_url, None).await.unwrap();

        // Verify SQLite contents
        let conn = Connection::open(&db_path_str).unwrap();
        let prod_name: String = conn.query_row(
            "SELECT nombre FROM productos WHERE id = 'prod-1'",
            [],
            |r| r.get(0)
        ).unwrap();
        assert_eq!(prod_name, "Producto 1");

        let last_sync = db::get_ultimo_sync_catalogo(&conn).unwrap();
        assert!(last_sync.is_some());

        server_handle.abort();
        let _ = std::fs::remove_file(db_path);
    }
}
