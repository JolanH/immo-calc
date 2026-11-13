# Go toolchain image used by every Go-related make target.
# CGO stays enabled here because `go test -race` needs it; release builds set CGO_ENABLED=0.
FROM golang:1.27.4-bookworm
# Delve, for `make debug` only. It never ships in the app binary.
RUN CGO_ENABLED=0 go install github.com/go-delve/delve/cmd/dlv@v1.27.2 \
	&& rm -rf /root/.cache /go/pkg
WORKDIR /src
