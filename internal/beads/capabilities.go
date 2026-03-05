package beads

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	backendBD = "bd"
	backendBR = "br"

	syncModeActive    = "active"
	syncModeFlushOnly = "flush_only"
	syncModeNoop      = "noop"
)

var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

type TrackerCapabilities struct {
	Backend  string
	SyncMode string
	Version  string
}

func ProbeTrackerCapabilities(runner Runner) (TrackerCapabilities, error) {
	if runner == nil {
		return TrackerCapabilities{}, fmt.Errorf("capability probe failed: runner is nil")
	}

	if output, err := runner.Run(backendBD, "version"); err == nil {
		version := parseVersionToken(output)
		return TrackerCapabilities{
			Backend:  backendBD,
			SyncMode: classifyBDSyncMode(version),
			Version:  version,
		}, nil
	}

	if output, err := runner.Run(backendBR, "version"); err == nil {
		return TrackerCapabilities{
			Backend:  backendBR,
			SyncMode: syncModeFlushOnly,
			Version:  parseVersionToken(output),
		}, nil
	}

	return TrackerCapabilities{}, fmt.Errorf("capability probe failed: unable to detect backend via `bd version` or `br version`")
}

func parseVersionToken(raw string) string {
	match := versionPattern.FindString(raw)
	if match == "" {
		return ""
	}
	return match
}

func classifyBDSyncMode(version string) string {
	if version == "" {
		return syncModeActive
	}
	major, minor, _, ok := parseSemver(version)
	if !ok {
		return syncModeActive
	}
	if major == 0 && minor >= 56 {
		return syncModeNoop
	}
	if major > 0 {
		return syncModeNoop
	}
	return syncModeActive
}

func parseSemver(version string) (int, int, int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}
