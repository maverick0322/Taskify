use std::{env, sync::Mutex};

use tauri::{Manager, RunEvent};
use tauri_plugin_shell::{process::CommandChild, ShellExt};

struct TaskifyBackendSidecar(Mutex<Option<CommandChild>>);

const DEFAULT_JWT_SECRET: &str = "taskify-local-desktop-secret-change-me";
const DEFAULT_ACCESS_TOKEN_TTL: &str = "5m";
const DEFAULT_REFRESH_TOKEN_TTL: &str = "24h";
const DEFAULT_PORT: &str = "8080";
const DEFAULT_BCRYPT_COST: &str = "10";

// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
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
                .env("JWT_SECRET", env_value("JWT_SECRET", DEFAULT_JWT_SECRET))
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
                .env("REMOTE_DB_URL", env_value("REMOTE_DB_URL", ""))
                .env("SUPABASE_URL", env_value("SUPABASE_URL", ""))
                .env(
                    "SUPABASE_SERVICE_ROLE_KEY",
                    env_value("SUPABASE_SERVICE_ROLE_KEY", ""),
                );
            let (_receiver, child) = sidecar.spawn()?;

            app.manage(TaskifyBackendSidecar(Mutex::new(Some(child))));

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![greet])
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
