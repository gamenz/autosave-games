package notify

import (
	"errors"
	"log/slog"
	"slices"
	"testing"

	"autosave-games/internal/config"
)

func TestNotifierLevels(t *testing.T) {
	cases := []struct {
		level config.Notify
		want  []string
	}{
		{config.NotifyAll, []string{"i: 1", "e: 2", "c: 3"}},
		{config.NotifyErrors, []string{"e: 2", "c: 3"}},
		{config.NotifyNone, []string{"c: 3"}},
	}
	for _, tc := range cases {
		rec := &RecordingSender{}
		n := NewNotifier(tc.level, rec, slog.New(slog.DiscardHandler))
		n.Info("i", "1")
		n.Error("e", "2")
		n.Critical("c", "3")
		if got := rec.Messages(); !slices.Equal(got, tc.want) {
			t.Errorf("level %s: got %v, want %v", tc.level, got, tc.want)
		}
	}
}

func TestSetLevel(t *testing.T) {
	rec := &RecordingSender{}
	n := NewNotifier(config.NotifyNone, rec, slog.New(slog.DiscardHandler))
	n.SetLevel(config.NotifyAll)
	n.Info("i", "1")
	if len(rec.Messages()) != 1 {
		t.Fatalf("messages = %v", rec.Messages())
	}
}

type failingSender struct{}

func (failingSender) Send(string, string) error { return errors.New("no toast") }

func TestSendFailureIsSwallowed(t *testing.T) {
	n := NewNotifier(config.NotifyAll, failingSender{}, slog.New(slog.DiscardHandler))
	n.Critical("c", "boom") // must not panic
}
