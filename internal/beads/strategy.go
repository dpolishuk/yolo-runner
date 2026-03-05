package beads

type lifecycleStrategy interface {
	ready(rootID string) []string
	listTree(rootID string) []string
	show(id string) []string
	updateStatus(id, status string) []string
	updateNotes(id, notes string) []string
	close(id string) []string
	closeEligible() []string
	sync() []string
}

type commandStrategy struct {
	binary   string
	syncMode string
}

func strategyFromCapabilities(capabilities TrackerCapabilities) lifecycleStrategy {
	binary := capabilities.Backend
	if binary == "" {
		binary = backendBD
	}
	syncMode := capabilities.SyncMode
	if syncMode == "" {
		syncMode = syncModeActive
	}
	return commandStrategy{binary: binary, syncMode: syncMode}
}

func defaultBDStrategy() lifecycleStrategy {
	return commandStrategy{binary: backendBD, syncMode: syncModeActive}
}

func (s commandStrategy) ready(rootID string) []string {
	return []string{s.binary, "ready", "--parent", rootID, "--json"}
}

func (s commandStrategy) listTree(rootID string) []string {
	return []string{s.binary, "list", "--parent", rootID, "--json"}
}

func (s commandStrategy) show(id string) []string {
	return []string{s.binary, "show", id, "--json"}
}

func (s commandStrategy) updateStatus(id, status string) []string {
	return []string{s.binary, "update", id, "--status", status}
}

func (s commandStrategy) updateNotes(id, notes string) []string {
	return []string{s.binary, "update", id, "--notes", notes}
}

func (s commandStrategy) close(id string) []string {
	return []string{s.binary, "close", id}
}

func (s commandStrategy) closeEligible() []string {
	return []string{s.binary, "epic", "close-eligible"}
}

func (s commandStrategy) sync() []string {
	switch s.syncMode {
	case syncModeNoop:
		return nil
	case syncModeFlushOnly:
		return []string{s.binary, "sync", "--flush-only"}
	default:
		return []string{s.binary, "sync"}
	}
}
