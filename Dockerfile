# Start from golang v1.11 base image
FROM golang:1.17 as builder

# add project files
WORKDIR /usr/src
COPY main.go go.mod go.sum ./
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main

FROM debian
WORKDIR /usr/src
COPY --from=builder /usr/src/main ./

ENTRYPOINT ./main
