FROM golang:1.22-alpine AS build
WORKDIR /src
COPY shared/ ./shared/
COPY provisioner/ ./provisioner/
WORKDIR /src/provisioner
RUN go mod download && CGO_ENABLED=0 go build -o /out/provisioner .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/provisioner /provisioner
USER nonroot:nonroot
ENTRYPOINT ["/provisioner"]
