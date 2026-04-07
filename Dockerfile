FROM golang:1.24-alpine AS build

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1

RUN GOOS=linux go build -o server .

FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat

WORKDIR /app

COPY --from=build /app/server .
COPY --from=build /app/web ./web

EXPOSE 8080

ENV PORT=8080
ENV JWT_SECRET=change-me-in-production
ENV DB_PATH=license_management.db

CMD ["./server"]
