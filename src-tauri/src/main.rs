// VibeX Workbench — Tauri Native Shell
// Entry point: spawns Go backend child process + opens SvelteKit WebView
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    vibex_workbench_lib::run();
}
