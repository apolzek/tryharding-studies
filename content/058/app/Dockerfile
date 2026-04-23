FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bigbench .

FROM gcr.io/distroless/static-debian12
COPY --from=build /bigbench /bigbench
EXPOSE 8765
USER 65532:65532
ENTRYPOINT ["/bigbench"]
