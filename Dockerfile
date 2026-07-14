FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mockingbox ./cmd/mockingbox

FROM alpine:3.20
COPY --from=build /mockingbox /usr/local/bin/mockingbox
ENTRYPOINT ["mockingbox"]
CMD ["--help"]
