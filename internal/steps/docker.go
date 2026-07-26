package steps

// Docker is the prerequisite every installable tool shares: a container
// runtime and the compose plugin.
//
// It comes from Docker's own apt repository rather than Ubuntu's. Ubuntu ships
// `docker.io`, which is usually a major version behind and has no compose
// plugin at all — and compose is what every tool in the catalogue is built on,
// so the distribution package would leave us with a runtime we cannot use.
// dockerWorks is the test for "there is already a usable runtime here".
// compose is part of it: the engine alone installs cleanly and every tool
// would then fail one by one on a missing subcommand.
const dockerWorks = `docker --version >/dev/null 2>&1 && docker compose version >/dev/null 2>&1`

func Docker() []Step {
	return []Step{
		shellStep{
			id:    "docker-repo",
			title: "Trust Docker's software source",
			// Nothing to do where a working runtime is already installed,
			// however it got there. Rewriting apt's sources on a machine that
			// is already fine is a change with no benefit, and someone who
			// installed Docker their own way did not ask us to move it.
			skipIf:      dockerWorks,
			skipMessage: "Docker is already installed here, so its software source was left alone.",
			check:       `test -s /etc/apt/keyrings/docker.asc && grep -qs download.docker.com /etc/apt/sources.list.d/docker.list`,
			apply: []string{
				`install -m 0755 -d /etc/apt/keyrings`,
				`curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc`,
				// Readable by _apt, which drops privileges before fetching.
				`chmod a+r /etc/apt/keyrings/docker.asc`,
				// UBUNTU_CODENAME first: on derivatives like Pop!_OS or Linux Mint
				// VERSION_CODENAME is the derivative's own name ("vera"), which
				// Docker publishes nothing for, and apt then fails with a 404 that
				// looks like a network problem.
				`. /etc/os-release && printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu %s stable\n' "$(dpkg --print-architecture)" "${UBUNTU_CODENAME:-$VERSION_CODENAME}" > /etc/apt/sources.list.d/docker.list`,
				`DEBIAN_FRONTEND=noninteractive apt-get update -qq`,
			},
		},
		shellStep{
			id:    "docker-engine",
			title: "Install the container runtime",
			check: dockerWorks,
			apply: []string{
				`DEBIAN_FRONTEND=noninteractive apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`,
			},
		},
		shellStep{
			id:    "docker-running",
			title: "Start the container runtime and keep it started",
			// enabled as well as active: without it nothing comes back after a
			// reboot, and the first anyone hears of it is a service being down.
			check: `systemctl is-active --quiet docker && systemctl is-enabled --quiet docker`,
			apply: []string{`systemctl enable --now docker`},
		},
	}
}
