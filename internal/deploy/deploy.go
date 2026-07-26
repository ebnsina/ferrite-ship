// Package deploy turns "here is my repository" into "it is running at this
// address".
//
// This is what the rest of the product was building toward: a hardened server
// with databases on it is scaffolding until the thing you wrote is running on
// it too. The shape follows internal/catalog — a compose project per app under
// a predictable path — so one mechanism starts, inspects, restarts and removes
// applications and tools alike.
package deploy

import (
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

const (
	root = "/opt/ferrite"
	// network is the bridge the proxy and every application share, so the
	// proxy can reach them by name and nothing has to publish a port.
	network = "ferrite"

	ingressDir  = root + "/ingress"
	ingressFile = ingressDir + "/compose.yaml"
	caddyfile   = ingressDir + "/Caddyfile"

	appsDir = root + "/apps"

	// nixpacksVersion is pinned. A build tool that changes underneath a
	// deployment turns "it worked yesterday" into a mystery.
	nixpacksVersion = "1.41.0"
)

// App is one application on one server.
type App struct {
	ID string
	// Name is what the owner calls it.
	Name string
	// Repository is a git URL. Public today; a deploy key comes later.
	Repository string
	Branch     string
	// Domain is where it should answer. Empty means it runs but is only
	// reachable from the server itself.
	Domain string
	// Port is the port the application listens on inside its container.
	Port int
	// Env are the variables it needs, most often a database URL from a tool
	// installed on the same machine.
	Env map[string]string
}

// Site is one route the proxy serves.
type Site struct {
	Domain    string
	Container string
	Port      int
}

func appDir(id string) string    { return appsDir + "/" + id }
func sourceDir(id string) string { return appDir(id) + "/src" }

// Container is the name the proxy reaches an application by.
func Container(id string) string { return "ferrite-app-" + id }

// Image is the tag a build produces.
func Image(id string) string { return "ferrite-app-" + id + ":latest" }

func compose(dir string, args ...string) string {
	return "docker compose --project-directory " + steps.Quote(dir) +
		" -f " + steps.Quote(dir+"/compose.yaml") + " " + strings.Join(args, " ")
}

// matches tests whether a file already holds exactly this content.
func matches(path, content string) string {
	return "printf '%s' " + steps.Quote(content) + " | cmp -s - " + steps.Quote(path)
}

func write(path, content, mode string) []string {
	return []string{
		"test -f " + steps.Quote(path) + " || install -m " + mode + " /dev/null " + steps.Quote(path),
		"chmod " + mode + " " + steps.Quote(path),
		"printf '%s' " + steps.Quote(content) + " > " + steps.Quote(path),
	}
}
