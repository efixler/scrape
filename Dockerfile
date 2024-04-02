FROM golang:latest AS builder

ENV CGO_ENABLED=1

RUN apt -y update && apt -y upgrade
RUN apt-get -y install sqlite3
WORKDIR /scrape
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go install -v ./cmd/scrape
RUN go install -v ./cmd/scrape-server
WORKDIR /go/bin


FROM debian:12-slim

RUN apt -y update && apt -y upgrade
RUN apt-get -y install \ 
    sqlite3 \
    ca-certificates \
    curl \
    chromium \
    gnupg wget apt-transport-https
RUN mkdir -p /scrape/bin
COPY --from=builder /go/bin/* /scrape/bin/
RUN mkdir -p /scrape_data
VOLUME [ "/scrape_data" ]
EXPOSE 8080/tcp
CMD ["cd", "/"]
# The default sqlite db will be in /scrape_data/scrape.db
ENTRYPOINT ["/scrape/bin/scrape-server"]