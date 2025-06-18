FROM golang:1.24.4 AS build

WORKDIR /build
COPY go.mod go.sum  /build/
RUN go mod download

COPY pkg      /build/pkg/
COPY internal /build/internal/
COPY main.go   /build/
RUN CGO_ENABLED=0 GOOS=linux go build -o /hixi .

FROM gcr.io/distroless/static-debian12:latest

WORKDIR /srv

COPY schema/ /srv/schema/
COPY static/ /srv/static/

COPY --from=build /hixi /usr/bin/hixi

ENTRYPOINT [ "hixi" ]