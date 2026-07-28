FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /model-server ./cmd/model-server

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /model-server /model-server
EXPOSE 8080
ENTRYPOINT ["/model-server"]
