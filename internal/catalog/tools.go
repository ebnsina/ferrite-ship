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
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ferrite -d app"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  data:
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
    healthcheck:
      # $$ escapes the dollar so compose leaves it for the container's shell.
      test: ["CMD-SHELL", "redis-cli -a \"$$REDIS_PASSWORD\" ping | grep -q PONG"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  data:
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
	Volumes:  []string{"data", "logs"},
	console:  clickhouseConsole,
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

volumes:
  data:
  logs:
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
`,
	env: func(in Install) []string {
		return []string{
			"MEDIAMTX_PASSWORD=" + in.Password,
			"MEDIAMTX_ADDRESS=" + in.Address,
		}
	},
}
