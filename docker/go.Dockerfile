# Go toolchain image used by every Go-related make target.
# CGO stays enabled here because `go test -race` needs it; release builds set CGO_ENABLED=0.
FROM golang:1.27.4-bookworm
WORKDIR /src
