"""ComponentRegistry — MF component registration and listing."""
import json
import os

COMPONENTS_FILE = "components/index.json"


class ComponentRegistry:
    def __init__(self, workspace: str = "."):
        self.workspace = workspace
        self.path = os.path.join(workspace, COMPONENTS_FILE)
        os.makedirs(os.path.dirname(self.path), exist_ok=True)
        if not os.path.exists(self.path):
            with open(self.path, "w") as f:
                json.dump({"components": [], "version": "1.0"}, f)

    def load(self) -> dict:
        with open(self.path) as f:
            return json.load(f)

    def save(self, data: dict):
        with open(self.path, "w") as f:
            json.dump(data, f, indent=2)

    def register(self, component: str, mf_url: str, dc_key: str = None) -> dict:
        data = self.load()
        # Remove existing entry with same name
        data["components"] = [c for c in data["components"] if c["name"] != component]
        entry = {"name": component, "mf_url": mf_url}
        if dc_key:
            entry["dc_key"] = dc_key
        data["components"].append(entry)
        self.save(data)
        return entry

    def list(self) -> list:
        return self.load().get("components", [])

    def unregister(self, component: str) -> dict:
        data = self.load()
        before = len(data["components"])
        data["components"] = [c for c in data["components"] if c["name"] != component]
        removed = before > len(data["components"])
        if removed:
            self.save(data)
        return {"ok": removed, "component": component}
