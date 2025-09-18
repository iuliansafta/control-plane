FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make protobuf protobuf-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .

RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/controlplane.proto

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o controller cmd/controller/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 controller && \
    adduser -D -u 1000 -G controller controller

WORKDIR /app

COPY --from=builder /app/controller /app/controller

RUN chown -R controller:controller /app

USER controller

EXPOSE 50051

ENV NOMAD_ADDR=""
ENV GRPC_PORT="50051"

CMD ["/app/controller"]
