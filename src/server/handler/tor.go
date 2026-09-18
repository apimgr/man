// Package handler — Tor status plumbing used by /healthz.
package handler

import "github.com/casapps/casman/src/server/model"

// TorBackend is the subset of the server the handler needs to report Tor
// status. Provided by src/server/server.go; nil when Tor isn't wired.
type TorBackend interface {
	TorAvailable() bool
	TorRunning() bool
	TorOnionAddress() string
}

var torBackend TorBackend

// SetTorBackend wires the Tor service into the handler package. Called once
// from src/server/server.go during initialization.
func SetTorBackend(b TorBackend) { torBackend = b }

// torInfoFor builds the model.TorInfo block used in /healthz responses.
func torInfoFor() model.TorInfo {
	if torBackend == nil {
		return model.TorInfo{Status: "disabled"}
	}
	enabled := torBackend.TorAvailable()
	running := torBackend.TorRunning()
	status := "disabled"
	switch {
	case running:
		status = "running"
	case enabled:
		status = "stopped"
	}
	return model.TorInfo{
		Enabled:  enabled,
		Running:  running,
		Status:   status,
		Hostname: torBackend.TorOnionAddress(),
	}
}
