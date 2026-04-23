package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type collectorConfig struct {
	Receivers  map[string]any `yaml:"receivers"`
	Processors map[string]any `yaml:"processors"`
	Exporters  map[string]any `yaml:"exporters"`
	Service    struct {
		Pipelines map[string]struct {
			Receivers  []string `yaml:"receivers"`
			Processors []string `yaml:"processors"`
			Exporters  []string `yaml:"exporters"`
		} `yaml:"pipelines"`
	} `yaml:"service"`
}

// Guard rails — enforced on every config push.
//
//  1. Must be valid YAML.
//  2. At least one pipeline.
//  3. Every pipeline must include the required processors (memory_limiter, batch)
//     to prevent OOMs and downstream bursts.
//  4. Banned exporters are rejected outright.
//  5. If an exporter declares an `endpoint:`, the host must be on the allowlist.
//     Override via env OPAMP_EXPORTER_HOSTS (comma-separated).
var (
	requiredProcessors = []string{"memory_limiter", "batch"}
	bannedExporters    = map[string]bool{"file": true}
	defaultAllowedHost = []string{"otel-sink", "localhost", "127.0.0.1"}
)

func allowedHosts() map[string]bool {
	m := map[string]bool{}
	for _, h := range defaultAllowedHost {
		m[h] = true
	}
	if extra := os.Getenv("OPAMP_EXPORTER_HOSTS"); extra != "" {
		for _, h := range strings.Split(extra, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				m[h] = true
			}
		}
	}
	return m
}

func validateConfig(raw []byte) error {
	if len(raw) == 0 {
		return errors.New("empty config")
	}
	var cfg collectorConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if len(cfg.Service.Pipelines) == 0 {
		return errors.New("no pipelines defined under service.pipelines")
	}

	for name, p := range cfg.Service.Pipelines {
		if len(p.Receivers) == 0 {
			return fmt.Errorf("pipeline %q has no receivers", name)
		}
		if len(p.Exporters) == 0 {
			return fmt.Errorf("pipeline %q has no exporters", name)
		}
		for _, req := range requiredProcessors {
			if !hasProcessor(p.Processors, req) {
				return fmt.Errorf("pipeline %q missing required processor %q (guard rail)", name, req)
			}
		}
		for _, e := range p.Exporters {
			base := strings.SplitN(e, "/", 2)[0]
			if bannedExporters[base] {
				return fmt.Errorf("pipeline %q uses banned exporter %q (guard rail)", name, e)
			}
		}
	}

	hosts := allowedHosts()
	for name, raw := range cfg.Exporters {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ep, _ := m["endpoint"].(string)
		if ep == "" {
			continue
		}
		host := extractHost(ep)
		if host == "" {
			continue
		}
		if !hosts[host] {
			return fmt.Errorf("exporter %q endpoint host %q not allowlisted (guard rail); allowed=%v", name, host, keysOf(hosts))
		}
	}
	return nil
}

func hasProcessor(list []string, required string) bool {
	for _, p := range list {
		base := strings.SplitN(p, "/", 2)[0]
		if base == required {
			return true
		}
	}
	return false
}

func extractHost(endpoint string) string {
	if strings.Contains(endpoint, "://") {
		if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
			return u.Hostname()
		}
	}
	if h, _, ok := strings.Cut(endpoint, ":"); ok {
		return h
	}
	return endpoint
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
