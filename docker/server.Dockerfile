FROM golang:1.24.4 AS build-server

WORKDIR /build
COPY go.mod go.sum  /build/
RUN go mod download

COPY pkg      /build/pkg/
COPY internal /build/internal/
COPY main.go   /build/
RUN CGO_ENABLED=0 GOOS=linux go build -o hixi .

FROM node:24 AS build-static

WORKDIR /build
COPY package.json /build/
COPY package-lock.json /build/
RUN npm install

COPY svelte.config.js /build/
COPY vite.config.ts   /build/
COPY index.html       /build/
COPY ui               /build/ui/
RUN npm run build

FROM gcr.io/distroless/static-debian12:latest

WORKDIR /srv

COPY                      schema/    /srv/schema/
COPY --from=build-static /build/dist /srv/dist
COPY --from=build-server /build/hixi /usr/bin/hixi

ENTRYPOINT [ "hixi" ]
