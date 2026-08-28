// Package history guarda, em JSONL append-only, tudo que o app alterou na
// máquina — é o que torna "desfazer" possível e auditável.
//
// Formato append-only por decisão de produto: barato, legível por humano,
// impossível de corromper por escrita parcial no meio do arquivo, e o usuário
// pode abrir no Bloco de Notas e conferir. Ver docs/arquitetura-app-desktop.md,
// seção 4 ("Histórico e desfazer tudo").
package history

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"optimizer/internal/tweak"
)

type Action string

const (
	ActionApply  Action = "apply"
	ActionRevert Action = "revert"
)

// Entry é uma linha do histórico.
type Entry struct {
	ID      string    `json:"id"`
	TweakID string    `json:"tweakId"`
	Action  Action    `json:"action"`
	At      time.Time `json:"at"`
	DryRun  bool      `json:"dryRun,omitempty"`
	Success bool      `json:"success"`
	Detail  string    `json:"detail,omitempty"`
	Error   string    `json:"error,omitempty"`
	// Snapshot é o estado ANTERIOR à ação — o que Revert precisa para desfazer.
	Snapshot tweak.Snapshot `json:"snapshot,omitempty"`
	// RestorePoint é o número de sequência do ponto de restauração criado antes
	// do lote, quando houve um.
	RestorePoint  uint64 `json:"restorePoint,omitempty"`
	RestartNeeded bool   `json:"restartNeeded,omitempty"`
	// BatchID identifica a transação/lote desta alteração.
	BatchID string `json:"batchId,omitempty"`
	// Origin identifica se a ação foi manual ou por perfil (ex: "perfil-jogo", "perfil-coding").
	Origin string `json:"origin,omitempty"`
}

// Store é o histórico em disco.
type Store struct {
	mu   sync.Mutex
	path string
}

// DefaultPath aponta para %LOCALAPPDATA%\Optimizer\history.jsonl.
func DefaultPath() (string, error) {
	dir, err := os.UserCacheDir() // no Windows: %LOCALAPPDATA%
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Optimizer", "history.jsonl"), nil
}

// Open abre (criando a pasta se preciso) o histórico no caminho dado.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("criando pasta do histórico: %w", err)
	}
	return &Store{path: path}, nil
}

func (s *Store) Path() string { return s.path }

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// Append grava uma entrada. Simulações (DryRun) também são registradas: o
// usuário deve conseguir ver o que o app pensou em fazer, não só o que fez.
func (s *Store) Append(e Entry) (Entry, error) {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.At.IsZero() {
		e.At = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return e, fmt.Errorf("abrindo histórico: %w", err)
	}
	defer f.Close()

	_ = lockFile(f)
	defer unlockFile(f)

	line, err := json.Marshal(e)
	if err != nil {
		return e, fmt.Errorf("serializando entrada do histórico: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return e, fmt.Errorf("gravando histórico: %w", err)
	}
	return e, nil
}

// All lê o histórico inteiro, em ordem cronológica. Linha corrompida é pulada
// (append-only: um desligamento no meio de uma escrita não pode inutilizar o
// histórico inteiro).
func (s *Store) All() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("abrindo histórico: %w", err)
	}
	defer f.Close()

	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return out, fmt.Errorf("lendo histórico: %w", err)
	}
	return out, nil
}

// Pending devolve, por tweak, o último Apply bem-sucedido que ainda NÃO foi
// desfeito — ou seja, exatamente o que o botão "desfazer" precisa.
func (s *Store) Pending() (map[string]Entry, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	pending := map[string]Entry{}
	for _, e := range all {
		if e.DryRun || !e.Success {
			continue
		}
		switch e.Action {
		case ActionApply:
			pending[e.TweakID] = e
		case ActionRevert:
			delete(pending, e.TweakID)
		}
	}
	return pending, nil
}

// PendingFor devolve o snapshot a desfazer de um tweak específico.
func (s *Store) PendingFor(tweakID string) (Entry, bool, error) {
	pending, err := s.Pending()
	if err != nil {
		return Entry{}, false, err
	}
	e, ok := pending[tweakID]
	return e, ok, nil
}

// PendingForBatch devolve todas as entradas ativas que pertencem a um lote específico.
func (s *Store) PendingForBatch(batchID string) ([]Entry, error) {
	pending, err := s.Pending()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range pending {
		if e.BatchID == batchID {
			out = append(out, e)
		}
	}
	return out, nil
}

// ActiveBatchForOrigin devolve o ID do último lote ativo para uma determinada origem (ex: "perfil-jogo").
func (s *Store) ActiveBatchForOrigin(origin string) (string, []Entry, error) {
	pending, err := s.Pending()
	if err != nil {
		return "", nil, err
	}
	var out []Entry
	var lastBatchID string
	for _, e := range pending {
		if e.Origin == origin {
			out = append(out, e)
			lastBatchID = e.BatchID
		}
	}
	return lastBatchID, out, nil
}
