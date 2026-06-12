use std::sync::Mutex;

use tauri::{Manager, RunEvent};
use tauri_plugin_shell::{process::CommandChild, ShellExt};

struct TaskifyApiSidecar(Mutex<Option<CommandChild>>);

// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            let sidecar = app.shell().sidecar("taskify-api")?;
            let (_receiver, child) = sidecar.spawn()?;

            app.manage(TaskifyApiSidecar(Mutex::new(Some(child))));

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![greet])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app_handle, event| {
            if let RunEvent::ExitRequested { .. } = event {
                if let Some(sidecar) = app_handle.try_state::<TaskifyApiSidecar>() {
                    if let Ok(mut child) = sidecar.0.lock() {
                        if let Some(process) = child.take() {
                            let _ = process.kill();
                        }
                    }
                }
            }
        });
}
