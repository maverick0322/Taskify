use std::{env, sync::Mutex};

use keyring::Entry;
use serde::{Deserialize, Serialize};
use tauri::{Manager, RunEvent};
use tauri_plugin_shell::{process::CommandChild, ShellExt};

struct TaskifyBackendSidecar(Mutex<Option<CommandChild>>);

const DEFAULT_ACCESS_TOKEN_TTL: &str = "5m";
const DEFAULT_REFRESH_TOKEN_TTL: &str = "24h";
const DEFAULT_PORT: &str = "8080";
const DEFAULT_BCRYPT_COST: &str = "10";
const SESSION_SERVICE_NAME: &str = "Taskify";
const SESSION_ACCOUNT_NAME: &str = "desktop-session";

#[derive(Serialize, Deserialize)]
struct StoredSession {
    access_token: String,
    refresh_token: String,
}

// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}

#[tauri::command]
fn set_secure_session(access_token: String, refresh_token: String) -> Result<(), String> {
    let session = StoredSession {
        access_token,
        refresh_token,
    };
    let serialized =
        serde_json::to_string(&session).map_err(|error| format!("serialize session: {error}"))?;

    session_entry()?
        .set_password(&serialized)
        .map_err(|error| format!("store secure session: {error}"))
}

#[tauri::command]
fn get_secure_session() -> Result<Option<StoredSession>, String> {
    let entry = session_entry()?;
    match entry.get_password() {
        Ok(value) => {
            let session = serde_json::from_str::<StoredSession>(&value)
                .map_err(|error| format!("parse secure session: {error}"))?;
            Ok(Some(session))
        }
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(error) => Err(format!("read secure session: {error}")),
    }
}

#[tauri::command]
fn clear_secure_session() -> Result<(), String> {
    let entry = session_entry()?;
    match entry.delete_credential() {
        Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
        Err(error) => Err(format!("delete secure session: {error}")),
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            let sidecar = app
                .shell()
                .sidecar("backend")?
                .env(
                    "ACCESS_TOKEN_TTL",
                    env_value("ACCESS_TOKEN_TTL", DEFAULT_ACCESS_TOKEN_TTL),
                )
                .env(
                    "REFRESH_TOKEN_TTL",
                    env_value("REFRESH_TOKEN_TTL", DEFAULT_REFRESH_TOKEN_TTL),
                )
                .env("PORT", env_value("PORT", DEFAULT_PORT))
                .env("BCRYPT_COST", env_value("BCRYPT_COST", DEFAULT_BCRYPT_COST))
                .env("REMOTE_API_URL", env_value("REMOTE_API_URL", ""));
            let (_receiver, child) = sidecar.spawn()?;

            app.manage(TaskifyBackendSidecar(Mutex::new(Some(child))));

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            greet,
            set_secure_session,
            get_secure_session,
            clear_secure_session
        ])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app_handle, event| {
            if let RunEvent::ExitRequested { .. } = event {
                if let Some(sidecar) = app_handle.try_state::<TaskifyBackendSidecar>() {
                    if let Ok(mut child) = sidecar.0.lock() {
                        if let Some(process) = child.take() {
                            let _ = process.kill();
                        }
                    }
                }
            }
        });
}

fn env_value(key: &str, fallback: &str) -> String {
    env::var(key).unwrap_or_else(|_| fallback.to_string())
}

fn session_entry() -> Result<Entry, String> {
    Entry::new(SESSION_SERVICE_NAME, SESSION_ACCOUNT_NAME)
        .map_err(|error| format!("open secure session entry: {error}"))
}
