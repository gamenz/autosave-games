// Package notify shows desktop notifications filtered by the notify level.
package notify

import (
	"log/slog"
	"sync"

	"autosave-games/internal/config"
)

// Sender delivers one notification.
type Sender interface {
	Send(title, body string) error
}

// Notifier logs every event and forwards it to Sender per level:
// Info only at "all", Error at "all"/"errors", Critical always.
type Notifier struct {
	mu     sync.Mutex
	level  config.Notify
	sender Sender
	log    *slog.Logger
}

// NewNotifier creates a Notifier.
func NewNotifier(level config.Notify, s Sender, log *slog.Logger) *Notifier {
	return &Notifier{level: level, sender: s, log: log}
}

// SetLevel changes the level (after a config reload).
func (n *Notifier) SetLevel(level config.Notify) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.level = level
}

// Info reports a routine success.
func (n *Notifier) Info(title, body string) {
	n.log.Info(title, "detail", body)
	n.send(n.currentLevel() == config.NotifyAll, title, body)
}

// Error reports a problem the user should look at.
func (n *Notifier) Error(title, body string) {
	n.log.Warn(title, "detail", body)
	n.send(n.currentLevel() != config.NotifyNone, title, body)
}

// Notice reports the result of a user action; always shown.
func (n *Notifier) Notice(title, body string) {
	n.log.Info(title, "detail", body)
	n.send(true, title, body)
}

// Critical reports a failure that is always shown (e.g. a write error).
func (n *Notifier) Critical(title, body string) {
	n.log.Error(title, "detail", body)
	n.send(true, title, body)
}

func (n *Notifier) currentLevel() config.Notify {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.level
}

func (n *Notifier) send(enabled bool, title, body string) {
	if !enabled {
		return
	}
	if err := n.sender.Send(title, body); err != nil {
		n.log.Warn("notification failed", "title", title, "err", err)
	}
}

// RecordingSender records notifications as "title: body" (tests).
type RecordingSender struct {
	mu   sync.Mutex
	Sent []string
}

// Send implements Sender.
func (r *RecordingSender) Send(title, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Sent = append(r.Sent, title+": "+body)
	return nil
}

// Messages returns a copy of the recorded notifications.
func (r *RecordingSender) Messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.Sent...)
}
