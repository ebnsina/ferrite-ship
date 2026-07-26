package steps

import "strings"

// BaselineOptions parameterises the first-run playbook.
type BaselineOptions struct {
	// AdminUser is the non-root account created for day-to-day access.
	AdminUser string
	// PublicKey, when set, is installed for AdminUser. Without a key we refuse
	// to turn off password logins — see the ssh-harden step.
	PublicKey string
	// Timezone in IANA form.
	Timezone string
	// SwapSizeGB is skipped when zero or when swap already exists.
	SwapSizeGB int
	// OpenPorts are the TCP ports the firewall should allow, besides SSH.
	OpenPorts []int
}

func (o BaselineOptions) withDefaults() BaselineOptions {
	if o.AdminUser == "" {
		o.AdminUser = "deploy"
	}
	if o.Timezone == "" {
		o.Timezone = "UTC"
	}
	if o.SwapSizeGB == 0 {
		o.SwapSizeGB = 2
	}
	if o.OpenPorts == nil {
		o.OpenPorts = []int{80, 443}
	}
	return o
}

const basePackages = "ufw fail2ban unattended-upgrades ca-certificates curl"

// Baseline returns the first-run playbook: update, harden, firewall, and the
// housekeeping every server wants. Every step is safe to run repeatedly.
func Baseline(opts BaselineOptions) []Step {
	o := opts.withDefaults()

	steps := []Step{
		shellStep{
			id:    "apt-update",
			title: "Refresh the list of available updates",
			// Skip if the lists were refreshed in the last day; apt-get update
			// is slow and pointless to repeat on every run.
			check: `find /var/lib/apt/lists -maxdepth 1 -name '*Release' -newermt '-24 hours' 2>/dev/null | grep -q .`,
			apply: []string{`DEBIAN_FRONTEND=noninteractive apt-get update -qq`},
		},
		shellStep{
			id:    "base-packages",
			title: "Install the essentials",
			check: `dpkg -s ` + basePackages + ` >/dev/null 2>&1`,
			apply: []string{
				`DEBIAN_FRONTEND=noninteractive apt-get install -y -qq ` + basePackages,
			},
		},
		shellStep{
			id:    "admin-user",
			title: "Create your everyday login account",
			check: `id -u ` + o.AdminUser + ` >/dev/null 2>&1`,
			apply: []string{
				`useradd --create-home --shell /bin/bash ` + o.AdminUser,
				`usermod -aG sudo ` + o.AdminUser,
			},
		},
	}

	if o.PublicKey != "" {
		home := "/home/" + o.AdminUser
		keyFile := home + "/.ssh/authorized_keys"
		steps = append(steps, shellStep{
			id:    "admin-key",
			title: "Install your login key",
			check: `grep -qxF ` + shellQuote(o.PublicKey) + ` ` + keyFile + ` 2>/dev/null`,
			apply: []string{
				`install -d -m 700 -o ` + o.AdminUser + ` -g ` + o.AdminUser + ` ` + home + `/.ssh`,
				`printf '%s\n' ` + shellQuote(o.PublicKey) + ` >> ` + keyFile,
				`chown ` + o.AdminUser + `:` + o.AdminUser + ` ` + keyFile,
				`chmod 600 ` + keyFile,
			},
		})
	}

	steps = append(steps,
		shellStep{
			id:    "ssh-harden",
			title: "Turn off root and password logins",
			// Refuse to disable password logins when no key exists anywhere —
			// doing so would lock the owner out of their own machine, and an
			// unrecoverable server is far worse than a slightly weaker one.
			skipIf: `! grep -rqs . /root/.ssh/authorized_keys /home/*/.ssh/authorized_keys`,
			skipMessage: "Left password logins on: no login key is installed yet, " +
				"and turning them off now would lock you out.",
			check: `sshd -T 2>/dev/null | grep -qix 'permitrootlogin no' && ` +
				`sshd -T 2>/dev/null | grep -qix 'passwordauthentication no'`,
			apply: []string{
				`install -d -m 755 /etc/ssh/sshd_config.d`,
				`printf 'PermitRootLogin no\nPasswordAuthentication no\nChallengeResponseAuthentication no\nKbdInteractiveAuthentication no\n' > /etc/ssh/sshd_config.d/10-ferrite.conf`,
				// Validate before reloading: a bad config plus a reload is a
				// locked door with the key inside.
				`sshd -t`,
				`systemctl reload ssh 2>/dev/null || systemctl reload sshd`,
			},
		},
		shellStep{
			id:    "firewall",
			title: "Close every door except the ones you use",
			check: `ufw status 2>/dev/null | grep -q '^Status: active'`,
			apply: append(
				[]string{
					`ufw --force default deny incoming`,
					`ufw --force default allow outgoing`,
					// The OpenSSH profile only covers port 22. Ask sshd which
					// port it is actually on and allow that too — enabling the
					// firewall without this cuts off a server running SSH
					// anywhere else, including the connection doing the work.
					`SSH_PORT="$(sshd -T 2>/dev/null | awk '/^port /{print $2}' | head -n1)"; ufw allow "${SSH_PORT:-22}"/tcp`,
					// Tolerated: the profile is absent on some minimal images,
					// and the explicit port rule above already covers us.
					`ufw allow OpenSSH || true`,
				},
				append(allowPortCommands(o.OpenPorts), `ufw --force enable`)...,
			),
		},
		shellStep{
			id:    "fail2ban",
			title: "Block repeated login attempts",
			check: `systemctl is-active --quiet fail2ban`,
			apply: []string{`systemctl enable --now fail2ban`},
		},
		shellStep{
			id:    "auto-updates",
			title: "Install security updates automatically",
			check: `grep -qs '"1"' /etc/apt/apt.conf.d/20auto-upgrades`,
			apply: []string{
				`printf 'APT::Periodic::Update-Package-Lists "1";\nAPT::Periodic::Unattended-Upgrade "1";\n' > /etc/apt/apt.conf.d/20auto-upgrades`,
				`systemctl enable --now unattended-upgrades`,
			},
		},
		shellStep{
			id:    "timezone",
			title: "Set the clock to " + o.Timezone,
			check: `test "$(timedatectl show -p Timezone --value 2>/dev/null)" = ` + shellQuote(o.Timezone),
			apply: []string{`timedatectl set-timezone ` + shellQuote(o.Timezone)},
		},
		shellStep{
			id:    "swap",
			title: "Add breathing room when memory runs short",
			// No skipIf: swap already existing is "already fine", not "not
			// needed here", and the check below says so with the right word.
			check: `swapon --show --noheadings 2>/dev/null | grep -q .`,
			apply: []string{
				`fallocate -l ` + itoa(o.SwapSizeGB) + `G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=` + itoa(o.SwapSizeGB*1024),
				`chmod 600 /swapfile`,
				`mkswap /swapfile`,
				`swapon /swapfile`,
				`grep -q '^/swapfile ' /etc/fstab || printf '/swapfile none swap sw 0 0\n' >> /etc/fstab`,
			},
		},
		shellStep{
			id:    "sysctl",
			title: "Apply sensible network settings",
			check: `test "$(sysctl -n net.ipv4.tcp_syncookies 2>/dev/null)" = "1"`,
			apply: []string{
				`printf 'net.ipv4.tcp_syncookies = 1\nnet.ipv4.conf.all.rp_filter = 1\nnet.ipv4.conf.all.accept_redirects = 0\nnet.ipv6.conf.all.accept_redirects = 0\nnet.ipv4.conf.all.accept_source_route = 0\nfs.file-max = 200000\n' > /etc/sysctl.d/60-ferrite.conf`,
				`sysctl --system >/dev/null`,
			},
		},
	)

	return steps
}

func allowPortCommands(ports []int) []string {
	cmds := make([]string, 0, len(ports))
	for _, port := range ports {
		cmds = append(cmds, `ufw allow `+itoa(port)+`/tcp`)
	}
	return cmds
}

// shellQuote wraps s in single quotes, escaping any it contains, so untrusted
// values (a public key, a timezone from the API) cannot break out of the
// command they are embedded in.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
