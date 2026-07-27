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
			// Not read by the compose file above — every route is a label on
			// the tool being routed. It is written so that changing the domain
			// changes this file's fingerprint, which is what makes the start
			// step notice and restart rather than reporting nothing to do.
			"FERRITE_DOMAIN=" + in.Domain,
		}
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
