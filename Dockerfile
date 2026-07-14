# mocking-box — single binary (dashboard / collector / verifier).
# libpcap is needed because live NIC sniffing (collect sniff) links it;
# the dashboard and mirror/pcap/verify paths don't use it but share the binary.
FROM golang:1.25-alpine AS build
RUN apk add --no-cache gcc musl-dev libpcap-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /mockingbox ./cmd/mockingbox

FROM alpine:3.20
RUN apk add --no-cache libpcap ca-certificates
COPY --from=build /mockingbox /usr/local/bin/mockingbox
WORKDIR /work
ENTRYPOINT ["mockingbox"]
CMD ["--help"]
