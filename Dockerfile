FROM golang:1.23.4-alpine as builder
WORKDIR /usr/local/src
RUN apk --no-cache add bash gcc musl-dev

# Dependencies
COPY ["go.mod", "go.sum", "./"]
RUN go mod download

# Build
COPY . .
RUN go build -o ./bin/app ./cmd/demoservice/main.go


FROM alpine AS runner
WORKDIR /usr/local/src

# Dependencies
ENV CONFIG_PATH=config/deploy.yaml
COPY ./config ./config
COPY ./templates ./templates
COPY --from=builder /usr/local/src/bin/app .

CMD ["./app"]