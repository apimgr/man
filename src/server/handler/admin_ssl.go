// Admin SSL/TLS form: lists configured DNS providers, lets the operator save
// or replace credentials per provider, and triggers an immediate provision
// when requested. CSRF is handled by the global middleware. See AI.md PART 15.

package handler

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/casapps/casman/src/ssl"
)

// SSLBackend is the subset of the SSL subsystem the handler needs. Provided
// by src/server/server.go after both the vault and provisioner are built.
type SSLBackend interface {
	Vault() *ssl.Vault
}

var sslBackend SSLBackend

// SetSSLBackend wires the vault into the handler package. Called once from
// src/server/server.go during initialization.
func SetSSLBackend(b SSLBackend) {
	sslBackend = b
}

// AdminSSLForm renders the SSL/TLS admin form (GET).
func (h *Handlers) AdminSSLForm(w http.ResponseWriter, r *http.Request) {
	csrf := getCSRFToken(r)

	configured := map[string]bool{}
	if sslBackend != nil && sslBackend.Vault() != nil {
		if names, err := sslBackend.Vault().List(); err == nil {
			for _, n := range names {
				configured[n] = true
			}
		}
	}

	var b strings.Builder
	b.WriteString(`<p>Configure DNS provider credentials for ACME DNS-01 challenges. Saved values are encrypted with the server master key.</p>`)
	b.WriteString(`<form method="POST" action="/admin/server/ssl" autocomplete="off">`)
	fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s">`, html.EscapeString(csrf))

	b.WriteString(`<p><label>Provider <select name="provider" onchange="toggleFields(this.value)">`)
	for _, p := range ssl.Providers() {
		marker := ""
		if configured[p.ID] {
			marker = " (configured)"
		}
		fmt.Fprintf(&b, `<option value="%s">%s%s</option>`, html.EscapeString(p.ID), html.EscapeString(p.Name), marker)
	}
	b.WriteString(`</select></label></p>`)

	for _, p := range ssl.Providers() {
		fmt.Fprintf(&b, `<fieldset id="fields-%s" style="display:none">`, html.EscapeString(p.ID))
		fmt.Fprintf(&b, `<legend>%s credentials</legend>`, html.EscapeString(p.Name))
		if len(p.Fields) == 0 {
			b.WriteString(`<p>No credentials needed (you publish TXT records by hand).</p>`)
		}
		for _, f := range p.Fields {
			inputType := "text"
			if f.Secret {
				inputType = "password"
			}
			required := ""
			if f.Required {
				required = " required"
			}
			fmt.Fprintf(&b,
				`<p><label>%s <input type="%s" name="field_%s" autocomplete="new-password"%s></label></p>`,
				html.EscapeString(f.Label),
				inputType,
				html.EscapeString(f.Key),
				required,
			)
		}
		b.WriteString(`</fieldset>`)
	}

	b.WriteString(`<p><button type="submit" name="action" value="save">Save credentials</button> `)
	b.WriteString(`<button type="submit" name="action" value="delete" formnovalidate>Delete credentials</button></p>`)
	b.WriteString(`</form>`)

	b.WriteString(`<script>
function toggleFields(id){
  document.querySelectorAll('fieldset[id^="fields-"]').forEach(f=>f.style.display='none');
  var t=document.getElementById('fields-'+id);
  if(t){t.style.display='';}
}
toggleFields(document.querySelector('select[name=provider]').value);
</script>`)

	h.adminPage(w, "SSL/TLS", b.String())
}

// AdminSSLSave processes the SSL/TLS admin form (POST).
func (h *Handlers) AdminSSLSave(w http.ResponseWriter, r *http.Request) {
	if sslBackend == nil || sslBackend.Vault() == nil {
		http.Error(w, "SSL backend not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	provider := r.FormValue("provider")
	action := r.FormValue("action")
	vault := sslBackend.Vault()

	switch action {
	case "delete":
		if err := vault.Delete(provider); err != nil {
			http.Error(w, fmt.Sprintf("delete: %v", err), http.StatusInternalServerError)
			return
		}
	case "save":
		fields := map[string]string{}
		for k, vs := range r.PostForm {
			if !strings.HasPrefix(k, "field_") || len(vs) == 0 {
				continue
			}
			fields[strings.TrimPrefix(k, "field_")] = vs[0]
		}
		clean, err := ssl.ValidateFields(provider, fields)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := vault.Save(provider, clean); err != nil {
			http.Error(w, fmt.Sprintf("save: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/server/ssl", http.StatusSeeOther)
}
