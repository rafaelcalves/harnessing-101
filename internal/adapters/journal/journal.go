// Package journal implements the local OutputJournal foundation shared by
// item 4 crash recovery and item 5's later complete-capture work.
package journal

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

type entry struct {
	Offset     uint64    `json:"offset"`
	Channel    string    `json:"channel"`
	Data       string    `json:"data"`
	CapturedAt time.Time `json:"capturedAt"`
}

type Journal struct {
	root string
	mu   sync.Mutex
}

func Open(root string) (*Journal, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Journal{root: root}, nil
}

func (j *Journal) Append(ctx context.Context, runID domain.RunID, channel string, data []byte, capturedAt time.Time) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	path := filepath.Join(j.root, string(runID)+".jsonl")
	var offset uint64
	if f, err := os.Open(path); err == nil {
		s := bufio.NewScanner(f)
		for s.Scan() {
			var e entry
			if json.Unmarshal(s.Bytes(), &e) == nil {
				offset = e.Offset + uint64(len(decoded(e.Data)))
			}
		}
		_ = f.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	e := entry{Offset: offset, Channel: channel, Data: base64.StdEncoding.EncodeToString(data), CapturedAt: capturedAt}
	b, err := json.Marshal(e)
	if err != nil {
		return 0, err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	return offset, nil
}

func (j *Journal) ReadAfter(ctx context.Context, runID domain.RunID, afterOffset uint64, limit int) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	out := domain.RunOutput{RunID: runID, CaptureStatus: domain.CaptureUnknown}
	path := filepath.Join(j.root, string(runID)+".jsonl")
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(f)
	used := 0
	for s.Scan() {
		var e entry
		if json.Unmarshal(s.Bytes(), &e) != nil {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(e.Data)
		if err != nil {
			continue
		}
		if e.Offset < afterOffset {
			continue
		}
		if limit > 0 && used+len(data) > limit {
			break
		}
		out.Chunks = append(out.Chunks, domain.OutputChunk{Offset: e.Offset, Channel: e.Channel, Bytes: data, CapturedAt: e.CapturedAt})
		used += len(data)
	}
	_ = f.Close()
	if err := s.Err(); err != nil {
		return nil, err
	}
	if status, err := os.ReadFile(filepath.Join(j.root, string(runID)+".status")); err == nil {
		out.CaptureStatus = domain.CaptureStatus(status)
	}
	return out, nil
}

func (j *Journal) Follow(ctx context.Context, runID domain.RunID, offset uint64) (<-chan any, error) {
	ch := make(chan any)
	close(ch)
	return ch, nil
}

func (j *Journal) Finish(ctx context.Context, runID domain.RunID, outcome any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	status := domain.CaptureUnknown
	if s, ok := outcome.(domain.CaptureStatus); ok {
		status = s
	}
	if status == "" {
		status = domain.CaptureInterrupted
	}
	f, err := os.OpenFile(filepath.Join(j.root, string(runID)+".status"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = f.Write([]byte(status)); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	return err
}

func decoded(s string) []byte { b, _ := base64.StdEncoding.DecodeString(s); return b }

var _ interface {
	Append(context.Context, domain.RunID, string, []byte, time.Time) (uint64, error)
	ReadAfter(context.Context, domain.RunID, uint64, int) (any, error)
	Follow(context.Context, domain.RunID, uint64) (<-chan any, error)
	Finish(context.Context, domain.RunID, any) error
} = (*Journal)(nil)
