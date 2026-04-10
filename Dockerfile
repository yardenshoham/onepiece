FROM golang:1.26-alpine AS builder

WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

# Install CA certificates so we can copy the trust store into the scratch image.
RUN apk add --no-cache ca-certificates

COPY . .

ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -trimpath -ldflags="-s -w" -o /out/onepiece .

FROM scratch

COPY --from=builder /out/onepiece /onepiece
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

ENTRYPOINT ["/onepiece", "web"]