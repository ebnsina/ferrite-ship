package deploy

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// appCompose describes how one application runs.
//
// No published ports: the proxy reaches it over the shared network by name,
// so an application is never exposed directly and cannot collide with another
// over a port number.
func appCompose(a App) string {
	return `# Managed by Ferrite Ship. Edits are replaced on the next deploy.
name: ferrite-app-` + a.ID + `

services:
  app:
    image: ` + Image(a.ID) + `
    container_name: ` + Container(a.ID) + `
    restart: unless-stopped
    env_file:
      - .env
    expose:
      - "` + strconv.Itoa(a.Port) + `"
    networks:
      - ferrite

networks:
  ferrite:
    external: true
`
}

// envFile renders the variables, sorted so the file only changes when its
// contents do rather than on every map iteration.
func envFile(a App) string {
	keys := make([]string, 0, len(a.Env))
	for key := range a.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var out strings.Builder
	out.WriteString("PORT=" + strconv.Itoa(a.Port) + "\n")
	for _, key := range keys {
		out.WriteString(key + "=" + a.Env[key] + "\n")
	}
	return out.String()
}

// BuilderSteps installs what is needed to turn source into an image.
//
// Nixpacks is only fetched when it will be used — a repository with its own
// Dockerfile needs nothing beyond Docker, and downloading a build tool for it
// would be work with no purpose.
func BuilderSteps() []steps.Step {
	return []steps.Step{
		steps.Shell(steps.ShellSpec{
			ID:    "nixpacks",
			Title: "Install the builder that reads your project",
			Check: `nixpacks --version 2>/dev/null | grep -q ` + steps.Quote(nixpacksVersion),
			Apply: []string{
				// Architecture is read from dpkg rather than assumed: these
				// servers are as often arm64 as amd64 now, and the wrong
				// package installs and then refuses to execute.
				`ARCH="$(dpkg --print-architecture)"; ` +
					`curl -fsSL -o /tmp/nixpacks.deb ` +
					`"https://github.com/railwayapp/nixpacks/releases/download/v` + nixpacksVersion +
					`/nixpacks-v` + nixpacksVersion + `-${ARCH}.deb"`,
				`DEBIAN_FRONTEND=noninteractive dpkg -i /tmp/nixpacks.deb`,
				`rm -f /tmp/nixpacks.deb`,
			},
		}),
	}
}

// AppSteps fetches the source, builds it and runs it.
func AppSteps(a App) []steps.Step {
	dir := appDir(a.ID)
	src := sourceDir(a.ID)

	branch := a.Branch
	if branch == "" {
		branch = "main"
	}

	playbook := []steps.Step{}

	if a.DeployKey != "" {
		key := strings.TrimRight(a.DeployKey, "\n") + "\n"
		keyFile := dir + "/deploy_key"

		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:    "app-key",
			Title: "Install the key that reads " + a.Name + "'s repository",
			Check: matches(keyFile, key),
			Apply: append(
				[]string{"install -d -m 700 " + steps.Quote(dir)},
				// 600 and nothing looser: ssh refuses to use a key file that
				// others can read, and the error it gives points at permissions
				// rather than at the key, which sends people the wrong way.
				write(keyFile, key, "600")...,
			),
		}))
	}

	return append(playbook,
		steps.Shell(steps.ShellSpec{
			ID:    "app-source",
			Title: "Fetch the latest of " + a.Name,
			// No check: fetching is the point of deploying, and a check that
			// compared commits would skip the build someone just asked for.
			Apply: []string{
				"install -d -m 755 " + steps.Quote(dir),
				// Clone once, then fetch. --depth 1 because deploying does not
				// need the history, and a large repository's history is most of
				// its size.
				gitEnv(a, dir) +
					"if [ -d " + steps.Quote(src+"/.git") + " ]; then " +
					"git -C " + steps.Quote(src) + " fetch --depth 1 origin " + steps.Quote(branch) +
					" && git -C " + steps.Quote(src) + " reset --hard FETCH_HEAD" +
					"; else " +
					"rm -rf " + steps.Quote(src) + " && git clone --depth 1 --branch " + steps.Quote(branch) +
					" " + steps.Quote(a.Repository) + " " + steps.Quote(src) +
					"; fi",
			},
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "app-build",
			Title: "Build " + a.Name,
			Apply: []string{
				// Attestations off.
				//
				// Docker 29 defaults to the containerd image store and buildx
				// attaches a provenance attestation to every build. Together
				// they fail at the very end — after a successful compile — with
				// "parent snapshot does not exist", which reads like a broken
				// project rather than a broken export. Nothing here needs a
				// signed build record.
				"export BUILDX_NO_DEFAULT_ATTESTATIONS=1; " +
					// A Dockerfile in the repository wins. It is the author
					// saying exactly how this is built, and guessing over the
					// top of that would be presumptuous and usually wrong.
					"if [ -f " + steps.Quote(src+"/Dockerfile") + " ]; then " +
					"docker build -t " + steps.Quote(Image(a.ID)) + " " + steps.Quote(src) +
					"; else " +
					"nixpacks build " + steps.Quote(src) + " --name " + steps.Quote(Image(a.ID)) +
					"; fi",
			},
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "app-config",
			Title: "Write " + a.Name + "'s settings",
			Check: matches(dir+"/compose.yaml", appCompose(a)) + " && " + matches(dir+"/.env", envFile(a)),
			Apply: append(
				write(dir+"/compose.yaml", appCompose(a), "644"),
				// 600: this holds database passwords and API keys far more
				// often than not.
				write(dir+"/.env", envFile(a), "600")...,
			),
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "app-up",
			Title: "Start " + a.Name,
			// Always recreate: the image tag is unchanged but its contents are
			// new, and compose would otherwise leave the previous build running
			// and report that everything is up to date.
			Apply: []string{
				compose(dir, "up", "-d", "--force-recreate", "--wait"),
			},
		}),
	)
}

// gitEnv prefixes a git command with the key to use, when there is one.
//
// StrictHostKeyChecking=accept-new rather than off: the host's identity is
// recorded on first contact and checked afterwards, which is the same bargain
// Ferrite Ship makes with the servers it manages. Turning it off entirely
// would accept any impostor for the life of the deployment.
func gitEnv(a App, dir string) string {
	if a.DeployKey == "" {
		return ""
	}
	return "export GIT_SSH_COMMAND=" + steps.Quote(
		"ssh -i "+dir+"/deploy_key -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new",
	) + "; "
}

// RemoveSteps stops an application and takes its files away.
func RemoveSteps(a App) []steps.Step {
	dir := appDir(a.ID)

	return []steps.Step{
		steps.Shell(steps.ShellSpec{
			ID:          "app-remove",
			Title:       "Stop " + a.Name + " and remove it",
			SkipIf:      "! test -d " + steps.Quote(dir),
			SkipMessage: a.Name + " is not on this server, so there was nothing to remove.",
			Apply: []string{
				compose(dir, "down", "--remove-orphans"),
				"docker image rm -f " + steps.Quote(Image(a.ID)) + " || true",
				"rm -rf " + steps.Quote(dir),
			},
		}),
	}
}
