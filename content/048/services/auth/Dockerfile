FROM golang:1.22-alpine AS build
WORKDIR /src
COPY shared/ ./shared/
COPY auth/ ./auth/
WORKDIR /src/auth
RUN go mod download && CGO_ENABLED=0 go build -o /out/auth .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/auth /auth
USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/auth"]
