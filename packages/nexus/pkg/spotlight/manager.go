package spotlight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/inizio/nexus/packages/nexus/pkg/store"
)

type Forward struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Service     string    `json:"service"`
	RemotePort  int       `json:"remotePort"`
	LocalPort   int       `json:"localPort"`
	Host        string    `json:"host"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ExposeSpec struct {
	WorkspaceID string `json:"workspaceId"`
	Service     string `json:"service"`
	RemotePort  int    `json:"remotePort"`
	LocalPort   int    `json:"localPort"`
	Host        string `json:"host,omitempty"`
}

type Manager struct {
	mu        sync.RWMutex
	forwards  map[string]*Forward
	localToID map[int]string
	listeners map[string]net.Listener
	repo      spotlightRepository
}

type spotlightRepository interface {
	UpsertSpotlightForwardRow(row store.SpotlightForwardRow) error
	DeleteSpotlightForwardRow(id string) error
	ListSpotlightForwardRows() ([]store.SpotlightForwardRow, error)
}

func NewManager() *Manager {
	return &Manager{
		forwards:  make(map[string]*Forward),
		localToID: make(map[int]string),
		listeners: make(map[string]net.Listener),
	}
}

func NewManagerWithRepository(repo spotlightRepository) (*Manager, error) {
	m := NewManager()
	m.repo = repo
	if err := m.hydrateFromRepository(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) hydrateFromRepository() error {
	if m.repo == nil {
		return nil
	}

	rows, err := m.repo.ListSpotlightForwardRows()
	if err != nil {
		return fmt.Errorf("list spotlight forwards: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.forwards = make(map[string]*Forward, len(rows))
	m.localToID = make(map[int]string, len(rows))
	for _, row := range rows {
		if row.ID == "" || row.LocalPort <= 0 || len(row.Payload) == 0 {
			continue
		}

		var fwd Forward
		if err := json.Unmarshal(row.Payload, &fwd); err != nil {
			continue
		}
		if fwd.ID == "" {
			fwd.ID = row.ID
		}
		if fwd.WorkspaceID == "" {
			fwd.WorkspaceID = row.WorkspaceID
		}
		if fwd.LocalPort <= 0 {
			fwd.LocalPort = row.LocalPort
		}
		if fwd.CreatedAt.IsZero() {
			fwd.CreatedAt = row.CreatedAt
		}
		if fwd.ID == "" || fwd.RemotePort <= 0 || fwd.LocalPort <= 0 {
			continue
		}
		if _, dup := m.forwards[fwd.ID]; dup {
			continue
		}
		if _, dup := m.localToID[fwd.LocalPort]; dup {
			continue
		}

		copy := fwd
		m.forwards[copy.ID] = &copy
		m.localToID[copy.LocalPort] = copy.ID
	}

	return nil
}

func (m *Manager) Expose(_ context.Context, spec ExposeSpec) (*Forward, error) {
	if spec.RemotePort <= 0 || spec.LocalPort <= 0 {
		return nil, fmt.Errorf("remotePort and localPort must be > 0")
	}

	m.mu.Lock()

	if existing, ok := m.localToID[spec.LocalPort]; ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("local port %d already in use by %s", spec.LocalPort, existing)
	}

	host := spec.Host
	if host == "" {
		host = "127.0.0.1"
	}

	now := time.Now().UTC()
	id := fmt.Sprintf("spot-%d", now.UnixNano())
	if _, exists := m.forwards[id]; exists {
		for suffix := 2; ; suffix++ {
			candidate := fmt.Sprintf("%s-%d", id, suffix)
			if _, dup := m.forwards[candidate]; !dup {
				id = candidate
				break
			}
		}
	}
	fwd := &Forward{
		ID:          id,
		WorkspaceID: spec.WorkspaceID,
		Service:     spec.Service,
		RemotePort:  spec.RemotePort,
		LocalPort:   spec.LocalPort,
		Host:        host,
		CreatedAt:   now,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, spec.LocalPort))
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("bind local spotlight port: %w", err)
	}

	m.forwards[id] = fwd
	m.localToID[spec.LocalPort] = id
	m.listeners[id] = listener
	targetAddr := fmt.Sprintf("%s:%d", host, spec.RemotePort)
	go serveForward(listener, targetAddr)

	if m.repo != nil {
		payload, err := json.Marshal(fwd)
		if err != nil {
			delete(m.forwards, id)
			delete(m.localToID, spec.LocalPort)
			if l, ok := m.listeners[id]; ok {
				_ = l.Close()
				delete(m.listeners, id)
			}
			m.mu.Unlock()
			return nil, fmt.Errorf("marshal spotlight forward: %w", err)
		}
		if err := m.repo.UpsertSpotlightForwardRow(store.SpotlightForwardRow{
			ID:          fwd.ID,
			WorkspaceID: fwd.WorkspaceID,
			LocalPort:   fwd.LocalPort,
			Payload:     payload,
			CreatedAt:   fwd.CreatedAt,
		}); err != nil {
			delete(m.forwards, id)
			delete(m.localToID, spec.LocalPort)
			if l, ok := m.listeners[id]; ok {
				_ = l.Close()
				delete(m.listeners, id)
			}
			m.mu.Unlock()
			return nil, fmt.Errorf("persist spotlight forward: %w", err)
		}
	}

	m.mu.Unlock()

	copy := *fwd
	return &copy, nil
}

func (m *Manager) List(workspaceID string) []*Forward {
	m.mu.RLock()
	all := make([]*Forward, 0, len(m.forwards))
	for _, fwd := range m.forwards {
		if workspaceID == "" || fwd.WorkspaceID == workspaceID {
			copy := *fwd
			all = append(all, &copy)
		}
	}
	m.mu.RUnlock()

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})

	return all
}

func (m *Manager) Close(id string) bool {
	m.mu.Lock()
	fwd, ok := m.forwards[id]
	if !ok {
		m.mu.Unlock()
		return false
	}
	listener := m.listeners[id]

	if m.repo != nil {
		if err := m.repo.DeleteSpotlightForwardRow(id); err != nil {
			m.mu.Unlock()
			return false
		}
	}

	delete(m.forwards, id)
	delete(m.localToID, fwd.LocalPort)
	delete(m.listeners, id)
	m.mu.Unlock()
	if listener != nil {
		_ = listener.Close()
	}
	return true
}

func serveForward(listener net.Listener, targetAddr string) {
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			return
		}
		go proxyTCP(clientConn, targetAddr)
	}
}

func proxyTCP(clientConn net.Conn, targetAddr string) {
	upstreamConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		_ = clientConn.Close()
		return
	}

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstreamConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(clientConn, upstreamConn)
		done <- struct{}{}
	}()
	<-done
	_ = clientConn.Close()
	_ = upstreamConn.Close()
}
