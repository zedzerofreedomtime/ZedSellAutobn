FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/zedsellauto-api ./cmd/api

FROM scratch

WORKDIR /app

COPY --from=builder /out/zedsellauto-api /app/zedsellauto-api

EXPOSE 18081

ENTRYPOINT ["/app/zedsellauto-api"]
