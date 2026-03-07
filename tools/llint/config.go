package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	globalConfigPath = "/etc/lunar/config"
	localConfigPath  = "/etc/lunar/local/config"
)

// Config holds lunar configuration values needed by llint.
type Config struct {
	Moonbase    string
	ModuleIndex string
}

// LoadConfig reads the global config then overlays the local config.
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(globalConfigPath, localConfigPath)
}

// LoadConfigFrom reads config from the given paths (for testing).
func LoadConfigFrom(globalPath, localPath string) (*Config, error) {
	values := make(map[string]string)

	if err := parseConfigFile(globalPath, values); err != nil {
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	// Local config is optional
	if _, err := os.Stat(localPath); err == nil {
		if err := parseConfigFile(localPath, values); err != nil {
			return nil, fmt.Errorf("reading local config: %w", err)
		}
	}

	cfg := &Config{
		Moonbase:    values["MOONBASE"],
		ModuleIndex: values["MODULE_INDEX"],
	}

	if cfg.Moonbase == "" {
		return nil, fmt.Errorf("MOONBASE not defined in config")
	}

	return cfg, nil
}

// parseConfigFile reads a bash-style KEY=VALUE config file into the map.
// It handles leading whitespace, comments, export prefix, and quoted values.
// It does NOT expand ${VAR:-default} patterns — it takes the literal value.
func parseConfigFile(path string, values map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export" prefix
		line = strings.TrimPrefix(line, "export")
		line = strings.TrimSpace(line)

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := line[idx+1:]

		// Strip surrounding quotes
		val = strings.TrimSpace(val)
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		// Skip array assignments (e.g., MOONBASE_URL[0]=...)
		if strings.Contains(key, "[") {
			continue
		}

		values[key] = val
	}

	return scanner.Err()
}
