# Build the ridectl binary
FROM golang:1 as builder

# Copy in the go src
COPY . /go/src/github.com/Ridecell/ridectl
WORKDIR /go/src/github.com/Ridecell/ridectl

# Build
RUN make dep generate && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ridectl -tags release github.com/Ridecell/ridectl/cmd/ridectl

# Copy the controller-manager into a thin image
FROM alpine:latest
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /go/src/github.com/Ridecell/ridectl/ridectl /ridectl
