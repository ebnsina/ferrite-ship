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
	// Accent is the tool's own brand colour, so a catalogue of four grey rows
	// becomes four things you recognise before reading a word. The dashboard
	// tints rather than uses it flat, because these are real brand colours and
	// some of them (ClickHouse yellow) are unreadable as-is on a white page.
	Accent string `json:"accent"`
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
	// HasConsole is whether this tool can be queried from the dashboard.
	HasConsole bool `json:"hasConsole"`
	// ConsoleLanguage names what you type into it, e.g. "SQL".
	ConsoleLanguage string `json:"consoleLanguage,omitempty"`
	// ConsolePlaceholder is an example query for an empty editor.
	ConsolePlaceholder string `json:"consolePlaceholder,omitempty"`
	// ConsolePresets are ready-made queries to start from.
	ConsolePresets []Preset `json:"consolePresets,omitempty"`
	// KeepsData is whether removing this tool can leave something behind, and
	// therefore whether it is worth offering to delete it. Derived from Volumes
	// in All() rather than written out per tool, so the two cannot disagree —
	// offering "also delete the data" for a tool that stores none would be a
	// frightening question with no answer.
	KeepsData bool `json:"keepsData"`
	// NeedsAddress marks a tool that must be told the server's public address
	// at install time — see Install.Address.
	NeedsAddress bool `json:"-"`
	// NeedsDomain marks a tool that cannot be installed until the server has a
	// domain, because everything it does is answer on one. Declared rather
	// than inferred so the install is refused with a sentence explaining what
	// to do, instead of rendering a compose file that fails on a missing
	// variable.
	NeedsDomain bool `json:"needsDomain"`
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
	// console is how you query this tool from the dashboard, or nil for one
	// there is nothing to query.
	console *Console
	// backup is how this tool is copied off the server, or nil where that is
	// not built yet.
	backup *Backup
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
	// Domain is the name whose wildcard record points at this server, and
	// Email is where Let's Encrypt writes about the certificates issued under
	// it. Both are empty on a server nobody has given a domain, which is why
	// every compose file that reads them belongs to a tool with NeedsDomain.
	Domain string
	Email  string
}

// All returns the catalogue in display order.
func All() []Tool {
	// Traefik first: it is the one thing here that other tools are reached
	// through, so it reads as infrastructure rather than as another database.
	tools := []Tool{traefik, postgres, redis, clickhouse, mediamtx}
	for i := range tools {
		tools[i].KeepsData = len(tools[i].Volumes) > 0
		if spec, ok := tools[i].ConsoleSpec(); ok {
			tools[i].HasConsole = true
			tools[i].ConsoleLanguage = spec.Language
			tools[i].ConsolePlaceholder = spec.Placeholder
			tools[i].ConsolePresets = spec.Presets
		}
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
