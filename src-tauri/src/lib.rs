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

    // FIX: binary moved from backend/ to agent/ (vibex-agent-web)
    let backend_bin = find_backend_binary(&exe_dir)
        .ok_or_else(|| "Go backend binary not found. Build it: cd agent && go build -o vibex-agent-web ./cmd/web/.".to_string())?;

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
        Ok(None) => {} // Still running
        Err(e) => {
            return Err(format!("Failed to check process status: {}", e));
        }
    }

    // Store handle
    let handle = GoBackendHandle { pid, port };
    {
        let mut guard = state.go_backend_handle.lock().unwrap();
        *guard = Some(handle.clone());
    }

    log::info!("Go backend spawned successfully, PID={}, port={}", pid, port);
    Ok(SpawnResult { pid, port })
}

/// Kill the Go backend child process
#[tauri::command]
pub fn kill_go_backend(state: State<'_, AppState>) -> Result<(), String> {
    let mut guard = state.go_backend_handle.lock().unwrap();
    if let Some(handle) = guard.take() {
        kill_process(handle.pid)?;
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

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    log::info!("make {} finished: ok={}", target, output.status.success());

    Ok(MakeResult {
        ok: output.status.success(),
        output: stdout,
        stderr,
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

        if output.status.success() {
            return Ok(String::from_utf8_lossy(&output.stdout).trim().to_string());
        }
    }
    #[cfg(target_os = "linux")]
    {
        // Try zenity, otherwise fall back to error
        let output = Command::new("zenity")
            .args(["--file-selection", "--directory"])
            .output()
            .map_err(|e| e.to_string())?;

        if output.status.success() {
            return Ok(String::from_utf8_lossy(&output.stdout).trim().to_string());
        }
        return Err("Directory picker not available (zenity required on Linux)".to_string());
    }
    #[cfg(target_os = "windows")]
    {
        let output = Command::new("powershell")
            .args(["-Command", "Add-Type -AssemblyName System.Windows.Forms; [Windows.Forms.FolderBrowserDialog]::new().SelectedPath"])
            .output()
            .map_err(|e| e.to_string())?;

        if output.status.success() {
            return Ok(String::from_utf8_lossy(&output.stdout).trim().to_string());
        }
    }

    Err("Directory picker not supported on this platform".to_string())
}

/// Find the Go backend binary by searching relative to the executable
fn find_backend_binary(exe_dir: &std::path::Path) -> Option<std::path::PathBuf> {
    // FIX: binary moved from backend/ to agent/ (vibex-agent-web)
    let candidates = [
        exe_dir.join("agent").join("vibex-agent-web"),
        exe_dir.join("agent").join("vibex-agent-web.exe"),
        exe_dir.join("vibex-agent-web.exe"),
        exe_dir.join("vibex-agent-web"),
        exe_dir.join("..").join("agent").join("vibex-agent-web"),
        exe_dir.join("..").join("..").join("agent").join("vibex-agent-web"),
        exe_dir.join("vibex-backend"),
        exe_dir.join("vibex-backend.exe"),
        exe_dir.join("..").join("backend").join("vibex-backend"),
        exe_dir.join("..").join("..").join("backend").join("vibex-backend"),
    ];

    for candidate in candidates.iter() {
        if candidate.exists() {
            log::info!("Found backend binary: {:?}", candidate);
            return Some(candidate.clone());
        }
    }

    // Try PATH
    if let Ok(path) = std::env::var("PATH") {
        for dir in std::env::split_paths(&path) {
            for name in &["vibex-agent-web", "vibex-agent-web.exe", "vibex-backend", "vibex-backend.exe"] {
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
    for port in [33338, 33336, 33337, 33339] {
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
    33338
}

fn is_port_available(port: u16) -> bool {
    std::net::TcpListener::bind(("127.0.0.1", port)).is_ok()
}

fn kill_process(pid: u32) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        Command::new("taskkill")
            .args(["/F", "/PID", &pid.to_string()])
            .output()
            .map_err(|e| e.to_string())?;
    }
    #[cfg(not(target_os = "windows"))]
    {
        Command::new("kill")
            .arg("-9")
            .arg(pid.to_string())
            .output()
            .map_err(|e| e.to_string())?;
    }
    Ok(())
}

pub fn run() {
    env_logger::init();

    log::info!("Starting VibeX Workbench (Tauri v1)...");

    tauri::Builder::default()
        .manage(AppState {
            go_backend_handle: Mutex::new(None),
        })
        .invoke_handler(tauri::generate_handler![
            spawn_go_backend,
            kill_go_backend,
            run_make,
            open_directory_picker,
        ])
        .setup(|_app| {
            log::info!("Tauri v1 app setup complete");
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

