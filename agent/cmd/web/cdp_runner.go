package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type cdpValidateRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	PlanID        string `json:"plan_id"`
	TargetEnv     struct {
		Deployment string `json:"deployment"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		TimeoutSec int    `json:"timeout_sec"`
		SessionID  string `json:"session_id"`
	} `json:"target_env"`
	EntryURL string `json:"entry_url"`
	Steps    []struct {
		ID         string           `json:"id"`
		URL        string           `json:"url"`
		Actions    []map[string]any `json:"actions"`
		Assertions []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Selector string `json:"selector"`
			Value    string `json:"value"`
		} `json:"assertions"`
		TimeoutSec int `json:"timeout_sec"`
	} `json:"steps"`
	ScreenshotOnFail bool `json:"screenshot_on_fail"`
}

type cdpValidateResponse struct {
	OK          bool     `json:"ok"`
	PlanID      string   `json:"plan_id"`
	Logs        []string `json:"logs,omitempty"`
	Screenshots []string `json:"screenshots,omitempty"`
	Error       string   `json:"error,omitempty"`
}

func cdpValidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cdpValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.PlanID) == "" || len(req.Steps) == 0 {
		http.Error(w, "plan_id and steps are required", http.StatusBadRequest)
		return
	}
	wsRoot := strings.TrimSpace(req.WorkspaceRoot)
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}
	resp := runCDPValidation(wsRoot, req)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func runCDPValidation(workspaceRoot string, req cdpValidateRequest) cdpValidateResponse {
	host := strings.TrimSpace(req.TargetEnv.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := req.TargetEnv.Port
	if port <= 0 {
		port = 9222
	}
	timeoutSec := req.TargetEnv.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	out := cdpValidateResponse{OK: false, PlanID: req.PlanID, Logs: []string{}}
	wsURL, err := resolveCDPWebSocketURL(host, port)
	if err != nil {
		out.Error = fmt.Sprintf("resolve cdp websocket failed: %v", err)
		return out
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, wsURL)
	defer allocCancel()
	tabCtx, tabCancel := chromedp.NewContext(allocCtx)
	defer tabCancel()

	ok := true
	failErr := ""
	if strings.TrimSpace(req.EntryURL) != "" {
		if err := chromedp.Run(tabCtx, chromedp.Navigate(req.EntryURL)); err != nil {
			ok = false
			failErr = fmt.Sprintf("entry_url navigate failed: %v", err)
		} else {
			out.Logs = append(out.Logs, "entry_url navigated")
		}
	}

	for _, step := range req.Steps {
		if !ok {
			break
		}
		stepID := step.ID
		if stepID == "" {
			stepID = "step"
		}
		if strings.TrimSpace(step.URL) != "" {
			if err := chromedp.Run(tabCtx, chromedp.Navigate(step.URL)); err != nil {
				ok = false
				failErr = fmt.Sprintf("%s navigate failed: %v", stepID, err)
				break
			}
			out.Logs = append(out.Logs, fmt.Sprintf("%s navigated %s", stepID, step.URL))
		}
		for _, action := range step.Actions {
			typ := strings.ToLower(strings.TrimSpace(anyToString(action["type"])))
			switch typ {
			case "click":
				sel := strings.TrimSpace(anyToString(action["selector"]))
				if sel == "" {
					continue
				}
				if err := chromedp.Run(tabCtx, chromedp.Click(sel, chromedp.ByQuery)); err != nil {
					ok = false
					failErr = fmt.Sprintf("%s click failed (%s): %v", stepID, sel, err)
				}
			case "type":
				sel := strings.TrimSpace(anyToString(action["selector"]))
				val := anyToString(action["value"])
				if sel == "" {
					continue
				}
				if err := chromedp.Run(tabCtx, chromedp.SetValue(sel, val, chromedp.ByQuery)); err != nil {
					ok = false
					failErr = fmt.Sprintf("%s type failed (%s): %v", stepID, sel, err)
				}
			case "wait_ms":
				ms := int(anyToFloat(action["value"]))
				if ms > 0 {
					time.Sleep(time.Duration(ms) * time.Millisecond)
				}
			}
			if !ok {
				break
			}
		}
		for _, as := range step.Assertions {
			if !ok {
				break
			}
			switch strings.ToLower(strings.TrimSpace(as.Type)) {
			case "text_contains":
				sel := strings.TrimSpace(as.Selector)
				if sel == "" {
					sel = "body"
				}
				var txt string
				if err := chromedp.Run(tabCtx, chromedp.Text(sel, &txt, chromedp.ByQuery, chromedp.NodeVisible)); err != nil {
					ok = false
					failErr = fmt.Sprintf("%s text_contains read failed (%s): %v", stepID, sel, err)
					break
				}
				if !strings.Contains(txt, as.Value) {
					ok = false
					failErr = fmt.Sprintf("%s text_contains failed: selector=%s expect=%q", stepID, sel, as.Value)
				}
			case "selector_visible":
				sel := strings.TrimSpace(as.Selector)
				if sel == "" {
					ok = false
					failErr = fmt.Sprintf("%s selector_visible requires selector", stepID)
					break
				}
				if err := chromedp.Run(tabCtx, chromedp.WaitVisible(sel, chromedp.ByQuery)); err != nil {
					ok = false
					failErr = fmt.Sprintf("%s selector_visible failed (%s): %v", stepID, sel, err)
				}
			case "url_matches":
				var current string
				if err := chromedp.Run(tabCtx, chromedp.Location(&current)); err != nil {
					ok = false
					failErr = fmt.Sprintf("%s url read failed: %v", stepID, err)
					break
				}
				matched, err := regexp.MatchString(as.Value, current)
				if err != nil {
					ok = false
					failErr = fmt.Sprintf("%s bad url_matches regex: %v", stepID, err)
					break
				}
				if !matched {
					ok = false
					failErr = fmt.Sprintf("%s url_matches failed: url=%s regex=%s", stepID, current, as.Value)
				}
			}
			if ok {
				out.Logs = append(out.Logs, fmt.Sprintf("%s assertion %s passed", stepID, as.Type))
			}
		}
	}

	if !ok && req.ScreenshotOnFail {
		_ = os.MkdirAll(filepath.Join(workspaceRoot, ".vibex", "cdp-snapshots"), 0755)
		p := filepath.Join(workspaceRoot, ".vibex", "cdp-snapshots", fmt.Sprintf("%s-%d.png", sanitizeFilename(req.PlanID), time.Now().Unix()))
		var buf []byte
		if err := chromedp.Run(tabCtx, chromedp.CaptureScreenshot(&buf)); err == nil && len(buf) > 0 {
			if writeErr := os.WriteFile(p, buf, 0644); writeErr == nil {
				rel, _ := filepath.Rel(workspaceRoot, p)
				out.Screenshots = append(out.Screenshots, filepath.ToSlash(rel))
			}
		}
	}
	out.OK = ok
	if !ok {
		out.Error = failErr
	}
	return out
}

func resolveCDPWebSocketURL(host string, port int) (string, error) {
	url := fmt.Sprintf("http://%s:%d/json/version", host, port)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return "", fmt.Errorf("webSocketDebuggerUrl missing at %s", url)
	}
	return payload.WebSocketDebuggerURL, nil
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func anyToFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	if s == "" {
		return "cdp-plan"
	}
	return s
}
