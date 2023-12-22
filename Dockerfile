FROM golang:latest AS builder

RUN apt update && apt upgrade
RUN apt-get -y install sqlite3
ENV CGO_ENABLED=1
RUN go install github.com/efixler/scrape/cmd/scrape-server@latest
RUN go install github.com/efixler/scrape/cmd/scrape@latest
WORKDIR /go/bin


FROM debian:12-slim

RUN apt update && apt upgrade
RUN apt-get -y install sqlite3 ca-certificates 
RUN mkdir -p /scrape/bin
COPY --from=builder /go/bin/* /scrape/bin/
RUN mkdir -p /scrape_data
VOLUME [ "/scrape_data" ]
EXPOSE 8080/tcp
CMD ["cd", "/"]
ENTRYPOINT ["/scrape/bin/scrape-server"]