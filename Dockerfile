FROM golang:1.27.3 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal/ ./internal/
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/gitea-issue-ics .

FROM alpine:3.24
COPY --from=build /out/gitea-issue-ics /gitea-issue-ics
EXPOSE 8080
ENTRYPOINT ["/gitea-issue-ics"]
