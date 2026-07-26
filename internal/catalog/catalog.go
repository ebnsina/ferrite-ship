// Package catalog describes the software Ferrite Ship can install on a server
// and turns a choice into a playbook.
//
// Every tool is a compose project under /opt/ferrite/<id>, which means one
// mechanism installs, inspects, restarts and removes all of them, and the
// owner can read the file to see exactly what is running. Adding a tool is a
// data change here rather than new Go anywhere else.
package catalog

import "errors"

// ErrUnknownTool is returned for an id that is not in the catalogue.
var ErrUnknownTool = errors.New("catalog: no such tool")

// Tool is one installable piece of software.
type Tool struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	// Category groups the catalogue in the UI.
	Category string `json:"category"`
	// Icon is a lucide name the dashboard imports.
	Icon string `json:"icon"`
	// Image is pinned to a major line: patch updates arrive when the tool is
	// updated on purpose, never silently underneath a running server.
	Image   string `json:"image"`
	Version string `json:"version"`
	Ports   []Port `json:"ports"`
	// Access describes how to connect once it is running, or is nil for tools
	// that are not reached with a connection string.
	Access *Access `json:"access,omitempty"`
	// DataNote says in plain language what removing this tool leaves behind.
	DataNote string `json:"dataNote"`
	// KeepsData is whether removing this tool can leave something behind, and
	// therefore whether it is worth offering to delete it. Derived from Volumes
	// in All() rather than written out per tool, so the two cannot disagree —
	// offering "also delete the data" for a tool that stores none would be a
	// frightening question with no answer.
	KeepsData bool `json:"keepsData"`
	// NeedsAddress marks a tool that must be told the server's public address
	// at install time — see Install.Address.
	NeedsAddress bool `json:"-"`
	// Volumes are the compose volume names holding this tool's data. Named
	// explicitly rather than derived, because they are what "also delete the
	// data" deletes and a wrong guess there is unrecoverable.
	Volumes []string `json:"-"`

	// compose is the file defining the containers. It is the same for every
	// installation and contains no secret: values are interpolated from the env
	// file beside it, so the owner can read what is running without reading
	// credentials, and the file can be diffed to spot local edits.
	compose string
	// env renders the KEY=value lines compose interpolates from. This is the
	// only file that holds a secret, and it is written 0600.
	env func(in Install) []string
}

// Port is one door a tool opens on the server.
type Port struct {
	Number   int    `json:"number"`
	Protocol string `json:"protocol"`
	// Purpose is plain language: "Database connections", not "pgwire".
	Purpose string `json:"purpose"`
	// Public opens the port in the firewall. Databases are deliberately not
	// public: they listen on loopback only, and are reached over an SSH tunnel.
	// A Postgres exposed to the internet is found by scanners within minutes.
	Public bool `json:"public"`
}

// Access is what someone needs to connect to a running tool.
type Access struct {
	// Scheme is the URL scheme of the connection string.
	Scheme   string `json:"scheme"`
	Username string `json:"username"`
	Database string `json:"database"`
	Port     int    `json:"port"`
}

// Install is one tool on one server: everything that makes its files unique.
type Install struct {
	Tool     Tool
	Password string
	// Address is how the outside world reaches this server. Only tools with
	// NeedsAddress use it: WebRTC hands clients an address to connect back on,
	// and a container only ever knows its own private one.
	Address string
}

// All returns the catalogue in display order.
func All() []Tool {
	tools := []Tool{postgres, redis, clickhouse, mediamtx}
	for i := range tools {
		tools[i].KeepsData = len(tools[i].Volumes) > 0
	}
	return tools
}

// Find looks a tool up by id.
func Find(id string) (Tool, error) {
	for _, tool := range All() {
		if tool.ID == id {
			return tool, nil
		}
	}
	return Tool{}, ErrUnknownTool
}

// NeedsPassword reports whether a credential is generated at install time.
func (t Tool) NeedsPassword() bool { return t.Access != nil }

// PublicPorts are the ports the firewall must open for this tool.
func (t Tool) PublicPorts() []Port {
	var public []Port
	for _, port := range t.Ports {
		if port.Public {
			public = append(public, port)
		}
	}
	return public
}
