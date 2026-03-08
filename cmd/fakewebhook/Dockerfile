FROM golang:1.26-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/fakewebhook/main.go .

RUN go build -o fakewebhook main.go

EXPOSE 8081

CMD ["./fakewebhook"]
