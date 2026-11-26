# Built by `snapctl byojob publish`, which builds this Dockerfile for
# linux/arm64, the platform Snapser runs job workers on. The builder stage runs natively and
# cross-compiles for the target platform, so building on an amd64 machine
# does not emulate the Go build.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.22 AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /worker .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /worker /worker
EXPOSE 8080
ENTRYPOINT ["/worker"]
