package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fibegg/sdk/fibe"
)

type statusLineOptions struct {
	Interactive     *bool
	FallbackStart   bool
	FallbackUpdates bool
	Interval        time.Duration
	Frames          []string
}

type statusLine struct {
	w               io.Writer
	interactive     bool
	fallbackStart   bool
	fallbackUpdates bool
	interval        time.Duration
	frames          []string

	mu        sync.Mutex
	wg        sync.WaitGroup
	stop      chan struct{}
	started   bool
	message   string
	frame     int
	lastWidth int
}

func newStatusLine(w io.Writer, opts statusLineOptions) *statusLine {
	interactive := isTerminalWriter(w)
	if opts.Interactive != nil {
		interactive = *opts.Interactive
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 120 * time.Millisecond
	}
	frames := opts.Frames
	if len(frames) == 0 {
		frames = []string{"-", "\\", "|", "/"}
	}
	return &statusLine{
		w:               w,
		interactive:     interactive,
		fallbackStart:   opts.FallbackStart,
		fallbackUpdates: opts.FallbackUpdates,
		interval:        interval,
		frames:          frames,
	}
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (s *statusLine) IsInteractive() bool {
	if s == nil {
		return false
	}
	return s.interactive
}

func (s *statusLine) Start(message string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.ensureStartedLocked()
	s.message = message
	if s.interactive {
		s.renderLocked()
		s.mu.Unlock()
		return
	}
	fallback := s.fallbackStart && message != ""
	s.mu.Unlock()
	if fallback {
		fmt.Fprintln(s.w, message)
	}
}

func (s *statusLine) Update(message string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.ensureStartedLocked()
	s.message = message
	if s.interactive {
		s.renderLocked()
		s.mu.Unlock()
		return
	}
	fallback := s.fallbackUpdates && message != ""
	s.mu.Unlock()
	if fallback {
		fmt.Fprintln(s.w, message)
	}
}

func (s *statusLine) Stop() {
	if s == nil {
		return
	}
	var stop chan struct{}
	s.mu.Lock()
	if s.stop != nil {
		stop = s.stop
		s.stop = nil
	}
	s.mu.Unlock()
	if stop != nil {
		close(stop)
		s.wg.Wait()
	}
	s.mu.Lock()
	if s.interactive && s.lastWidth > 0 {
		s.clearLocked()
	}
	s.started = false
	s.message = ""
	s.mu.Unlock()
}

func (s *statusLine) Progress(action string) fibe.ProgressFunc {
	return func(_ context.Context, event fibe.ProgressEvent) {
		s.Update(progressMessage(action, event))
	}
}

func progressMessage(action string, event fibe.ProgressEvent) string {
	status := strings.TrimSpace(event.Status)
	if status == "" {
		status = "waiting"
	}
	if action == "" {
		return "status: " + status
	}
	return action + ": " + status
}

func (s *statusLine) ensureStartedLocked() {
	if s.started {
		return
	}
	s.started = true
	if !s.interactive {
		return
	}
	s.stop = make(chan struct{})
	stop := s.stop
	interval := s.interval
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				if len(s.frames) > 0 {
					s.frame = (s.frame + 1) % len(s.frames)
				}
				s.renderLocked()
				s.mu.Unlock()
			}
		}
	}()
}

func (s *statusLine) renderLocked() {
	frame := ""
	if len(s.frames) > 0 {
		frame = s.frames[s.frame%len(s.frames)]
	}
	text := frame
	if s.message != "" {
		if text != "" {
			text += " "
		}
		text += s.message
	}
	width := len(text)
	padding := ""
	if s.lastWidth > width {
		padding = strings.Repeat(" ", s.lastWidth-width)
	}
	fmt.Fprintf(s.w, "\r%s%s", text, padding)
	s.lastWidth = width
}

func (s *statusLine) clearLocked() {
	fmt.Fprintf(s.w, "\r%s\r", strings.Repeat(" ", s.lastWidth))
	s.lastWidth = 0
}
