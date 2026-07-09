FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM builder AS build-tester
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/anthropic-compatibility-tester ./cmd/anthropic-compatibility-tester

FROM builder AS build-mockserver
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mockserver ./cmd/mockserver

FROM gcr.io/distroless/static-debian12:nonroot AS mockserver

COPY --from=build-mockserver /out/mockserver /usr/local/bin/mockserver

ENTRYPOINT ["/usr/local/bin/mockserver"]

FROM gcr.io/distroless/static-debian12:nonroot AS tester

COPY --from=build-tester /out/anthropic-compatibility-tester /usr/local/bin/anthropic-compatibility-tester

ENTRYPOINT ["/usr/local/bin/anthropic-compatibility-tester"]