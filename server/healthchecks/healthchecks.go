package healthchecks

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/efixler/scrape/store"
)

func Handler(root string, dbo store.Observable) http.Handler {
	root = strings.TrimSuffix(root, "/")
	mux := http.NewServeMux()
	mux.HandleFunc("/heartbeat", heartbeat)
	mux.Handle("/health", HealthHandler(dbo))
	switch root {
	case "":
		return mux
	default:
		return http.StripPrefix(root, mux)
	}
}

type Observer func() (any, error)

func heartbeat(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type health struct {
	Application Application `json:"application"`
	Memory      *Memory     `json:"memory"`
	dbObserver  Observer
}

func HealthHandler(dbObservable store.Observable) http.Handler {
	h := health{
		Application: Application{
			StartTime: time.Now().UTC().Format(time.RFC3339),
		},
	}
	if dbObservable != nil {
		h.dbObserver = dbObservable.Stats
	} else {
		slog.Warn("Healthchecks: no database observer, will not include database stats")
	}
	return h
}

func (h health) MarshalJSON() ([]byte, error) {
	type alias health
	var dbStats interface{}
	if h.dbObserver != nil {
		dbStats, _ = h.dbObserver()
	}

	return json.Marshal(&struct {
		*alias
		Database interface{} `json:"database,omitempty"`
	}{
		alias:    (*alias)(&h),
		Database: dbStats,
	})
}

func (h health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	h.read()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(h)
}

func (h *health) read() error {
	h.Application.GoroutineCount = runtime.NumGoroutine()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	h.Memory = &Memory{
		HeapAlloc:  m.HeapAlloc,
		HeapSys:    m.HeapSys,
		StackSys:   m.StackSys,
		StackInUse: m.StackInuse,
		System:     m.Sys,
		LastGC:     m.LastGC,
		NumGC:      m.NumGC,
	}
	return nil
}

type Memory struct {
	System     uint64 `json:"-"`
	HeapSys    uint64 `json:"-"`
	HeapAlloc  uint64 `json:"-"`
	StackSys   uint64 `json:"-"`
	StackInUse uint64 `json:"-"`
	LastGC     uint64 `json:"-"`
	NumGC      uint32 `json:"gc_count"`
}

func (m Memory) MarshalJSON() ([]byte, error) {
	type alias Memory
	mb := func(b uint64) uint64 {
		return b / 1024 / 1024
	}
	lgc := time.Since(time.Unix(0, int64(m.LastGC))) / time.Second
	return json.Marshal(&struct {
		System       uint64        `json:"application_total_mb"`
		HeapSysMB    uint64        `json:"heap_system_mb"`
		HeapAllocMB  uint64        `json:"heap_allocated_mb"`
		StackSysMB   uint64        `json:"stack_system_mb"`
		StackInUseMB uint64        `json:"stack_in_use_mb"`
		TimeSinceGC  time.Duration `json:"seconds_since_last_gc"`
		*alias
	}{
		alias:        (*alias)(&m),
		System:       mb(m.System),
		HeapSysMB:    mb(m.HeapSys),
		HeapAllocMB:  mb(m.HeapAlloc),
		StackSysMB:   mb(m.StackSys),
		StackInUseMB: mb(m.StackInUse),
		TimeSinceGC:  lgc,
	})
}

func (m *Memory) UnmarshalJSON(data []byte) error {
	var proxy struct {
		System       uint64        `json:"application_total_mb"`
		HeapSysMB    uint64        `json:"heap_system_mb"`
		HeapAllocMB  uint64        `json:"heap_allocated_mb"`
		StackSysMB   uint64        `json:"stack_system_mb"`
		StackInUseMB uint64        `json:"stack_in_use_mb"`
		TimeSinceGC  time.Duration `json:"seconds_since_last_gc"`
	}
	err := json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}
	m.System = proxy.System * 1024 * 1024
	m.HeapSys = proxy.HeapSysMB * 1024 * 1024
	m.HeapAlloc = proxy.HeapAllocMB * 1024 * 1024
	m.StackSys = proxy.StackSysMB * 1024 * 1024
	m.StackInUse = proxy.StackInUseMB * 1024 * 1024
	m.LastGC = uint64(time.Now().Add(-1 * proxy.TimeSinceGC).UnixNano())
	return nil
}

type Application struct {
	StartTime      string `json:"start_time"`
	GoroutineCount int    `json:"goroutine_count"`
}
