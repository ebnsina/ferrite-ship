package deploy

import (
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// ingressCompose runs Caddy, which is the only thing on the server holding
// ports 80 and 443.
//
// Caddy rather than nginx: it obtains and renews certificates on its own, and
// the alternative is a certbot cron job that fails silently three months from
// now. Its config is generated from the applications that exist, so adding one
// is not a text-editing exercise.
const ingressCompose = `# Managed by Ferrite Ship. Edits are replaced on the next deploy.
name: ferrite-ingress

services:
  caddy:
    image: caddy:2-alpine
    container_name: ferrite-ingress
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      # Certificates live in a volume: losing them means asking Let's Encrypt
      # for new ones on every restart, which hits a rate limit quickly.
      - data:/data
      - config:/config
    networks:
      - ferrite

volumes:
  data:
  config:

networks:
  ferrite:
    external: true
`

// Caddyfile renders the routes for every application that has a domain.
func Caddyfile(sites []Site) string {
	var out strings.Builder

	out.WriteString("# Managed by Ferrite Ship. Edits are replaced on the next deploy.\n")
	// The admin API stays on, deliberately. `caddy reload` works by posting the
	// new config to it, so turning it off makes every routing change a restart
	// — which drops connections. It listens on localhost inside the container
	// and no port is published, so nothing outside can reach it.
	out.WriteString("{\n\tadmin localhost:2019\n}\n\n")

	if len(sites) == 0 {
		// Caddy will not start on an empty config, and a proxy that refuses to
		// run is a proxy that cannot serve the first app you add.
		out.WriteString("# No applications have a domain yet.\n")
		out.WriteString(":80 {\n\trespond \"Nothing is published here yet.\" 404\n}\n")
		return out.String()
	}

	for _, site := range sites {
		out.WriteString(site.Domain + " {\n")
		out.WriteString("\treverse_proxy " + site.Container + ":" + strconv.Itoa(site.Port) + "\n")
		out.WriteString("}\n\n")
	}

	return out.String()
}

// IngressSteps makes sure the proxy exists and is serving these routes.
func IngressSteps(sites []Site) []steps.Step {
	config := Caddyfile(sites)

	return []steps.Step{
		steps.Shell(steps.ShellSpec{
			ID:    "ingress-config",
			Title: "Write the routing rules",
			Check: matches(ingressFile, ingressCompose) + " && " + matches(caddyfile, config),
			Apply: append(
				append(
					[]string{"install -d -m 755 " + steps.Quote(ingressDir)},
					write(ingressFile, ingressCompose, "644")...,
				),
				write(caddyfile, config, "644")...,
			),
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "ingress-up",
			Title: "Start the front door",
			Check: `test "$(docker inspect -f '{{.State.Running}}' ferrite-ingress 2>/dev/null)" = "true"`,
			Apply: []string{compose(ingressDir, "up", "-d", "--wait")},
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "ingress-reload",
			Title: "Apply the routing rules",
			// No check: reloading is cheap, idempotent, and the alternative is
			// comparing Caddy's running config with the file, which is a lot of
			// machinery to avoid a sub-second command.
			Apply: []string{
				// Validate first. Caddy keeps serving the old config if the new
				// one is rejected, so a bad reload is survivable — but only if
				// we notice rather than reporting success.
				`docker exec ferrite-ingress caddy validate --config /etc/caddy/Caddyfile`,
				// Reload if we can, restart if we cannot. Reloading is the
				// gentler option — it swaps the config without dropping a
				// connection — but it goes through the admin API, so it cannot
				// be the way a Caddy that has the admin API switched off gets
				// the config that switches it back on. The fallback is what
				// makes that recoverable rather than permanent.
				`docker exec ferrite-ingress caddy reload --config /etc/caddy/Caddyfile || ` +
					compose(ingressDir, "restart", "caddy"),
			},
		}),
	}
}

// NetworkStep creates the bridge the proxy and applications share.
//
// Separate from IngressSteps and run before anything else, because an
// application's compose file declares the network as external: created after
// the app, compose refuses to start it with "network ferrite declared as
// external, but could not be found".
func NetworkStep() steps.Step {
	return steps.Shell(steps.ShellSpec{
		ID:    "app-network",
		Title: "Create the network applications share",
		Check: "docker network inspect " + network + " >/dev/null 2>&1",
		Apply: []string{"docker network create " + network},
	})
}
