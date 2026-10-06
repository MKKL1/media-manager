FROM golang:1.27-trixie AS builder

RUN apt-get update && apt-get install -y git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./main.go
RUN ./scripts/build-plugins.sh


FROM alpine AS production

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /main .
COPY --from=builder /app/plugins/wasm/tmdb/tmdb.wasm plugins/wasm/tmdb/tmdb.wasm
COPY --from=builder /app/plugins/wasm/tv/tv.wasm plugins/wasm/tv/tv.wasm
COPY --from=builder /app/plugins/wasm/movie/movie.wasm plugins/wasm/movie/movie.wasm

EXPOSE 80

CMD ["./main"]
