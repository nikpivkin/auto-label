FROM golang:1.21-bullseye as build

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go LICENSE README.md ./

RUN CGO_ENABLED=0 go build -o /app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /app /app
ENTRYPOINT ["/app"]