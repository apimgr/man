// Tor admin surface — read-only: shows whether the bundled hidden service is
// running, the onion address, and the configured virtual port. Per AI.md
// PART 32 the hidden service is auto-enabled when a tor binary is present;
// there is no operator toggle, only status.

package handler

import (
	"fmt"
	"html"
	"net/http"

	"github.com/casapps/casman/src/server/model"
)

// TorBackend is the subset of the server the handler needs to render Tor
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

// torInfoFor builds the model.TorInfo block used in /healthz responses. It
// reads the same backend the admin page uses, so the two surfaces never
// drift apart.
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

// AdminTor handles /{admin_path}/server/network/tor.
// Replaces the placeholder previously in handler.go.
func (h *Handlers) AdminTor(w http.ResponseWriter, r *http.Request) {
	var status, addr, available string
	if torBackend == nil {
		status = "uninitialized"
	} else {
		if torBackend.TorAvailable() {
			available = "yes"
		} else {
			available = "no"
		}
		if torBackend.TorRunning() {
			status = "running"
		} else if torBackend.TorAvailable() {
			status = "stopped"
		} else {
			status = "unavailable (no tor binary on PATH)"
		}
		addr = torBackend.TorOnionAddress()
	}

	body := `<p>Per AI.md PART 32, the Tor hidden service is auto-enabled when the <code>tor</code> binary is reachable on PATH. There is no enable/disable toggle.</p>`
	body += `<table class="pages-table"><tbody>`
	body += fmt.Sprintf(`<tr><th>Tor binary</th><td>%s</td></tr>`, html.EscapeString(available))
	body += fmt.Sprintf(`<tr><th>Status</th><td>%s</td></tr>`, html.EscapeString(status))
	if addr != "" {
		body += fmt.Sprintf(`<tr><th>Onion address</th><td><code>%s</code></td></tr>`, html.EscapeString(addr))
	}
	body += `</tbody></table>`
	body += `<p class="note"><small>If the binary is installed inside the Docker image (per docker/Dockerfile) but the host doesn't ship one, the hidden service will be a silent no-op locally and active in production.</small></p>`

	h.adminPage(w, "Tor Hidden Service", body)
}
