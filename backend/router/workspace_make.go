package router

import (
	"encoding/json"
	"net/http"
	"os/exec"
)

// RegisterWorkspaceMake 注册 POST /api/workspace/run-make
func RegisterWorkspaceMake(mux *http.ServeMux, workspaceRoot string) {
    mux.HandleFunc("/api/workspace/run-make", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "POST only", http.StatusMethodNotAllowed)
            return
        }

        var body struct {
            Target    string `json:"target"`
            Workspace string `json:"workspace"`
        }
        json.NewDecoder(r.Body).Decode(&body)

        ws := workspaceRoot
        if body.Workspace != "" {
            ws = body.Workspace
        }

        target := body.Target
        if target == "" {
            target = "validate"
        }

        cmd := exec.Command("make", target)
        cmd.Dir = ws
        out, err := cmd.CombinedOutput()
        output := string(out)

        json.NewEncoder(w).Encode(map[string]interface{}{
            "ok":        err == nil,
            "target":    target,
            "output":    output,
            "workspace": ws,
        })
    })
}
