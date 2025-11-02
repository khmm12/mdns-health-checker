FROM --platform=$BUILDPLATFORM golang:1.25.3 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN \
  GOOS=${TARGETOS} \
  GOARCH=${TARGETARCH} \
  GOARM=${TARGETVARIANT#v} \
  CGO_ENABLED=0 \
  go build -ldflags "-s -w" -o ./mdns-health-checker ./cmd/mdns-health-checker

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/mdns-health-checker /
ENTRYPOINT ["/mdns-health-checker"]
