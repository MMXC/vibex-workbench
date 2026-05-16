package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// SpecPilot CLI base dir — set via env or default
const specpilotBaseDir = "/tmp/specpilot"

// runSP runs a specpilot CLI command and returns the output
func runSP(args ...string) string {
	cmd := exec.Command("python3", append([]string{"-m", "cli"}, args...)...)
	cmd.Dir = specpilotBaseDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf(`{"error": "specpilot CLI failed: %v", "stderr": "%v"}`, err, err)
	}
	return strings.TrimSpace(string(out))
}

// dcListHandler — List all DataCenter keys and values
func dcListHandler(arguments string) string {
	return runSP("dc", "list")
}

// dcGetHandler — Get a single DataCenter key
func dcGetHandler(arguments string) string {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key required"}`
	}
	return runSP("dc", "get", args.Key)
}

// dcSetHandler — Set a DataCenter key/value
func dcSetHandler(arguments string) string {
	var args struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key and value required"}`
	}
	return runSP("dc", "set", args.Key, fmt.Sprintf("%v", args.Value))
}

// ecHistoryHandler — Get EventCenter history
func ecHistoryHandler(arguments string) string {
	return runSP("ec", "history")
}

// ecEmitHandler — Emit an event
func ecEmitHandler(arguments string) string {
	var args struct {
		Event   string `json:"event"`
		Payload any    `json:"payload"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and payload required"}`
	}
	payloadJSON, _ := json.Marshal(args.Payload)
	return runSP("ec", "emit", args.Event, string(payloadJSON))
}

// ecSubscribeHandler — Subscribe a component to an event
func ecSubscribeHandler(arguments string) string {
	var args struct {
		Event      string `json:"event"`
		Subscriber string `json:"subscriber"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and subscriber required"}`
	}
	return runSP("ec", "subscribe", args.Event, args.Subscriber)
}

// adListHandler — List all adapters
func adListHandler(arguments string) string {
	return runSP("ad", "list")
}

// adSwitchHandler — Switch active adapter
func adSwitchHandler(arguments string) string {
	var args struct {
		Adapter string `json:"adapter"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: adapter name required"}`
	}
	return runSP("ad", "switch", args.Adapter)
}

// adQueryHandler — Query via current adapter
func adQueryHandler(arguments string) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: query required"}`
	}
	return runSP("ad", "query", args.Query)
}

// specListHandler — List all specs
func specListHandler(arguments string) string {
	return runSP("spec", "list")
}

// specGetHandler — Get a single spec
func specGetHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("spec", "get", args.Name)
}

// specBindingHandler — Check field bindings for a spec
func specBindingHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("spec", "binding", args.Name)
}

// mfListHandler — List registered MF components
func mfListHandler(arguments string) string {
	return runSP("mf", "list")
}

// mfRegisterHandler — Register a MF component
func mfRegisterHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: name and path required"}`
	}
	return runSP("mf", "register", args.Name, args.Path)
}

// mfResolveHandler — Resolve components from a spec
func mfResolveHandler(arguments string) string {
	var args struct {
		Spec string `json:"spec"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("mf", "resolve-from-spec", args.Spec)
}
