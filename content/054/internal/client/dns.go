package client

import (
	"bufio"
	"bytes"
	"fmt"
	"net/netip"
	"os"
	"sort"
	"strings"
)

// MagicDNS: we don't ship a DNS server. Instead we rewrite /etc/hosts in place,
// bracketing our entries with a sentinel so we can update or remove them
// idempotently. Simple, works on any Linux distro, no systemd-resolved tricks.

const (
	hostsPath    = "/etc/hosts"
	hostsBegin   = "# BEGIN trynet -- managed block, do not edit"
	hostsEnd     = "# END trynet"
)

// ApplyHosts rewrites /etc/hosts with the supplied name→IP map.
func ApplyHosts(hosts map[string]netip.Addr) error {
	existing, err := os.ReadFile(hostsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	stripped := stripManagedBlock(existing)

	// Sort for deterministic output.
	names := make([]string, 0, len(hosts))
	for n := range hosts {
		names = append(names, n)
	}
	sort.Strings(names)

	var block bytes.Buffer
	block.WriteString(hostsBegin)
	block.WriteByte('\n')
	for _, name := range names {
		fmt.Fprintf(&block, "%s\t%s\n", hosts[name].String(), name)
	}
	block.WriteString(hostsEnd)
	block.WriteByte('\n')

	final := append(bytes.TrimRight(stripped, "\n"), '\n')
	final = append(final, block.Bytes()...)
	return writeAtomic(hostsPath, final, 0o644)
}

// RemoveHosts pulls the managed block out of /etc/hosts. Called on logout.
func RemoveHosts() error {
	existing, err := os.ReadFile(hostsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return writeAtomic(hostsPath, stripManagedBlock(existing), 0o644)
}

func stripManagedBlock(b []byte) []byte {
	var out bytes.Buffer
	skipping := false
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, hostsBegin) {
			skipping = true
			continue
		}
		if skipping {
			if strings.Contains(line, hostsEnd) {
				skipping = false
			}
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func writeAtomic(path string, b []byte, mode os.FileMode) error {
	tmp := path + ".tmp.trynet"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
