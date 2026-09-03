// Package notify renders and dispatches server notification emails per
// AI.md PART 18. The package owns nothing of its own: it's a thin layer
// that takes an *smtp.Client + a recipient resolver and wraps the
// "no SMTP = no emails, do not even try" guarantee from the spec.
package notify

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
	"time"
)

// SMTPClient is the subset of *smtp.Client the notifier needs. Defined as
// an interface so tests can stub it without pulling in the SMTP package.
type SMTPClient interface {
	IsAvailable() bool
	Send(to, subject, body string) error
}

// RecipientFunc returns the list of admin email recipients at the time the
// notification is about to be sent. It is invoked once per notification so
// admin email changes take effect on the next event without restart.
type RecipientFunc func() []string

// Notifier renders templated emails and fans them out to the admin
// recipient list. All methods are safe to call when SMTP is not configured;
// they short-circuit silently per spec.
type Notifier struct {
	client     SMTPClient
	recipients RecipientFunc
	appName    string
}

// New builds a Notifier. appName is substituted as {app_name} in subjects.
func New(client SMTPClient, appName string, recipients RecipientFunc) *Notifier {
	if appName == "" {
		appName = "casman"
	}
	return &Notifier{client: client, recipients: recipients, appName: appName}
}

// BackupComplete dispatches the backup_complete template per PART 18.
func (n *Notifier) BackupComplete(filename string, sizeBytes int64) {
	n.dispatch("backup_complete",
		fmt.Sprintf("Backup Complete - %s", n.appName),
		backupCompleteTpl,
		map[string]any{
			"AppName":  n.appName,
			"Filename": filename,
			"Size":     formatBytes(sizeBytes),
			"Time":     time.Now().UTC().Format(time.RFC3339),
		})
}

// BackupFailed dispatches the backup_failed template per PART 18.
func (n *Notifier) BackupFailed(reason string) {
	n.dispatch("backup_failed",
		fmt.Sprintf("Backup Failed - %s", n.appName),
		backupFailedTpl,
		map[string]any{
			"AppName": n.appName,
			"Reason":  reason,
			"Time":    time.Now().UTC().Format(time.RFC3339),
		})
}

// SSLExpiring dispatches the ssl_expiring template per PART 18. daysLeft is
// the integer number of days until expiry; the spec recommends sending at
// 30/14/7/3/1 day thresholds — that scheduling decision belongs to the
// caller, this method just renders the message.
func (n *Notifier) SSLExpiring(domain string, daysLeft int) {
	n.dispatch("ssl_expiring",
		fmt.Sprintf("SSL Certificate Expiring - %s", n.appName),
		sslExpiringTpl,
		map[string]any{
			"AppName":  n.appName,
			"Domain":   domain,
			"DaysLeft": daysLeft,
			"Time":     time.Now().UTC().Format(time.RFC3339),
		})
}

// SSLRenewed dispatches the ssl_renewed template per PART 18.
func (n *Notifier) SSLRenewed(domain string) {
	n.dispatch("ssl_renewed",
		fmt.Sprintf("SSL Certificate Renewed - %s", n.appName),
		sslRenewedTpl,
		map[string]any{
			"AppName": n.appName,
			"Domain":  domain,
			"Time":    time.Now().UTC().Format(time.RFC3339),
		})
}

// SchedulerError dispatches the scheduler_error template per PART 18.
func (n *Notifier) SchedulerError(taskName, reason string) {
	n.dispatch("scheduler_error",
		fmt.Sprintf("Scheduled Task Failed - %s", n.appName),
		schedulerErrorTpl,
		map[string]any{
			"AppName":  n.appName,
			"TaskName": taskName,
			"Reason":   reason,
			"Time":     time.Now().UTC().Format(time.RFC3339),
		})
}

// dispatch renders the template, then fans out to every admin recipient.
// Per AI.md PART 18: "No SMTP = No emails. Don't even try." We honour that
// by short-circuiting before template rendering.
func (n *Notifier) dispatch(name, subject, tpl string, data any) {
	if n == nil || n.client == nil || !n.client.IsAvailable() {
		return
	}
	if n.recipients == nil {
		return
	}
	recipients := n.recipients()
	if len(recipients) == 0 {
		return
	}

	body, err := render(tpl, data)
	if err != nil {
		log.Printf("notify: %s render: %v", name, err)
		return
	}
	for _, to := range recipients {
		if err := n.client.Send(to, subject, body); err != nil {
			log.Printf("notify: %s send to %s: %v", name, to, err)
		}
	}
}

func render(tpl string, data any) (string, error) {
	t, err := template.New("notify").Parse(tpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func formatBytes(n int64) string {
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/(k*k*k))
	}
}

const backupCompleteTpl = `Backup completed successfully for {{.AppName}}.

File: {{.Filename}}
Size: {{.Size}}
Time: {{.Time}}

This is an automated message from {{.AppName}}.
`

const backupFailedTpl = `Backup FAILED for {{.AppName}}.

Reason: {{.Reason}}
Time: {{.Time}}

Please check the server logs and recent retention activity.
This is an automated message from {{.AppName}}.
`

const sslExpiringTpl = `SSL certificate for {{.Domain}} on {{.AppName}} is expiring in {{.DaysLeft}} day(s).

If automatic renewal is enabled, no action is required — this is a
heads-up. If renewal fails or is disabled, plan a manual rotation.

Time: {{.Time}}
This is an automated message from {{.AppName}}.
`

const sslRenewedTpl = `SSL certificate for {{.Domain}} on {{.AppName}} was renewed successfully.

Time: {{.Time}}
This is an automated message from {{.AppName}}.
`

const schedulerErrorTpl = `Scheduled task "{{.TaskName}}" failed on {{.AppName}}.

Reason: {{.Reason}}
Time: {{.Time}}

Check the scheduler logs and recent runs in the admin panel.
This is an automated message from {{.AppName}}.
`
