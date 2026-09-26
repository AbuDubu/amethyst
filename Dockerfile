# Production image: one Amethyst server (Go binary + built frontend).
# Build from the repository root:
#   docker build --build-arg VERSION=$(git describe --tags --always --dirty) -t amethyst .

# --- Frontend ---
FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Server ---
FROM golang:1.27-alpine AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
ARG VERSION=dev
# CGO off: a fully static binary that runs on the minimal base image below.
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/amethyst ./cmd/amethyst

# --- Runtime ---
# Distroless: no shell or package manager, runs as an unprivileged user.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=server /out/amethyst /amethyst
COPY --from=web /src/web/dist /web
ENV AMETHYST_WEB_DIR=/web
EXPOSE 8080
ENTRYPOINT ["/amethyst"]
CMD ["serve"]
