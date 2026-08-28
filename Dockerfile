FROM golang:1.27.0-alpine AS build

WORKDIR /source
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=action" -o /out/lockreview ./cmd/lockreview

FROM alpine:3.22

RUN apk add --no-cache ca-certificates git
COPY --from=build /out/lockreview /usr/local/bin/lockreview
ENTRYPOINT ["/usr/local/bin/lockreview"]
