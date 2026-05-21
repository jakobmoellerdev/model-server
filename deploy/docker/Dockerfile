FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /model-server ./cmd/model-server

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /model-server /model-server
EXPOSE 8080
ENTRYPOINT ["/model-server"]
