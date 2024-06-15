FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -v -o ddns .

FROM alpine:latest AS release

WORKDIR /

COPY --from=build /src/ddns /app/ddns

ENTRYPOINT ["/app/ddns"]
