package catalog

// The catalogue itself.
//
// Image tags are pinned to a major line rather than "latest": a server that
// silently changes major version underneath a running application is a bad
// surprise, and "latest" makes every restart a coin toss.
//
// Databases publish on 127.0.0.1 only. They are reached over an SSH tunnel,
// which the dashboard hands you as a copyable command. This is not caution for
// its own sake: an open Postgres port is found by scanners within minutes of
// appearing, and the default account here would be the one we just created.

// traefik is the way in. Every other tool publishes to 127.0.0.1 and is
// reached through it, so this is the only thing here holding 80 and 443.
//
// The Docker socket is mounted read-only. Traefik's whole model is watching
// for containers and configuring itself, and there is no way to do that
// without it — but read-only means it can be told what exists and cannot
// start anything. It is worth being clear-eyed that access to the socket is
// effectively root on the host, which is why nothing else here gets it.
var traefik = Tool{
	ID:       "traefik",
	Name:     "Traefik",
	Summary:  "Gives everything you install a web address of its own, with certificates kept up to date for you.",
	Category: "Networking",
	Icon:     "Route",
	// Traefik's own blue-green, from their mark.
	Accent:      "#24A1C1",
	Image:       "traefik:v3",
	Version:     "3",
	NeedsDomain: true,
	Ports: []Port{
		{Number: 80, Protocol: "tcp", Purpose: "Web traffic, and proving you own the domain", Public: true},
		{Number: 443, Protocol: "tcp", Purpose: "Secure web traffic", Public: true},
	},
	// No Access: there is no connection string, and nothing signs in to it.
	// That also means no password is generated for it — see NeedsPassword.
	Volumes:  []string{"certificates"},
	DataNote: "Removing Traefik stops it and deletes its settings, but keeps the certificates it collected unless you ask for those too. Web addresses stop working until something else answers on them.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-traefik

services:
  traefik:
    image: traefik:v3
    restart: unless-stopped
    command:
      - --providers.docker=true
      # Nothing is routed unless it says so. Without this every container on
      # the network would be published the moment it started, which is how a
      # database ends up on the internet by accident.
      - --providers.docker.exposedbydefault=false
      - --providers.docker.network=ferrite
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      # Anything arriving in the clear is sent straight back as https. The
      # redirect lives here rather than on each tool so a tool cannot forget it.
      - --entrypoints.web.http.redirections.entrypoint.to=websecure
      - --entrypoints.web.http.redirections.entrypoint.scheme=https
      - --certificatesresolvers.le.acme.email=${TRAEFIK_EMAIL:?}
      # Written out rather than left to the default, so which endpoint issued
      # a certificate can be read off the server rather than guessed. Staging
      # certificates are untrusted and browsers warn about them — that is the
      # point of them, and the trade for not being rate limited while getting
      # a new setup working.
      - --certificatesresolvers.le.acme.caserver=${TRAEFIK_ACME_SERVER:?}
      - --certificatesresolvers.le.acme.storage=/certificates/acme.json
      # The HTTP challenge, which needs nothing but port 80 already being
      # open. DNS challenges would allow wildcard certificates, but they need
      # an API key for whoever hosts the domain — a credential we would have
      # to ask for and store to save one certificate per tool.
      - --certificatesresolvers.le.acme.httpchallenge=true
      - --certificatesresolvers.le.acme.httpchallenge.entrypoint=web
      # Answers the healthcheck below. On the container's own interface only —
      # it is not an entrypoint and nothing outside can reach it.
      - --ping=true
    ports:
      - "80:80"
      - "443:443"
    volumes:
      # Read-only: Traefik has to see containers appear, and nothing more.
      - /var/run/docker.sock:/var/run/docker.sock:ro
      # Certificates outlive the container. Without this every restart asks
      # Let's Encrypt for new ones and walks into a rate limit — five per
      # domain per week, which you discover by being locked out for a week.
      - certificates:/certificates
    networks: [ferrite]
    healthcheck:
      test: ["CMD", "traefik", "healthcheck", "--ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    # Traefik does not route to Traefik. Explicit rather than relying on
    # exposedbydefault, so the intent survives a change to that flag.
    labels:
      - traefik.enable=false

volumes:
  certificates:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{
			"TRAEFIK_EMAIL=" + in.Email,
			"TRAEFIK_ACME_SERVER=" + in.ACMEDirectory,
			// Not read by the compose file above — every route is a label on
			// the tool being routed. It is written so that changing the domain
			// changes this file's fingerprint, which is what makes the start
			// step notice and restart rather than reporting nothing to do.
			"FERRITE_DOMAIN=" + in.Domain,
		}
	},
}

// routing renders the two variables every web tool's compose file reads.
//
// A server with no domain gets ROUTED=false, and Traefik then ignores every
// other label on the container — which is how one compose file serves both
// the routed case and the tunnel-only one. The domain is written either way
// so that setting one changes the file's fingerprint and the start step
// notices, rather than reporting nothing to do.
func routing(prefix string, in Install) []string {
	routed := "false"
	if in.Domain != "" {
		routed = "true"
	}
	return []string{
		prefix + "_ROUTED=" + routed,
		"FERRITE_DOMAIN=" + in.Domain,
	}
}

// grafana is the first tool reached through Traefik rather than through a
// tunnel, and the pattern every web tool after it follows.
//
// It works with or without a domain, which is the point. Given one it is
// published at grafana.<domain> with a certificate; without one it stays on
// loopback and is reached exactly as a database is. Requiring a domain would
// have been simpler and would have made installing it impossible on every
// server that works perfectly well today.
var grafana = Tool{
	ID:       "grafana",
	Name:     "Grafana",
	Summary:  "Draws charts from your data, so you can see what is happening rather than read it.",
	Category: "Monitoring",
	Icon:     "ChartLine",
	// Grafana's orange, from their mark.
	Accent:  "#F46800",
	Image:   "grafana/grafana:13.1",
	Version: "13.1",
	Web:     true,
	Ports: []Port{
		// Not public. Traefik reaches it over the shared network, and without a
		// domain it is reached through a tunnel like everything else.
		{Number: 3000, Protocol: "tcp", Purpose: "The Grafana web interface"},
	},
	Access:   &Access{Scheme: "https", Username: "ferrite", Port: 3000},
	Volumes:  []string{"data"},
	DataNote: "Removing Grafana stops it and deletes its settings, but keeps the dashboards you built unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-grafana

services:
  grafana:
    image: grafana/grafana:13.1
    restart: unless-stopped
    environment:
      GF_SECURITY_ADMIN_USER: ferrite
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD:?}
      # Grafana builds its own links from this — sign-in redirects, and the
      # addresses in any alert it sends. Left at the default it emits
      # localhost URLs that work on the server and nowhere else.
      GF_SERVER_ROOT_URL: ${GRAFANA_ROOT_URL:?}
      # Nobody signs up for a Grafana that is on the internet.
      GF_USERS_ALLOW_SIGN_UP: "false"
    ports:
      # Loopback only, exactly like a database. Being routed does not mean
      # being published: Traefik reaches it over the network below.
      - "127.0.0.1:3000:3000"
    volumes:
      - data:/var/lib/grafana
    networks: [ferrite]
    labels:
      # False on a server with no domain, and then every label below it is
      # ignored — which is how this one file serves both cases.
      - traefik.enable=${GRAFANA_ROUTED:?}
      - traefik.http.routers.grafana.rule=Host(` + "`" + `grafana.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.grafana.entrypoints=websecure
      - traefik.http.routers.grafana.tls.certresolver=le
      # Which port to send to. Traefik guesses when a container exposes one
      # port and refuses to guess when it exposes several, so it is always
      # said rather than left to chance.
      - traefik.http.services.grafana.loadbalancer.server.port=3000
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:3000/api/health || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 10

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		// Grafana builds its own links from this, so it is the one web tool
		// that needs its address spelled out rather than just routed to.
		rootURL := "http://localhost:3000/"
		if in.Domain != "" {
			rootURL = "https://" + in.Tool.Subdomain(in.Domain) + "/"
		}
		return append(
			[]string{
				"GRAFANA_PASSWORD=" + in.Password,
				"GRAFANA_ROOT_URL=" + rootURL,
			},
			routing("GRAFANA", in)...,
		)
	},
}

var postgres = Tool{
	ID:       "postgres",
	Name:     "PostgreSQL",
	Summary:  "The database most applications start with. Stores your tables, rows and relationships.",
	Category: "Databases",
	Icon:     "Database",
	// The elephant blue from postgresql.org.
	Accent:  "#336791",
	Image:   "postgres:18-trixie",
	Version: "18",
	Ports: []Port{
		{Number: 5432, Protocol: "tcp", Purpose: "Database connections"},
	},
	Access:   &Access{Scheme: "postgresql", Username: "ferrite", Database: "app", Port: 5432},
	Volumes:  []string{"data"},
	console:  postgresConsole,
	backup:   postgresBackup,
	DataNote: "Removing PostgreSQL stops it and deletes its settings, but keeps your tables and rows unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-postgres

services:
  postgres:
    image: postgres:18-trixie
    restart: unless-stopped
    environment:
      POSTGRES_USER: ferrite
      POSTGRES_DB: app
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?}
    ports:
      # Loopback only: reachable from this server, never from the internet.
      - "127.0.0.1:5432:5432"
    volumes:
      # PostgreSQL 18 moved its data directory to /var/lib/postgresql/18/docker
      # and the image's volume with it. Mounting the old .../data path here
      # would leave the database writing inside the container, and every row
      # would disappear the next time it was recreated.
      - data:/var/lib/postgresql
    shm_size: 256mb
    networks: [ferrite]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ferrite -d app"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  data:

# Shared with every other tool, and created before any of them starts. Joining
# it exposes nothing further: the published port above is still loopback only.
networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{"POSTGRES_PASSWORD=" + in.Password}
	},
}

var redis = Tool{
	ID:       "redis",
	Name:     "Redis",
	Summary:  "Fast temporary storage. Good for caching, queues and sessions.",
	Category: "Caching",
	Icon:     "Zap",
	// Redis red, as of their 2024 mark.
	Accent:  "#FF4438",
	Image:   "redis:8-trixie",
	Version: "8",
	Ports: []Port{
		{Number: 6379, Protocol: "tcp", Purpose: "Cache connections"},
	},
	Access:   &Access{Scheme: "redis", Username: "default", Port: 6379},
	Volumes:  []string{"data"},
	console:  redisConsole,
	backup:   redisBackup,
	DataNote: "Removing Redis stops it and deletes its settings, but keeps what it has saved to disk unless you ask for that too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-redis

services:
  redis:
    image: redis:8-trixie
    restart: unless-stopped
    # appendonly writes every change to disk, so a restart does not empty the
    # cache. Redis defaults to snapshots, which lose the last few minutes.
    command: ["redis-server", "--requirepass", "${REDIS_PASSWORD:?}", "--appendonly", "yes"]
    environment:
      # Repeated here only so the health check below can sign in.
      REDIS_PASSWORD: ${REDIS_PASSWORD:?}
    ports:
      # Loopback only: an open Redis with a password is still an open Redis.
      - "127.0.0.1:6379:6379"
    volumes:
      - data:/data
    networks: [ferrite]
    healthcheck:
      # $$ escapes the dollar so compose leaves it for the container's shell.
      test: ["CMD-SHELL", "redis-cli -a \"$$REDIS_PASSWORD\" ping | grep -q PONG"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{"REDIS_PASSWORD=" + in.Password}
	},
}

var clickhouse = Tool{
	ID:       "clickhouse",
	Name:     "ClickHouse",
	Summary:  "A database built for analytics. Counts and groups very large numbers of rows quickly.",
	Category: "Databases",
	Icon:     "ChartColumn",
	// ClickHouse yellow. Too pale to use flat on white, hence the tinting.
	Accent:  "#FFCC01",
	Image:   "clickhouse:25.8",
	Version: "25.8 (long-term support)",
	Ports: []Port{
		{Number: 8123, Protocol: "tcp", Purpose: "Queries over HTTP"},
		{Number: 9000, Protocol: "tcp", Purpose: "Queries from ClickHouse clients"},
	},
	Access:   &Access{Scheme: "clickhouse", Username: "ferrite", Database: "app", Port: 8123},
	Volumes:  []string{"data", "logs", "backups"},
	console:  clickhouseConsole,
	backup:   clickhouseBackup,
	DataNote: "Removing ClickHouse stops it and deletes its settings, but keeps your tables unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-clickhouse

services:
  clickhouse:
    image: clickhouse:25.8
    restart: unless-stopped
    environment:
      CLICKHOUSE_USER: ferrite
      CLICKHOUSE_DB: app
      CLICKHOUSE_PASSWORD: ${CLICKHOUSE_PASSWORD:?}
      # Without this the account we create cannot grant anything, and the first
      # time you try to add a second user it fails with a permissions error.
      CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: "1"
    ports:
      - "127.0.0.1:8123:8123"
      - "127.0.0.1:9000:9000"
    volumes:
      - data:/var/lib/clickhouse
      - logs:/var/log/clickhouse-server
      # Where a backup is assembled before it is streamed off the server, and
      # where one is put while it is being restored. Its own volume rather
      # than the container's writable layer: a backup of a real database is
      # large, and writing it into the layer makes the container itself large.
      - backups:/backups
    configs:
      - source: backups
        target: /etc/clickhouse-server/config.d/backups.xml
    networks: [ferrite]
    ulimits:
      # ClickHouse opens a file per column part and hits the default limit of
      # 1024 quickly; it logs "too many open files" and stops answering.
      nofile:
        soft: 262144
        hard: 262144
    healthcheck:
      test: ["CMD-SHELL", "clickhouse-client --user ferrite --password \"$$CLICKHOUSE_PASSWORD\" --query 'SELECT 1'"]
      interval: 10s
      timeout: 5s
      retries: 10

# ClickHouse will not back up to a path it has not been told about, so the
# disk is declared here and named again under <backups> as one it may write to.
# Inline rather than a second file on the server: the install writes exactly
# two files and hashes them to decide whether anything changed, and a third
# would be invisible to that check.
configs:
  backups:
    content: |
      <clickhouse>
        <storage_configuration>
          <disks>
            <backups>
              <type>local</type>
              <path>/backups/</path>
            </backups>
          </disks>
        </storage_configuration>
        <backups>
          <allowed_disk>backups</allowed_disk>
          <allowed_path>/backups/</allowed_path>
        </backups>
      </clickhouse>

volumes:
  data:
  logs:
  backups:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{"CLICKHOUSE_PASSWORD=" + in.Password}
	},
}

// meilisearch is a search engine with a browser dashboard, so it follows
// Grafana's pattern rather than a database's: an address and a key, not one
// string with the credential inside it. Its key goes in an Authorization
// header, which no connection string can express anyway.
var meilisearch = Tool{
	ID:       "meilisearch",
	Name:     "Meilisearch",
	Summary:  "Powers the search box in your application, and is forgiving about spelling.",
	Category: "Search",
	Icon:     "Search",
	// Meilisearch's pink-red, from their mark.
	Accent:  "#FF5CAA",
	Image:   "getmeili/meilisearch:v1.50",
	Version: "1.50",
	Web:     true,
	Ports: []Port{
		{Number: 7700, Protocol: "tcp", Purpose: "Searching and adding documents"},
	},
	Access:   &Access{Scheme: "https", Username: "ferrite", Port: 7700},
	Volumes:  []string{"data"},
	DataNote: "Removing Meilisearch stops it and deletes its settings, but keeps what you had indexed unless you ask for that too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-meilisearch

services:
  meilisearch:
    image: getmeili/meilisearch:v1.50
    restart: unless-stopped
    environment:
      # Meilisearch refuses to start in production without a key of at least
      # 16 bytes, which is the same rule the generated password already meets.
      MEILI_MASTER_KEY: ${MEILISEARCH_PASSWORD:?}
      # Production rather than development: development leaves the whole
      # instance open to anyone who can reach it, key or no key.
      MEILI_ENV: production
    ports:
      - "127.0.0.1:7700:7700"
    volumes:
      # Not /data. Meilisearch writes to /meili_data, and a volume on the
      # wrong path leaves it writing inside the container, where the index
      # disappears the next time it is recreated.
      - data:/meili_data
    networks: [ferrite]
    labels:
      - traefik.enable=${MEILISEARCH_ROUTED:?}
      - traefik.http.routers.meilisearch.rule=Host(` + "`" + `meilisearch.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.meilisearch.entrypoints=websecure
      - traefik.http.routers.meilisearch.tls.certresolver=le
      - traefik.http.services.meilisearch.loadbalancer.server.port=7700
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:7700/health || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 10

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return append(
			[]string{"MEILISEARCH_PASSWORD=" + in.Password},
			routing("MEILISEARCH", in)...,
		)
	},
}

// qdrant stores vectors — the representation a model produces — so searching
// it finds things that mean the same rather than things spelled the same.
var qdrant = Tool{
	ID:       "qdrant",
	Name:     "Qdrant",
	Summary:  "Search that understands meaning, so a question finds the right answer even when it shares no words with it.",
	Category: "Search",
	// The four-pointed star, which is how anything model-flavoured is marked
	// here. Never a sparkle or a robot.
	Icon: "Astroid",
	// Qdrant's crimson, from their mark.
	Accent:  "#DC244C",
	Image:   "qdrant/qdrant:v1.18",
	Version: "1.18",
	Web:     true,
	Ports: []Port{
		{Number: 6333, Protocol: "tcp", Purpose: "Searching and storing vectors"},
	},
	Access:   &Access{Scheme: "https", Username: "ferrite", Port: 6333},
	Volumes:  []string{"data"},
	DataNote: "Removing Qdrant stops it and deletes its settings, but keeps the vectors you stored unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-qdrant

services:
  qdrant:
    image: qdrant/qdrant:v1.18
    restart: unless-stopped
    environment:
      # Double underscores are how Qdrant nests configuration in an
      # environment variable. Without this it accepts every request from
      # anyone who can reach it.
      QDRANT__SERVICE__API_KEY: ${QDRANT_PASSWORD:?}
    ports:
      - "127.0.0.1:6333:6333"
    volumes:
      - data:/qdrant/storage
    networks: [ferrite]
    labels:
      - traefik.enable=${QDRANT_ROUTED:?}
      - traefik.http.routers.qdrant.rule=Host(` + "`" + `qdrant.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.qdrant.entrypoints=websecure
      - traefik.http.routers.qdrant.tls.certresolver=le
      - traefik.http.services.qdrant.loadbalancer.server.port=6333
    # No healthcheck. The image carries no shell and no wget or curl, so every
    # form of one fails and reports the container unhealthy while it is
    # serving perfectly well — which would leave compose waiting on it forever.

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return append(
			[]string{"QDRANT_PASSWORD=" + in.Password},
			routing("QDRANT", in)...,
		)
	},
}

// nats passes messages between the parts of an application. Not a web tool:
// it is spoken to by a client, and its address is a connection string in the
// ordinary way.
var nats = Tool{
	ID:       "nats",
	Name:     "NATS",
	Summary:  "Passes messages between parts of your application, and keeps them until something has read them.",
	Category: "Messaging",
	Icon:     "Radio",
	// NATS blue-green, from their mark.
	Accent:  "#27AAE1",
	Image:   "nats:2-alpine",
	Version: "2",
	Ports: []Port{
		{Number: 4222, Protocol: "tcp", Purpose: "Publishing and subscribing"},
	},
	Access:   &Access{Scheme: "nats", Username: "ferrite", Port: 4222},
	Volumes:  []string{"data"},
	DataNote: "Removing NATS stops it and deletes its settings, but keeps any messages it had stored unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-nats

services:
  nats:
    image: nats:2-alpine
    restart: unless-stopped
    # JetStream is what makes a message survive a restart. Without -js this is
    # a relay: anything published while a subscriber is down is simply gone,
    # which is rarely what someone means by a queue.
    command:
      - "-js"
      - "-sd"
      - "/data"
      - "--user"
      - "ferrite"
      - "--pass"
      - "${NATS_PASSWORD:?}"
      - "-m"
      - "8222"
    ports:
      - "127.0.0.1:4222:4222"
    volumes:
      - data:/data
    networks: [ferrite]
    healthcheck:
      # The monitoring port, which is bound inside the container only — it is
      # not in ports above, so nothing outside can reach it.
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8222/healthz || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{"NATS_PASSWORD=" + in.Password}
	},
}

// rabbitmq is the first tool with two faces: clients speak AMQP on 5672, and
// there is a management page on 15672. Both belong to the same tool, so it is
// one entry with a WebPort rather than two.
var rabbitmq = Tool{
	ID:       "rabbitmq",
	Name:     "RabbitMQ",
	Summary:  "Queues work to be done later, so a slow job never keeps somebody waiting.",
	Category: "Messaging",
	Icon:     "Layers",
	// RabbitMQ orange, from their mark.
	Accent:  "#FF6600",
	Image:   "rabbitmq:4-management",
	Version: "4",
	Web:     true,
	WebPort: 15672,
	Ports: []Port{
		{Number: 5672, Protocol: "tcp", Purpose: "Publishing and consuming"},
		{Number: 15672, Protocol: "tcp", Purpose: "The management page"},
	},
	Access:   &Access{Scheme: "amqp", Username: "ferrite", Port: 5672},
	Volumes:  []string{"data"},
	DataNote: "Removing RabbitMQ stops it and deletes its settings, but keeps any queued messages unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-rabbitmq

services:
  rabbitmq:
    # The -management tag, because the plain image has no web page at all and
    # adding the plugin afterwards means a custom image to maintain.
    image: rabbitmq:4-management
    restart: unless-stopped
    environment:
      RABBITMQ_DEFAULT_USER: ferrite
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD:?}
    ports:
      - "127.0.0.1:5672:5672"
      - "127.0.0.1:15672:15672"
    volumes:
      - data:/var/lib/rabbitmq
    networks: [ferrite]
    labels:
      - traefik.enable=${RABBITMQ_ROUTED:?}
      - traefik.http.routers.rabbitmq.rule=Host(` + "`" + `rabbitmq.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.rabbitmq.entrypoints=websecure
      - traefik.http.routers.rabbitmq.tls.certresolver=le
      # The management page, not the port clients use. Routing 5672 would put
      # a binary protocol behind an HTTP proxy, which fails in a way that
      # looks like the queue is broken.
      - traefik.http.services.rabbitmq.loadbalancer.server.port=15672
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 15s
      timeout: 10s
      # Slower than the others on purpose: RabbitMQ takes a while to come up,
      # and a tighter window reports a healthy broker as failed.
      retries: 12
      start_period: 30s

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return append(
			[]string{"RABBITMQ_PASSWORD=" + in.Password},
			routing("RABBITMQ", in)...,
		)
	},
}

// minio stores files the way S3 does, so anything written against S3 works
// against it unchanged.
//
// Worth knowing before choosing it: the community edition's administrative
// interface was removed in RELEASE.2025-05-24, leaving an object browser, and
// the newest community tag is from September 2025. The storage itself is
// unaffected and the API is the part applications use, but managing it means
// the mc command line rather than a page.
var minio = Tool{
	ID:       "minio",
	Name:     "MinIO",
	Summary:  "Holds files and uploads, and speaks the same language as S3 so existing code works against it.",
	Category: "Storage",
	Icon:     "HardDrive",
	// MinIO red, from their mark.
	Accent: "#C72E49",
	// A dated release rather than a major line: MinIO publishes no moving tag
	// but "latest", and latest is how a server changes underneath you.
	Image:   "minio/minio:RELEASE.2025-09-07T16-13-09Z",
	Version: "2025-09-07",
	Web:     true,
	WebPort: 9001,
	Ports: []Port{
		{Number: 9000, Protocol: "tcp", Purpose: "Reading and writing files"},
		{Number: 9001, Protocol: "tcp", Purpose: "The file browser"},
	},
	Access:   &Access{Scheme: "s3", Username: "ferrite", Port: 9000},
	Volumes:  []string{"data"},
	DataNote: "Removing MinIO stops it and deletes its settings, but keeps the files you uploaded unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-minio

services:
  minio:
    image: minio/minio:RELEASE.2025-09-07T16-13-09Z
    restart: unless-stopped
    # The console port has to be fixed. Left unset MinIO picks a new one on
    # every start, and the address in the dashboard is then right only until
    # the container restarts.
    command: ["server", "/data", "--console-address", ":9001"]
    environment:
      MINIO_ROOT_USER: ferrite
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD:?}
    ports:
      - "127.0.0.1:9000:9000"
      - "127.0.0.1:9001:9001"
    volumes:
      - data:/data
    networks: [ferrite]
    labels:
      - traefik.enable=${MINIO_ROUTED:?}
      - traefik.http.routers.minio.rule=Host(` + "`" + `minio.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.minio.entrypoints=websecure
      - traefik.http.routers.minio.tls.certresolver=le
      # The browser, not the S3 API. Uploads go to the API port through the
      # tunnel or from the server itself.
      - traefik.http.services.minio.loadbalancer.server.port=9001
    healthcheck:
      test: ["CMD-SHELL", "mc ready local || exit 1"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 20s

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return append(
			[]string{"MINIO_PASSWORD=" + in.Password},
			routing("MINIO", in)...,
		)
	},
}

// keycloak is the first tool made of more than one container: it needs a
// database, and it brings its own rather than sharing the catalogue's.
//
// That is a deliberate reversal of "Keycloak is blocked until tools can depend
// on tools". Keycloak's database is not a database somebody uses — it is
// Keycloak's private storage, whose schema Keycloak owns and migrates on every
// upgrade. Putting that inside the PostgreSQL somebody keeps their own tables
// in would couple two upgrade cycles that have no business being coupled, and
// the "shared dependency" it would demonstrate is one nobody should want. An
// application needing a DATABASE_URL is the case that genuinely wants sharing,
// and it is a different problem.
//
// The database is deliberately NOT on the shared network. Everything on that
// network can reach everything else on it, so a bundled dependency that joined
// it would be reachable by every other tool on the server — and it would be
// competing for a service name in a namespace shared by the whole catalogue.
// It sits on this project's own default network, where the only thing that can
// see it is the Keycloak beside it.
var keycloak = Tool{
	ID:       "keycloak",
	Name:     "Keycloak",
	Summary:  "Handles signing in, so your application never stores a password itself.",
	Category: "Identity",
	Icon:     "KeyRound",
	// Keycloak's blue, from their mark.
	Accent:  "#4D4D4D",
	Image:   "quay.io/keycloak/keycloak:26.7",
	Version: "26.7",
	Web:     true,
	// The first tool that genuinely cannot work without one. Keycloak builds
	// every sign-in redirect and every token issuer from its hostname, so a
	// wrong or missing one produces an installation that starts cleanly and
	// fails the moment anybody tries to use it.
	NeedsDomain: true,
	Ports: []Port{
		{Number: 8080, Protocol: "tcp", Purpose: "The sign-in pages and the admin console"},
	},
	Access:   &Access{Scheme: "https", Username: "ferrite", Port: 8080},
	Volumes:  []string{"data"},
	DataNote: "Removing Keycloak stops it and deletes its settings, but keeps its accounts and realms unless you ask for those too.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-keycloak

services:
  keycloak:
    image: quay.io/keycloak/keycloak:26.7
    restart: unless-stopped
    # Plain start rather than --optimized: that flag requires a build step to
    # have been run into the image first, and without it Keycloak refuses to
    # start at all rather than falling back.
    command: ["start"]
    environment:
      KC_BOOTSTRAP_ADMIN_USERNAME: ferrite
      KC_BOOTSTRAP_ADMIN_PASSWORD: ${KEYCLOAK_PASSWORD:?}

      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://keycloak-db:5432/keycloak
      KC_DB_USERNAME: keycloak
      KC_DB_PASSWORD: ${KEYCLOAK_PASSWORD:?}

      # The full public URL, not a bare host. Keycloak builds sign-in
      # redirects and token issuers from it, and every client that trusts this
      # server compares the issuer exactly — so this being wrong looks like
      # working sign-ins that every application then rejects.
      KC_HOSTNAME: ${KEYCLOAK_URL:?}
      # TLS is terminated at Traefik, so Keycloak itself speaks plain HTTP on
      # the private network. Without this it refuses to serve anything.
      KC_HTTP_ENABLED: "true"
      # Take the scheme and port from the proxy's headers. Without it Keycloak
      # sees an http request and writes http:// into redirects it has just
      # been reached over https, which browsers then block.
      KC_PROXY_HEADERS: xforwarded
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      keycloak-db:
        condition: service_healthy
    # Both networks: the shared one so Traefik can route to it, and this
    # project's own so it can reach its database.
    networks: [ferrite, default]
    labels:
      - traefik.enable=${KEYCLOAK_ROUTED:?}
      - traefik.http.routers.keycloak.rule=Host(` + "`" + `keycloak.${FERRITE_DOMAIN}` + "`" + `)
      - traefik.http.routers.keycloak.entrypoints=websecure
      - traefik.http.routers.keycloak.tls.certresolver=le
      - traefik.http.services.keycloak.loadbalancer.server.port=8080
    # No healthcheck. Keycloak's own is on a separate management port and the
    # image carries no curl or wget to ask it with, so every version of one
    # reports a working server as unhealthy.

  keycloak-db:
    image: postgres:18-trixie
    restart: unless-stopped
    environment:
      POSTGRES_USER: keycloak
      POSTGRES_DB: keycloak
      POSTGRES_PASSWORD: ${KEYCLOAK_PASSWORD:?}
    volumes:
      # PostgreSQL 18 keeps its data under /var/lib/postgresql/18/docker, so
      # the volume goes one level up. The familiar .../data path would leave
      # every account written inside the container.
      - data:/var/lib/postgresql
    # No networks, so it joins this project's default network and nothing
    # else. Not published either: only the container above can reach it.
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U keycloak -d keycloak"]
      interval: 10s
      timeout: 5s
      retries: 10

volumes:
  data:

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return append(
			[]string{
				"KEYCLOAK_PASSWORD=" + in.Password,
				"KEYCLOAK_URL=https://" + in.Tool.Subdomain(in.Domain) + "/",
			},
			routing("KEYCLOAK", in)...,
		)
	},
}

var mediamtx = Tool{
	ID:       "mediamtx",
	Name:     "MediaMTX",
	Summary:  "Receives live video and hands it out again, so you can publish a camera or a stream and watch it in a browser.",
	Category: "Media",
	Icon:     "Video",
	// Sampled from MediaMTX's own logo, which is drawn in two blues. The
	// teal is the one that does not read as PostgreSQL's navy.
	Accent:  "#1C94B5",
	Image:   "bluenviron/mediamtx:1",
	Version: "1",
	// The only tool here meant to be reached from the internet — a stream
	// nobody outside can watch is not a stream.
	Ports: []Port{
		{Number: 8554, Protocol: "tcp", Purpose: "Publish or watch over RTSP", Public: true},
		{Number: 1935, Protocol: "tcp", Purpose: "Publish over RTMP", Public: true},
		{Number: 8888, Protocol: "tcp", Purpose: "Watch in a browser over HLS", Public: true},
		{Number: 8889, Protocol: "tcp", Purpose: "Watch in a browser over WebRTC", Public: true},
		{Number: 8890, Protocol: "udp", Purpose: "Publish or watch over SRT", Public: true},
		// Without this open, WebRTC negotiates and then silently never connects:
		// the media itself travels over UDP 8189 even though the page loaded.
		{Number: 8189, Protocol: "udp", Purpose: "Carries WebRTC video once a viewer connects", Public: true},
	},
	Access:       &Access{Scheme: "rtsp", Username: "ferrite", Port: 8554},
	NeedsAddress: true,
	DataNote:     "Removing MediaMTX stops it and deletes its settings. It does not keep recordings, so there is nothing else to delete.",
	compose: `# Managed by Ferrite Ship. Edits are replaced the next time this tool is set up.
name: ferrite-mediamtx

services:
  mediamtx:
    image: bluenviron/mediamtx:1
    restart: unless-stopped
    environment:
      # UDP is dropped or reordered by plenty of networks; TCP is the transport
      # that works from behind an office firewall.
      MTX_RTSPTRANSPORTS: tcp
      # WebRTC tells the viewer's browser which address to send media to. The
      # container only knows its own private address, so without this the page
      # loads and the video never starts.
      MTX_WEBRTCADDITIONALHOSTS: ${MEDIAMTX_ADDRESS:?}

      # MediaMTX ships allowing anyone to publish. On a public port that means
      # a stranger can overwrite your stream, so publishing takes a password
      # and only watching is open.
      MTX_AUTHINTERNALUSERS_0_USER: ferrite
      MTX_AUTHINTERNALUSERS_0_PASS: ${MEDIAMTX_PASSWORD:?}
      MTX_AUTHINTERNALUSERS_0_PERMISSIONS_0_ACTION: publish

      MTX_AUTHINTERNALUSERS_1_USER: any
      MTX_AUTHINTERNALUSERS_1_PERMISSIONS_0_ACTION: read
      MTX_AUTHINTERNALUSERS_1_PERMISSIONS_1_ACTION: playback

      # Its own control interface, from this server only.
      MTX_AUTHINTERNALUSERS_2_USER: any
      MTX_AUTHINTERNALUSERS_2_IPS: 127.0.0.1,::1
      MTX_AUTHINTERNALUSERS_2_PERMISSIONS_0_ACTION: api
      MTX_AUTHINTERNALUSERS_2_PERMISSIONS_1_ACTION: metrics
    ports:
      - "8554:8554"
      - "1935:1935"
      - "8888:8888"
      - "8889:8889"
      - "8890:8890/udp"
      - "8189:8189/udp"
    networks: [ferrite]

networks:
  ferrite:
    external: true
`,
	env: func(in Install) []string {
		return []string{
			"MEDIAMTX_PASSWORD=" + in.Password,
			"MEDIAMTX_ADDRESS=" + in.Address,
		}
	},
}
