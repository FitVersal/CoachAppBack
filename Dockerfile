FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /src

COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /src/docs/openapi.yaml /src/docs/openapi.yaml

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
