#!/usr/bin/env python3
"""
spec-to-tauri v0.1
从 Tauri DSL spec 生成 Tauri 2.x 项目文件

用法:
  python3 gen.py <spec_dir> <output_dir>
"""

import sys
import json
import yaml
from pathlib import Path

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")
OUT_DIR = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("desktop")
DATE = __import__('datetime').date.today()


def load_yaml(path: Path) -> dict:
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)


def ensure_dir(path: Path):
    path.mkdir(parents=True, exist_ok=True)


def gen_cargo_toml(app_name: str) -> str:
    return f"""[package]
name = "{app_name}-desktop"
version = "0.1.0"
edition = "2021"

[lib]
name = "vibex_desktop_lib"
crate-type = ["staticlib", "cdylib", "rlib"]

[build-dependencies]
tauri-build = {{ version = "2", features = [] }}

[dependencies]
tauri = {{ version = "2", features = ["tray-icon"] }}
tauri-plugin-shell = "2"
tauri-plugin-fs = {{ version = "2", features = ["scope"] }}
tauri-plugin-dialog = "2"
tauri-plugin-notification = "2"
tauri-plugin-clipboard-manager = "2"
tauri-plugin-process = "2"
tauri-plugin-opener = "2"
serde = {{ version = "1", features = ["derive"] }}
serde_json = "1"
log = "0.4"
env_logger = "0.11"

[features]
default = ["custom-protocol"]
custom-protocol = ["tauri/custom-protocol"]

[profile.release]
panic = "abort"
codegen-units = 1
lto = true
opt-level = "s"
strip = true
"""


def gen_tauri_conf(app_name: str, spec_dir: Path) -> dict:
    conf = {
        "$schema": "https://schema.tauri.app/config/2",
        "productName": "VibeX",
        "identifier": "com.vibex.desktop",
        "version": "0.1.0",
        "build": {
            "devtools": True,
            "beforeDevCommand": "cd ../frontend && npm run dev",
            "beforeBuildCommand": "cd ../frontend && npm run build",
            "devUrl": "http://localhost:5173",
            "frontendDist": "../frontend/.svelte-kit/output",
        },
        "app": {
            "withGlobalTauri": True,
            "windows": [
                {
                    "title": "VibeX",
                    "width": 1280,
                    "height": 800,
                    "minWidth": 900,
                    "minHeight": 600,
                    "resizable": True,
                    "fullscreen": False,
                    "decorations": True,
                    "center": True,
                    "dragDropEnabled": True,
                }
            ],
            "security": {
                "csp": (
                    "default-src 'self'; "
                    "script-src 'self' 'unsafe-inline'; "
                    "style-src 'self' 'unsafe-inline'; "
                    "img-src 'self' asset: https: data:; "
                    "connect-src 'self' ipc: http://ipc.localhost"
                )
            },
            "trayIcon": {
                "iconPath": "icons/icon.png",
                "iconAsTemplate": True,
                "id": "main"
            },
        },
        "bundle": {
            "active": True,
            "targets": ["msi", "nsis"],
            "icon": [
                "icons/32x32.png",
                "icons/128x128.png",
                "icons/128x128@2x.png",
                "icons/icon.icns",
                "icons/icon.ico",
            ],
            "windows": {
                "certificateThumbprint": None,
                "digestAlgorithm": "sha256",
                "timestampUrl": "",
            },
        },
        "plugins": {
            "shell": {"open": True},
            "fs": {
                "scope": {
                    "allow": [
                        "$HOME/.vibex/**",
                        "$APPDATA/**",
                        "$LOCALAPPDATA/**",
                    ],
                    "deny": ["$HOME/.vibex/secrets/**"],
                }
            },
        },
    }

    # 从 spec 覆盖配置
    for tauri_file in spec_dir.rglob("canvas_tauri.yaml"):
        data = load_yaml(tauri_file)
        app_cfg = data.get("content", {}).get("app", {})
        build_cfg = data.get("content", {}).get("build", {})
        plugins_cfg = data.get("content", {}).get("plugins", {})

        if "windows" in app_cfg:
            conf["app"]["windows"] = app_cfg["windows"]
        if "productName" in app_cfg:
            conf["productName"] = app_cfg["productName"]
        if "identifier" in app_cfg:
            conf["identifier"] = app_cfg["identifier"]
        if "security" in app_cfg:
            conf["app"]["security"] = app_cfg["security"]
        if "beforeDevCommand" in build_cfg:
            conf["build"]["beforeDevCommand"] = build_cfg["beforeDevCommand"]
        if "beforeBuildCommand" in build_cfg:
            conf["build"]["beforeBuildCommand"] = build_cfg["beforeBuildCommand"]
        if plugins_cfg:
            conf["plugins"] = {**conf.get("plugins", {}), **plugins_cfg}

    return conf


def gen_capabilities() -> list[dict]:
    return [
        {
            "$schema": "https://schemas.tauri.app/config/2/capability",
            "identifier": "default",
            "description": "Default capabilities for the main window",
            "windows": ["main"],
            "permissions": [
                "core:default",
                "core:window:default",
                "core:window:allow-close",
                "core:window:allow-minimize",
                "core:window:allow-maximize",
                "core:window:allow-set-title",
                "core:window:allow-set-fullscreen",
                "core:window:allow-set-always-on-top",
                "core:window:allow-is-maximized",
                "core:window:allow-is-minimized",
                "core:window:allow-is-fullscreen",
                "core:tray:default",
                "core:tray:allow-new",
                "core:tray:allow-set-icon",
                "core:tray:allow-set-menu",
                "core:tray:allow-set-tooltip",
                "shell:allow-open",
                "shell:default",
                "fs:default",
                "fs:allow-app-read-recursive",
                "fs:allow-app-write-recursive",
                "fs:allow-appdata-read-recursive",
                "fs:allow-appdata-write-recursive",
                "fs:allow-appcache-read-recursive",
                "fs:allow-appcache-write-recursive",
                "fs:allow-appconfig-read-recursive",
                "fs:allow-appconfig-write-recursive",
                "fs:allow-applog-read-recursive",
                "fs:allow-applog-write-recursive",
                "fs:allow-home-read-recursive",
                "fs:allow-home-write-recursive",
                "dialog:default",
                "dialog:allow-open",
                "dialog:allow-save",
                "dialog:allow-message",
                "dialog:allow-ask",
                "dialog:allow-confirm",
                "notification:default",
                "notification:allow-is-permission-granted",
                "notification:allow-request-permission",
                "notification:allow-notify",
                "clipboard-manager:default",
                "clipboard-manager:allow-read",
                "clipboard-manager:allow-write",
                "clipboard-manager:allow-write-text",
                "clipboard-manager:allow-read-text",
                "process:default",
                "process:allow-exit",
                "process:allow-restart",
                "opener:default",
            ],
        }
    ]


def gen_main_rs() -> str:
    header = (
        "// ============================================================\n"
        "// This file is auto-generated by spec-to-tauri\n"
        f"// from: {SPEC_DIR}\n"
        f"// date: {DATE}\n"
        "// DO NOT edit directly\n"
        "// ============================================================\n\n"
    )
    body = (
        "// Prevents additional console window on Windows in release\n"
        "#![cfg_attr(not(debug_assertions), windows_subsystem = \"windows\")]\n\n"
        "fn main() {\n"
        "    env_logger::Builder::from_env(\n"
        "        env_logger::Env::default().default_filter_or(\"info\"),\n"
        "    )\n"
        "    .format_timestamp_millis()\n"
        "    .init();\n\n"
        "    log::info!(\"Starting VibeX Desktop...\");\n"
        "    vibex_desktop_lib::run();\n"
        "}\n"
    )
    return header + body


def gen_lib_rs() -> str:
    header = (
        "// ============================================================\n"
        "// This file is auto-generated by spec-to-tauri\n"
        f"// from: {SPEC_DIR}\n"
        f"// date: {DATE}\n"
        "// DO NOT edit directly\n"
        "// ============================================================\n\n"
    )
    body = (
        "use tauri::{\n"
        "    menu::{Menu, MenuItem},\n"
        "    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},\n"
        "    Manager, WindowEvent,\n"
        "};\n"
        "use tauri_plugin_opener::open;\n\n"
        "mod commands;\n\n"
        "#[cfg_attr(mobile, tauri::mobile_entry_point)]\n"
        "pub fn run() {\n"
        "    log::info!(\"Initializing Tauri application...\");\n\n"
        "    let builder = tauri::Builder::default()\n"
        "        .plugin(tauri_plugin_shell::init())\n"
        "        .plugin(tauri_plugin_fs::init())\n"
        "        .plugin(tauri_plugin_dialog::init())\n"
        "        .plugin(tauri_plugin_notification::init())\n"
        "        .plugin(tauri_plugin_clipboard_manager::init())\n"
        "        .plugin(tauri_plugin_process::init())\n"
        "        .plugin(tauri_plugin_opener::init())\n"
        "        .invoke_handler(tauri::generate_handler![\n"
        "            commands::save_canvas_snapshot,\n"
        "            commands::load_canvas_snapshot,\n"
        "            commands::export_canvas,\n"
        "            commands::get_app_data_dir,\n"
        "        ])\n"
        "        .setup(|app| {\n"
        "            log::info!(\"Setting up application...\");\n\n"
        "            // System Tray\n"
        "            let quit = MenuItem::with_id(app, \"quit\", \"Quit\", true, None::<&str>)?;\n"
        "            let show = MenuItem::with_id(app, \"show\", \"Show\", true, None::<&str>)?;\n"
        "            let hide = MenuItem::with_id(app, \"hide\", \"Hide\", true, None::<&str>)?;\n"
        "            let about = MenuItem::with_id(app, \"about\", \"About\", true, None::<&str>)?;\n\n"
        "            let menu = Menu::with_items(app, &[&show, &hide, &about, &quit])?;\n\n"
        "            let _tray = TrayIconBuilder::with_id(\"main\")\n"
        "                .tooltip(\"VibeX Desktop\")\n"
        "                .menu(&menu)\n"
        "                .menu_on_left_click(false)\n"
        "                .on_menu_event(|app, event| {\n"
        "                    match event.id.as_ref() {\n"
        "                        \"quit\" => {\n"
        "                            log::info!(\"Quit from tray\");\n"
        "                            app.exit(0);\n"
        "                        }\n"
        "                        \"show\" => {\n"
        "                            if let Some(window) = app.get_webview_window(\"main\") {\n"
        "                                let _ = window.show();\n"
        "                                let _ = window.set_focus();\n"
        "                            }\n"
        "                        }\n"
        "                        \"hide\" => {\n"
        "                            if let Some(window) = app.get_webview_window(\"main\") {\n"
        "                                let _ = window.hide();\n"
        "                            }\n"
        "                        }\n"
        "                        \"about\" => {\n"
        "                            let _ = open(\"https://github.com/MMXC/vibex\", None::<&str>);\n"
        "                        }\n"
        "                        _ => {}\n"
        "                    }\n"
        "                })\n"
        "                .on_tray_icon_event(|tray, event| {\n"
        "                    if let TrayIconEvent::Click {\n"
        "                        button: MouseButton::Left,\n"
        "                        button_state: MouseButtonState::Up,\n"
        "                        ..\n"
        "                    } = event\n"
        "                    {\n"
        "                        let app = tray.app_handle();\n"
        "                        if let Some(window) = app.get_webview_window(\"main\") {\n"
        "                            let _ = window.show();\n"
        "                            let _ = window.set_focus();\n"
        "                        }\n"
        "                    }\n"
        "                })\n"
        "                .build(app)?;\n\n"
        "            log::info!(\"Application setup complete\");\n"
        "            Ok(())\n"
        "        })\n"
        "        .on_window_event(|window, event| {\n"
        "            match event {\n"
        "                WindowEvent::CloseRequested { api, .. } => {\n"
        "                    #[cfg(target_os = \"macos\")]\n"
        "                    {\n"
        "                        let _ = window.hide();\n"
        "                        api.prevent_close();\n"
        "                    }\n"
        "                }\n"
        "                _ => {}\n"
        "            }\n"
        "        });\n\n"
        "    builder\n"
        "        .run(tauri::generate_context!())\n"
        "        .expect(\"error while running tauri application\");\n"
        "}\n"
    )
    return header + body


def gen_commands_rs() -> str:
    header = (
        "// ============================================================\n"
        "// This file is auto-generated by spec-to-tauri\n"
        f"// from: {SPEC_DIR}\n"
        f"// date: {DATE}\n"
        "// DO NOT edit directly\n"
        "// ============================================================\n\n"
    )
    body = (
        "use serde::{Deserialize, Serialize};\n"
        "use std::fs;\n"
        "use tauri::AppHandle;\n\n"
        "#[derive(Debug, Serialize, Deserialize)]\n"
        "pub struct CanvasSnapshot {\n"
        "    pub id: String,\n"
        "    pub canvas_id: String,\n"
        "    pub data: String,\n"
        "    pub created_at: String,\n"
        "    pub is_auto: bool,\n"
        "}\n\n"
        "#[derive(Debug, Serialize, Deserialize)]\n"
        "pub struct ExportResult {\n"
        "    pub path: String,\n"
        "    pub format: String,\n"
        "}\n\n"
        "/// Save a canvas snapshot to disk\n"
        "#[tauri::command]\n"
        "pub async fn save_canvas_snapshot(\n"
        "    app: AppHandle,\n"
        "    snapshot: CanvasSnapshot,\n"
        ") -> Result<String, String> {\n"
        "    let app_data = app\n"
        "        .path()\n"
        "        .app_data_dir()\n"
        "        .map_err(|e| format!(\"Failed to get app data dir: {}\", e))?;\n\n"
        "    let snapshots_dir = app_data.join(\"snapshots\").join(&snapshot.canvas_id);\n"
        "    fs::create_dir_all(&snapshots_dir)\n"
        "        .map_err(|e| format!(\"Failed to create snapshots dir: {}\", e))?;\n\n"
        "    let file_path = snapshots_dir.join(format!(\"{}.json\", snapshot.id));\n"
        "    let json = serde_json::to_string_pretty(&snapshot)\n"
        "        .map_err(|e| format!(\"Failed to serialize snapshot: {}\", e))?;\n\n"
        "    fs::write(&file_path, json)\n"
        "        .map_err(|e| format!(\"Failed to write snapshot: {}\", e))?;\n\n"
        "    log::info!(\"Saved snapshot: {:?}\", file_path);\n"
        "    Ok(file_path.to_string_lossy().to_string())\n"
        "}\n\n"
        "/// Load a canvas snapshot from disk\n"
        "#[tauri::command]\n"
        "pub async fn load_canvas_snapshot(\n"
        "    app: AppHandle,\n"
        "    snapshot_id: String,\n"
        "    canvas_id: String,\n"
        ") -> Result<CanvasSnapshot, String> {\n"
        "    let app_data = app\n"
        "        .path()\n"
        "        .app_data_dir()\n"
        "        .map_err(|e| format!(\"Failed to get app data dir: {}\", e))?;\n\n"
        "    let file_path = app_data\n"
        "        .join(\"snapshots\")\n"
        "        .join(&canvas_id)\n"
        "        .join(format!(\"{}.json\", snapshot_id));\n\n"
        "    let json = fs::read_to_string(&file_path)\n"
        "        .map_err(|e| format!(\"Failed to read snapshot: {}\", e))?;\n\n"
        "    serde_json::from_str(&json)\n"
        "        .map_err(|e| format!(\"Failed to parse snapshot: {}\", e))\n"
        "}\n\n"
        "/// Export canvas as image or JSON\n"
        "#[tauri::command]\n"
        "pub async fn export_canvas(\n"
        "    app: AppHandle,\n"
        "    canvas_id: String,\n"
        "    format: String,\n"
        "    content: String,\n"
        ") -> Result<ExportResult, String> {\n"
        "    let downloads = app\n"
        "        .path()\n"
        "        .download_dir()\n"
        "        .map_err(|e| format!(\"Failed to get downloads dir: {}\", e))?;\n\n"
        "    let filename = match format.as_str() {\n"
        "        \"png\" | \"jpg\" | \"jpeg\" => format!(\"{}.{}\", canvas_id, format),\n"
        "        \"json\" => format!(\"{}.json\", canvas_id),\n"
        "        \"svg\" => format!(\"{}.svg\", canvas_id),\n"
        "        _ => return Err(format!(\"Unsupported format: {}\", format)),\n"
        "    };\n\n"
        "    let file_path = downloads.join(&filename);\n\n"
        "    if format == \"json\" || format == \"svg\" {\n"
        "        fs::write(&file_path, &content)\n"
        "            .map_err(|e| format!(\"Failed to write export: {}\", e))?;\n"
        "    } else {\n"
        "        // Binary: content is base64 encoded\n"
        "        use base64::Engine;\n"
        "        use std::io::Write;\n"
        "        let decoded = base64::engine::general_purpose::STANDARD\n"
        "            .decode(&content)\n"
        "            .map_err(|_| \"Failed to decode base64 content\")?;\n"
        "        let mut file = fs::File::create(&file_path)\n"
        "            .map_err(|e| format!(\"Failed to create file: {}\", e))?;\n"
        "        file.write_all(&decoded)\n"
        "            .map_err(|e| format!(\"Failed to write binary: {}\", e))?;\n"
        "    }\n\n"
        "    log::info!(\"Exported canvas to: {:?}\", file_path);\n"
        "    Ok(ExportResult {\n"
        "        path: file_path.to_string_lossy().to_string(),\n"
        "        format,\n"
        "    })\n"
        "}\n\n"
        "/// Get the app data directory path\n"
        "#[tauri::command]\n"
        "pub fn get_app_data_dir(app: AppHandle) -> Result<String, String> {\n"
        "    app.path()\n"
        "        .app_data_dir()\n"
        "        .map(|p| p.to_string_lossy().to_string())\n"
        "        .map_err(|e| format!(\"Failed to get app data dir: {}\", e))\n"
        "}\n"
    )
    return header + body


def gen_build_rs() -> str:
    return "fn main() {\n    tauri_build::build()\n}\n"


def gen_icon_script() -> str:
    return (
        "#!/usr/bin/env python3\n"
        "# Icon generator - generates icons from SVG\n"
        "# Dependencies: pip install cairosvg Pillow\n\n"
        "from pathlib import Path\n\n"
        "ICONS_DIR = Path(__file__).parent / 'icons'\n"
        "ICONS_DIR.mkdir(exist_ok=True)\n\n"
        "SVG = '''\n"
        "<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 512 512'>\n"
        "  <rect width='512' height='512' rx='96' fill='#6366f1'/>\n"
        "  <text x='256' y='340' text-anchor='middle' font-family='system-ui' font-size='280' font-weight='700' fill='white'>V</text>\n"
        "</svg>\n"
        "'''\n\n"
        "def generate_icons():\n"
        "    try:\n"
        "        import cairosvg\n"
        "        svg_file = ICONS_DIR / 'icon.svg'\n"
        "        svg_file.write_text(SVG)\n"
        "        sizes = [32, 128, 256, 512]\n"
        "        for size in sizes:\n"
        "            out = ICONS_DIR / f'{size}x{size}.png'\n"
        "            cairosvg.svg2png(url=str(svg_file), write_to=str(out), output_width=size, output_height=size)\n"
        "            print(f'  Generated {size}x{size}.png')\n"
        "        out2x = ICONS_DIR / '128x128@2x.png'\n"
        "        cairosvg.svg2png(url=str(svg_file), write_to=str(out2x), output_width=256, output_height=256)\n"
        "        print('  Generated 128x128@2x.png')\n"
        "    except ImportError:\n"
        "        print('cairosvg not installed - skipping icon generation')\n\n"
        "if __name__ == '__main__':\n"
        "    generate_icons()\n"
    )


def gen_readme() -> str:
    return (
        "# VibeX Desktop\n\n"
        "Tauri 2.x desktop application, generated from canvas_tauri.yaml.\n\n"
        "## Generate\n\n"
        "```bash\n"
        "python3 generators/spec-to-tauri/gen.py specs/ desktop/\n"
        "```\n\n"
        "## Develop\n\n"
        "```bash\n"
        "cd src-tauri\n"
        "cargo tauri dev\n"
        "```\n\n"
        "## Build\n\n"
        "```bash\n"
        "cargo tauri build\n"
        "```\n\n"
        "## Permissions\n\n"
        "Permissions are declared in capabilities/default.json.\n"
        "To add new permissions (e.g. filesystem access), edit that file.\n\n"
        "## Commands\n\n"
        "Rust commands are called from frontend via invoke(), defined in src/commands.rs.\n\n"
        "| Command | Args | Description |\n"
        "|---------|------|-------------|\n"
        "| save_canvas_snapshot | snapshot | Save canvas snapshot |\n"
        "| load_canvas_snapshot | snapshot_id, canvas_id | Load snapshot |\n"
        "| export_canvas | canvas_id, format, content | Export canvas |\n"
        "| get_app_data_dir | - | Get app data directory |\n"
    )


def main():
    app_name = "vibex"

    print(f"spec-to-tauri v0.1")
    print(f"  spec_dir : {SPEC_DIR}")
    print(f"  output_dir: {OUT_DIR}")

    src_tauri = OUT_DIR / "src-tauri"
    capabilities = src_tauri / "capabilities"
    icons = src_tauri / "icons"

    for d in [src_tauri, src_tauri / "src", capabilities, icons]:
        ensure_dir(d)

    print("  Generating Cargo.toml...")
    (src_tauri / "Cargo.toml").write_text(gen_cargo_toml(app_name), encoding="utf-8")

    print("  Generating tauri.conf.json...")
    conf = gen_tauri_conf(app_name, SPEC_DIR)
    (src_tauri / "tauri.conf.json").write_text(
        json.dumps(conf, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    print("  Generating capabilities/default.json...")
    ensure_dir(capabilities)
    caps = gen_capabilities()
    (capabilities / "default.json").write_text(
        json.dumps(caps, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    print("  Generating src-tauri/src/main.rs...")
    (src_tauri / "src" / "main.rs").write_text(gen_main_rs(), encoding="utf-8")

    print("  Generating src-tauri/src/lib.rs...")
    (src_tauri / "src" / "lib.rs").write_text(gen_lib_rs(), encoding="utf-8")

    print("  Generating src-tauri/src/commands.rs...")
    (src_tauri / "src" / "commands.rs").write_text(gen_commands_rs(), encoding="utf-8")

    print("  Generating src-tauri/build.rs...")
    (src_tauri / "build.rs").write_text(gen_build_rs(), encoding="utf-8")

    print("  Generating icons/generate.py...")
    (icons / "generate.py").write_text(gen_icon_script(), encoding="utf-8")

    print("  Generating README.md...")
    (OUT_DIR / "README.md").write_text(gen_readme(), encoding="utf-8")

    print(f"\n✅ Tauri project generated!")
    print(f"   cd {OUT_DIR}/src-tauri && cargo tauri dev")


if __name__ == "__main__":
    main()
