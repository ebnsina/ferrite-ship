package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// serverView is the wire shape the dashboard consumes. It is deliberately not
// store.Server: credentials must never be able to leak into a response by
// someone adding a field to the storage struct.
type serverView struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Hostname        string    `json:"hostname"`
	IPAddress       string    `json:"ipAddress"`
	Region          string    `json:"region"`
	OperatingSystem string    `json:"operatingSystem"`
	AgentVersion    string    `json:"agentVersion"`
	Status          string    `json:"status"`
	CPUUsage        float64   `json:"cpuUsage"`
	Memory          usageView `json:"memory"`
	Disk            usageView `json:"disk"`
	UptimeMs        int64     `json:"uptimeMs"`
	LastSeenAt      string    `json:"lastSeenAt"`
	Services        []string  `json:"services"`
	Connection      string    `json:"connectionKind"`
}

type usageView struct {
	UsedBytes  int64 `json:"usedBytes"`
	TotalBytes int64 `json:"totalBytes"`
}

func toServerView(s store.Server) serverView {
	lastSeen := s.CreatedAt
	if s.LastSeenAt != nil {
		lastSeen = *s.LastSeenAt
	}

	hostname := s.Facts.Hostname
	if hostname == "" {
		hostname = s.Host
	}
	// Prefer the address the control plane connects on. `hostname -I` reports
	// the first interface, which on most providers is a private address the
	// owner would not recognise — the real server showed 10.65.x.x rather than
	// its public IP.
	ip := s.Host
	if ip == "" {
		ip = s.Facts.IPAddress
	}

	return serverView{
		ID:              s.ID,
		Name:            s.Name,
		Hostname:        hostname,
		IPAddress:       ip,
		Region:          s.Region,
		OperatingSystem: s.Facts.OperatingSystem,
		// No agent yet; the field exists so the UI does not change when it lands.
		AgentVersion: "ssh",
		Status:       string(s.Status),
		CPUUsage:     s.Facts.CPUUsage,
		Memory: usageView{
			UsedBytes:  s.Facts.MemoryUsedBytes,
			TotalBytes: s.Facts.MemoryTotalBytes,
		},
		Disk: usageView{
			UsedBytes:  s.Facts.DiskUsedBytes,
			TotalBytes: s.Facts.DiskTotalBytes,
		},
		UptimeMs:   s.Facts.UptimeMs,
		LastSeenAt: lastSeen.UTC().Format(time.RFC3339),
		Services:   s.Services,
		Connection: string(s.Kind),
	}
}

func (a *API) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	views := make([]serverView, 0, len(servers))
	for _, s := range servers {
		views = append(views, toServerView(s))
	}
	writeJSON(w, http.StatusOK, views)
}

type createServerRequest struct {
	Name string `json:"name"`
	// Kind is "demo" or "ssh".
	Kind       string `json:"connectionKind"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Region     string `json:"region"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	// PublicKey is installed for the admin account the baseline creates.
	// Without one that account cannot be logged into, and password logins are
	// left on rather than locking the owner out.
	PublicKey string `json:"publicKey"`
}

func (a *API) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req createServerRequest
	if err := decodeJSON(r, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "parse",
			"We could not read that request. Check the fields and try again.")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "parse", "Give the server a name.")
		return
	}

	kind := store.ConnectionKind(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = store.ConnectionDemo
	}

	srv := store.Server{
		ID:        ids.New("srv"),
		Name:      req.Name,
		Kind:      kind,
		Region:    strings.TrimSpace(req.Region),
		Status:    store.StatusUnknown,
		Services:  []string{},
		PublicKey: strings.TrimSpace(req.PublicKey),
		CreatedAt: time.Now().UTC(),
	}

	switch kind {
	case store.ConnectionDemo:
		srv.Region = orDefault(srv.Region, "Simulated")

	case store.ConnectionSSH:
		if err := a.applySSHDetails(&srv, req); err != nil {
			a.writeError(w, http.StatusBadRequest, "parse", err.Error())
			return
		}

	default:
		a.writeError(w, http.StatusBadRequest, "parse",
			`Connection must be either "demo" or "ssh".`)
		return
	}

	if err := a.store.CreateServer(r.Context(), srv); err != nil {
		a.writeStoreError(w, err)
		return
	}

	a.log.Info("server connected", "id", srv.ID, "name", srv.Name, "kind", srv.Kind)
	writeJSON(w, http.StatusCreated, toServerView(srv))
}

func (a *API) applySSHDetails(srv *store.Server, req createServerRequest) error {
	host := strings.TrimSpace(req.Host)
	user := strings.TrimSpace(req.User)

	if host == "" {
		return errText("Enter the server's address.")
	}
	if user == "" {
		return errText("Enter the username to log in with.")
	}
	if req.Password == "" && req.PrivateKey == "" {
		return errText("Provide either a password or a private key.")
	}

	port := req.Port
	if port == 0 {
		port = 22
	}
	if port < 1 || port > 65535 {
		return errText("That port number is not valid.")
	}

	sealedPassword, err := a.sealer.Seal(req.Password)
	if err != nil {
		return errText("We could not store that password safely.")
	}
	sealedKey, err := a.sealer.Seal(req.PrivateKey)
	if err != nil {
		return errText("We could not store that key safely.")
	}

	srv.Host = host
	srv.Port = port
	srv.User = user
	srv.SealedPassword = sealedPassword
	srv.SealedPrivateKey = sealedKey
	srv.Region = orDefault(srv.Region, "Unknown")
	return nil
}

func (a *API) handleGetServer(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toServerView(server))
}

func (a *API) handleServerJobs(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	jobs, err := a.store.ListJobsForServer(r.Context(), server.ID, 20)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	views := make([]activityView, 0, len(jobs))
	for _, job := range jobs {
		views = append(views, activityView{
			ID:         job.ID,
			Title:      job.Title,
			ServerName: server.Name,
			Actor:      job.Actor,
			Status:     string(job.Status),
			StartedAt:  job.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
			DurationMs: job.DurationMs(),
		})
	}
	writeJSON(w, http.StatusOK, views)
}

func (a *API) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteServer(r.Context(), r.PathValue("id")); err != nil {
		a.writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type textError string

func (e textError) Error() string { return string(e) }

func errText(msg string) error { return textError(msg) }

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
