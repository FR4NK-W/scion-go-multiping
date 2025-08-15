FROM golang:1.23-alpine

WORKDIR /src
COPY . .
RUN apk add build-base
RUN go mod tidy
RUN CGO_ENABLED=1 go build -o app
RUN cp app /bin/importer

ENTRYPOINT ["/bin/importer"]