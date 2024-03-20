FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS base

# devcontainer
FROM base AS gopls
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
        GOBIN=/build/ GO111MODULE=on go install "golang.org/x/tools/gopls@latest" \
     && /build/gopls version

FROM base AS devcontainer
COPY --from=gopls /build/gopls /usr/local/bin/gopls

# build
FROM base AS build

WORKDIR /go/src

COPY go.mod go.sum ./
RUN --mount=type=bind,src=./go.mod,target=./go.mod \
    --mount=type=bind,src=./go.sum,target=./go.sum \
    go mod download

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=bind,target=/go/src \
    --mount=type=cache,target=/root/.cache/go-build \
        CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin ./...

# binary
FROM scratch AS binary
COPY --from=build /go/bin/* /

# run
FROM gcr.io/distroless/static-debian12 AS run

WORKDIR /

COPY --from=binary /zte /app
USER nonroot:nonroot

ENTRYPOINT ["/app"]
