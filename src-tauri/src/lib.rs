// VibeX Workbench — Tauri v1 Native Shell
// Entry point: spawns Go backend child process + opens SvelteKit WebView

use serde::{Deserialize, Serialize};
use std::process::Command;
use std::sync::Mutex;
use tauri::{AppHandle, Manager, State};

pub struct AppState {
    pub go_backend_handle: Mutex<Option<GoBackendHandle>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GoBackendHandle {
    pub pid: u32,
    pub port: u16,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MakeResult {
    pub ok: bool,
    pub output: String,
    pub stderr: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SpawnResult {
    pub pid: u32,
    pub port: u16,
}

/// Spawn Go backend as a child process
#[tauri::command]
pub fn spawn_go_backend(state: State<'_, AppState>) -> Result<SpawnResult, String> {
    let exe_dir = std::env::current_exe()
        .map(|p| p.parent().map(|p| p.to_path_buf()))
        .map_err(|e| e.to_string())?
        .ok_or("Could not determine executable directory")?;

    let backend_bin = find_backend_binary(&exe_dir)
        .ok_or_else(|| "Go backend binary not found. Build it: cd backend && go build -o vibex-backend .".to_string())?;

    // Pick an available port
    let port = find_available_port();

    log::info!("Spawning Go backend: {:?} on port {}", backend_bin, port);

    let mut child = Command::new(&backend_bin)
        .env("PORT", port.to_string())
        .spawn()
        .map_err(|e| format!("Failed to spawn Go backend: {}", e))?;

    let pid = child.id();

    // Wait briefly to confirm it started
    std::thread::sleep(std::time::Duration::from_millis(500));

    // Check if still alive
    match child.try_wait() {
        Ok(Some(status)) => {
            return Err(format!("Go backend exited immediately with: {}", status));
        }
        _ => {}
    }

    let handle = GoBackendHandle { pid, port };
    *state.go_backend_handle.lock().unwrap() = Some(handle.clone());

    log::info!("Go backend spawned with PID {}", handle.pid);
    Ok(SpawnResult { pid, port })
}

/// Kill the Go backend child process
#[tauri::command]
pub fn kill_go_backend(state: State<'_, AppState>) -> Result<(), String> {
    let mut guard = state.go_backend_handle.lock().unwrap();
    if let Some(handle) = guard.take() {
        kill_process(handle.pid)?;
        log::info!("Go backend (PID {}) killed", handle.pid);
    }
    Ok(())
}

/// Run a make target in the given workspace
#[tauri::command]
pub fn run_make(target: String, workspace: String) -> Result<MakeResult, String> {
    log::info!("Running make {} in {}", target, workspace);
    let output = Command::new("make")
        .arg(&target)
        .current_dir(&workspace)
        .output()
        .map_err(|e| format!("Failed to run make: {}", e))?;

    Ok(MakeResult {
        ok: output.status.success(),
        output: String::from_utf8_lossy(&output.stdout).to_string(),
        stderr: String::from_utf8_lossy(&output.stderr).to_string(),
    })
}

/// Open native directory picker via system command
#[tauri::command]
pub fn open_directory_picker() -> Result<String, String> {
    #[cfg(target_os = "macos")]
    {
        let output = Command::new("osascript")
            .args(["-e", "POSIX path of (choose folder)"])
            .output()
            .map_err(|e| e.to_string())?;
        let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
        if path.is_empty() {
            Err("No directory selected".to_string())
        } else {
            Ok(path)
        }
    }
    #[cfg(target_os = "linux")]
    {
        // Try zenity, otherwise fall back to error
        let output = Command::new("zenity")
            .args(["--file-selection", "--directory"])
            .output()
            .map_err(|e| e.to_string())?;
        let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
        if path.is_empty() {
            Err("No directory selected".to_string())
        } else {
            Ok(path)
        }
    }
    #[cfg(target_os = "windows")]
    {
        Err("Windows directory picker not yet implemented".to_string())
    }
}

fn find_backend_binary(exe_dir: &std::path::Path) -> Option<std::path::PathBuf> {
    let candidates = [
        exe_dir.join("backend").join("vibex-backend.exe"),
        exe_dir.join("backend").join("vibex-backend"),
        exe_dir.join("vibex-backend.exe"),
        exe_dir.join("vibex-backend"),
        exe_dir.join("..").join("backend").join("vibex-backend"),
        exe_dir.join("..").join("..").join("backend").join("vibex-backend"),
        exe_dir.join("..").join("vibex-backend"),
        exe_dir.join("..").join("vibex-backend.exe"),
    ];

    for candidate in candidates.iter() {
        if candidate.exists() {
            log::info!("Found backend binary at {:?}", candidate);
            return Some(candidate.clone());
        }
    }

    // Try PATH
    if let Ok(path) = std::env::var("PATH") {
        for dir in std::env::split_paths(&path) {
            for name in &["vibex-backend", "vibex-backend.exe"] {
                let full = dir.join(name);
                if full.exists() {
                    log::info!("Found backend binary in PATH: {:?}", &full);
                    return Some(full);
                }
            }
        }
    }
    None
}

fn find_available_port() -> u16 {
    // Simple heuristic: try common ports
    for port in [33335, 33336, 33337, 33338, 33339] {
        if is_port_available(port) {
            return port;
        }
    }
    // Fallback: use a socket
    use std::net::TcpListener;
    if let Ok(listener) = TcpListener::bind("127.0.0.1:0") {
        if let Ok(addr) = listener.local_addr() {
            return addr.port();
        }
    }
    33335
}

fn is_port_available(port: u16) -> bool {
    std::net::TcpListener::bind(("127.0.0.1", port)).is_ok()
}

fn kill_process(pid: u32) -> Result<(), String> {
    #[cfg(windows)]
    {
        Command::new("taskkill")
            .args(["/F", "/PID", &pid.to_string()])
            .output()
            .map_err(|e| e.to_string())?;
    }
    #[cfg(not(windows))]
    {
        Command::new("kill")
            .args(["-9", &pid.to_string()])
            .output()
            .map_err(|e| e.to_string())?;
    }
    Ok(())
}

// ── App Entry ─────────────────────────────────────────────────

pub fn run() {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info"))
        .init();

    log::info!("Starting VibeX Workbench (Tauri v1)...");

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(AppState {
            go_backend_handle: Mutex::new(None),
        })
        .invoke_handler(tauri::generate_handler![
            spawn_go_backend,
            kill_go_backend,
            run_make,
            open_directory_picker,
        ])
        .setup(|app| {
            log::info!("Tauri app setup complete, window opened");

            // Auto-spawn Go backend
            let state = app.state::<AppState>();
            match spawn_backend_internal(&state) {
                Ok(handle) => {
                    log::info!("Go backend auto-spawned: PID {} port {}", handle.pid, handle.port);
                }
                Err(e) => {
                    log::warn!("Failed to auto-spawn Go backend: {}", e);
                }
            }

            Ok(())
        })
        .on_window_event(|event| {
            if let tauri::WindowEvent::CloseRequested { .. } = event.event {
                log::info!("Window close requested, killing Go backend...");
                let state = event.window().state::<AppState>();
                if let Some(handle) = state.go_backend_handle.lock().unwrap().take() {
                    let _ = kill_process(handle.pid);
                }
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn spawn_backend_internal(state: &State<'_, AppState>) -> Result<GoBackendHandle, String> {
    let exe_dir = std::env::current_exe()
        .map(|p| p.parent().map(|p| p.to_path_buf()))
        .map_err(|e| e.to_string())?
        .ok_or("Could not determine executable directory")?;

    let backend_bin = find_backend_binary(&exe_dir)
        .ok_or("Go backend binary not found in any expected location")?;

    let port = find_available_port();

    let child = Command::new(&backend_bin)
        .env("PORT", port.to_string())
        .spawn()
        .map_err(|e| format!("Failed to spawn: {}", e))?;

    std::thread::sleep(std::time::Duration::from_millis(500));

    let handle = GoBackendHandle {
        pid: child.id(),
        port,
    };
    *state.go_backend_handle.lock().unwrap() = Some(handle.clone());
    Ok(handle)
}
