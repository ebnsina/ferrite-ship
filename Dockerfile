# Build the dashboard first: it changes more often than the Go code, but it
# only needs Node, so keeping the stages separate lets Docker cache each.
FROM node:22-alpine AS web

WORKDIR /src/web
ENV CI=true
RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./

# Baked in at build time, so the image is tied to the origin it will serve
# from. Override with --build-arg when building for a real domain.
ARG PUBLIC_APP_URL="http://localhost:8080"
# Same origin by default: one process serves the dashboard and the API, so the
# browser calls back to where it was loaded from. The build refuses an empty
# value rather than producing a dashboard that points nowhere.
ARG PUBLIC_API_BASE_URL=""
ARG PUBLIC_APP_ENV=production
ARG PUBLIC_DATA_SOURCE=api

# Passed as environment rather than written to .env: Vite reads process.env in
# preference to the file, and an ARG declared with an empty default is present
# in the environment as an empty string — which silently shadowed the file and
# failed the build with "PUBLIC_API_BASE_URL is required".
RUN PUBLIC_API_BASE_URL="${PUBLIC_API_BASE_URL:-$PUBLIC_APP_URL}" \
    PUBLIC_APP_URL="$PUBLIC_APP_URL" \
    PUBLIC_APP_ENV="$PUBLIC_APP_ENV" \
    PUBLIC_DATA_SOURCE="$PUBLIC_DATA_SOURCE" \
    pnpm build

FROM golang:1.25-alpine AS api

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

# CGO off keeps this a static binary — modernc's SQLite driver is pure Go, so
# nothing is lost by it and the final image needs no C library.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ferrite-ship ./cmd/ferrite-ship

FROM alpine:3.21

# ca-certificates for outbound TLS; tzdata so the timezone step and timestamps
# behave. Nothing else — this image opens SSH connections to other people's
# servers and every extra tool in it is a tool an attacker inherits.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 ferrite

COPY --from=api /out/ferrite-ship /usr/local/bin/ferrite-ship
COPY --from=web /src/web/build /srv/web

# The database lives here; mount a volume over it or lose your servers on
# every redeploy.
RUN mkdir -p /data && chown ferrite /data
VOLUME /data

USER ferrite
EXPOSE 8080

ENV FERRITE_ADDR=0.0.0.0:8080 \
    FERRITE_DATABASE_PATH=/data/ferrite.db \
    FERRITE_WEB_DIR=/srv/web \
    FERRITE_ALLOWED_ORIGIN=none

# No FERRITE_SECRET_KEY default on purpose: the process refuses to start
# without one, which is far better than silently using a known key.

ENTRYPOINT ["/usr/local/bin/ferrite-ship"]
