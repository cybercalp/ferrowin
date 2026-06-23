use crate::DbState;
use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use tauri::State;

/// Managed Tauri state for authentication.
pub struct AuthState {
    pub token: Mutex<Option<String>>,
    pub user: Mutex<Option<UserInfo>>,
}

impl AuthState {
    pub fn new() -> Self {
        Self {
            token: Mutex::new(None),
            user: Mutex::new(None),
        }
    }
}

/// Public user information returned from login.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub id: String,
    pub username: String,
}

/// Response from the Go backend login endpoint.
#[derive(Debug, Clone, Serialize, Deserialize)]
struct BackendLoginResponse {
    token: String,
    user: BackendUserInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct BackendUserInfo {
    id: String,
    username: String,
}

/// Authenticates the user against the Go backend, stores the token in AuthState,
/// and returns the user info to the frontend.
#[tauri::command]
pub async fn login(
    username: String,
    password: String,
    auth: State<'_, AuthState>,
    db_state: State<'_, DbState>,
) -> Result<UserInfo, String> {
    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .map_err(|e| format!("Failed to build HTTP client: {}", e))?;

    let body = serde_json::json!({
        "username": username,
        "password": password,
    });

    let resp = client
        .post(format!("{}/api/v1/auth/login", db_state.backend_url))
        .json(&body)
        .send()
        .await
        .map_err(|_| "Invalid credentials".to_string())?;

    if !resp.status().is_success() {
        return Err("Invalid credentials".to_string());
    }

    let backend: BackendLoginResponse = resp
        .json()
        .await
        .map_err(|_| "Invalid credentials".to_string())?;

    let user_info = UserInfo {
        id: backend.user.id,
        username: backend.user.username,
    };

    // Store token and user in managed state
    if let Ok(mut t) = auth.token.lock() {
        *t = Some(backend.token);
    }
    if let Ok(mut u) = auth.user.lock() {
        *u = Some(user_info.clone());
    }

    Ok(user_info)
}

/// Restores the auth state from data persisted in localStorage (called on app init).
#[tauri::command]
pub fn set_auth_state(
    token: String,
    user_id: String,
    username: String,
    auth: State<'_, AuthState>,
) {
    if let Ok(mut t) = auth.token.lock() {
        *t = Some(token);
    }
    if let Ok(mut u) = auth.user.lock() {
        *u = Some(UserInfo {
            id: user_id,
            username,
        });
    }
}

/// Clears the auth state (logout).
#[tauri::command]
pub fn clear_auth(auth: State<'_, AuthState>) {
    if let Ok(mut t) = auth.token.lock() {
        *t = None;
    }
    if let Ok(mut u) = auth.user.lock() {
        *u = None;
    }
}

/// Returns the stored auth token (for frontend to persist in localStorage).
#[tauri::command]
pub fn get_auth_token(auth: State<'_, AuthState>) -> Result<String, String> {
    auth.token
        .lock()
        .map_err(|e| format!("Failed to read auth token: {}", e))?
        .clone()
        .ok_or_else(|| "Not authenticated".to_string())
}

/// Returns the stored auth token, if any (non-command helper).
pub fn get_token(auth: &AuthState) -> Option<String> {
    auth.token.lock().ok().and_then(|t| t.clone())
}

/// Creates a `reqwest::Client` with the Authorization header set to the stored Bearer token,
/// or a plain client if no token is set.
pub fn get_authorized_client(auth: &AuthState) -> Result<reqwest::Client, String> {
    let mut headers = reqwest::header::HeaderMap::new();

    if let Some(token) = get_token(auth) {
        let bearer = format!("Bearer {}", token);
        let mut auth_value = reqwest::header::HeaderValue::from_str(&bearer)
            .map_err(|e| format!("Failed to set auth header: {}", e))?;
        auth_value.set_sensitive(true);
        headers.insert(reqwest::header::AUTHORIZATION, auth_value);
    }

    reqwest::Client::builder()
        .default_headers(headers)
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .map_err(|e| format!("Failed to build HTTP client: {}", e))
}
