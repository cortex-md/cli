package plugin

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cortex/cli/internal/rules"
)

//go:embed inspect.cjs
var inspectScript []byte

var ErrNoJavaScriptRuntime = errors.New("node or bun is required to inspect the bundle")

type BundleManifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Permissions []string `json:"permissions"`
}

type Inspection struct {
	Manifest                 *BundleManifest `json:"manifest"`
	HasMarkdownContributions bool            `json:"hasMarkdownContributions"`
	Error                    string          `json:"error"`
}

func InspectBundle(bundlePath string) (*Inspection, error) {
	runtime := javaScriptRuntime()
	if runtime == "" {
		return nil, ErrNoJavaScriptRuntime
	}

	scriptDir, err := os.MkdirTemp("", "cortex-inspect-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scriptDir)

	scriptPath := filepath.Join(scriptDir, "inspect.cjs")
	if err := os.WriteFile(scriptPath, inspectScript, 0o600); err != nil {
		return nil, err
	}

	permissions, err := json.Marshal(rules.PluginPermissions)
	if err != nil {
		return nil, err
	}

	output, err := exec.Command(runtime, scriptPath, bundlePath, string(permissions)).Output()
	if err != nil {
		return nil, fmt.Errorf("bundle inspection failed: %w", err)
	}

	var inspection Inspection
	if err := json.Unmarshal(output, &inspection); err != nil {
		return nil, fmt.Errorf("bundle inspection returned invalid output: %w", err)
	}
	return &inspection, nil
}

func javaScriptRuntime() string {
	for _, candidate := range []string{"node", "bun"} {
		if execExists(candidate) {
			return candidate
		}
	}
	return ""
}
