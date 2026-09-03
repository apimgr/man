package notify

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

type fakeClient struct {
	mu        sync.Mutex
	available bool
	sent      []sentMsg
	failNext  error
}

type sentMsg struct{ To, Subject, Body string }

func (f *fakeClient) IsAvailable() bool { return f.available }

func (f *fakeClient) Send(to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	f.sent = append(f.sent, sentMsg{To: to, Subject: subject, Body: body})
	return nil
}

func TestNotifier_NoSMTPNoSend(t *testing.T) {
	c := &fakeClient{available: false}
	n := New(c, "appy", func() []string { return []string{"a@example.com"} })
	n.BackupComplete("file.tar.gz", 1024)
	if len(c.sent) != 0 {
		t.Errorf("sent %d msgs, want 0 when SMTP unavailable", len(c.sent))
	}
}

func TestNotifier_NoRecipientsNoSend(t *testing.T) {
	c := &fakeClient{available: true}
	n := New(c, "appy", func() []string { return nil })
	n.BackupFailed("disk full")
	if len(c.sent) != 0 {
		t.Errorf("sent %d msgs, want 0 with empty recipients", len(c.sent))
	}
}

func TestNotifier_BackupComplete_FansOut(t *testing.T) {
	c := &fakeClient{available: true}
	recips := []string{"a@example.com", "b@example.com"}
	n := New(c, "casman", func() []string { return recips })
	n.BackupComplete("backup_2026.tar.gz.enc", 2*1024*1024)

	if len(c.sent) != 2 {
		t.Fatalf("sent %d msgs, want 2", len(c.sent))
	}
	for i, msg := range c.sent {
		if msg.To != recips[i] {
			t.Errorf("recipient[%d] = %q, want %q", i, msg.To, recips[i])
		}
		if !strings.Contains(msg.Subject, "Backup Complete") || !strings.Contains(msg.Subject, "casman") {
			t.Errorf("subject: %q", msg.Subject)
		}
		if !strings.Contains(msg.Body, "backup_2026.tar.gz.enc") {
			t.Errorf("body missing filename: %q", msg.Body)
		}
		if !strings.Contains(msg.Body, "2.0 MB") {
			t.Errorf("body missing humanized size: %q", msg.Body)
		}
	}
}

func TestNotifier_SSLExpiring_FormatsDays(t *testing.T) {
	c := &fakeClient{available: true}
	n := New(c, "casman", func() []string { return []string{"x@example.com"} })
	n.SSLExpiring("example.com", 7)
	if len(c.sent) != 1 {
		t.Fatalf("sent %d, want 1", len(c.sent))
	}
	body := c.sent[0].Body
	if !strings.Contains(body, "expiring in 7 day(s)") {
		t.Errorf("body: %q", body)
	}
	if !strings.Contains(body, "example.com") {
		t.Errorf("body missing domain")
	}
}

func TestNotifier_SchedulerError(t *testing.T) {
	c := &fakeClient{available: true}
	n := New(c, "casman", func() []string { return []string{"o@example.com"} })
	n.SchedulerError("backup_daily", "disk full")
	if !strings.Contains(c.sent[0].Body, `"backup_daily"`) {
		t.Errorf("expected task name in body: %q", c.sent[0].Body)
	}
	if !strings.Contains(c.sent[0].Body, "disk full") {
		t.Errorf("expected reason in body")
	}
}

func TestNotifier_SendErrorsLoggedNotPanic(t *testing.T) {
	c := &fakeClient{available: true, failNext: errors.New("boom")}
	n := New(c, "casman", func() []string { return []string{"x@example.com"} })
	// Should not panic; error is logged.
	n.BackupFailed("disk")
}

// Nil-receiver safety on the method itself is not part of the contract; the
// short-circuit lives in dispatch() and is exercised by the no-SMTP / no-
// recipient tests above.
