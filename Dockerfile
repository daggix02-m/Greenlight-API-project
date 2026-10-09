FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 go build -mod=vendor -trimpath \
    -ldflags="-s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api

EXPOSE 4000

USER nonroot:nonroot

ENTRYPOINT ["/api"]
