# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS build
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod ./
COPY . .
ARG SERVICE
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/service /app/service
COPY migrations /app/migrations
ENTRYPOINT ["/app/service"]
