FROM golang:1.26-trixie AS builder

RUN apt-get update && apt-get install -y git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./main.go


FROM alpine AS production

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /main .

EXPOSE 80

CMD ["./main"]
