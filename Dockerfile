# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/reduta-server ./cmd/reduta-server
RUN CGO_ENABLED=0 go build -trimpath -o /out/reduta-worker ./cmd/reduta-worker
RUN CGO_ENABLED=0 go build -trimpath -o /out/reduta-cli ./cmd/reduta-cli
RUN CGO_ENABLED=0 go build -trimpath -o /out/koth-plugin ./cmd/koth-plugin

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 reduta
USER reduta
COPY --from=build /out/reduta-server /usr/local/bin/reduta-server
COPY --from=build /out/reduta-worker /usr/local/bin/reduta-worker
COPY --from=build /out/reduta-cli /usr/local/bin/reduta-cli
COPY --from=build /out/koth-plugin /usr/local/bin/koth-plugin
EXPOSE 8080
ENTRYPOINT ["reduta-server"]
