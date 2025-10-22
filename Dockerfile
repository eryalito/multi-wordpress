FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mwpfm .

FROM ubuntu:24.04

RUN apt update && apt upgrade -y && apt install -y apache2 wget php libapache2-mod-php php-mysql php-curl php-gd php-mbstring php-xml php-xmlrpc php-soap php-intl php-zip inotify-tools

RUN wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz && \
    rm go1.24.0.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

COPY --from=builder /app/mwpfm /usr/local/bin/mwpfm

COPY reload_apache.sh /usr/local/bin/reload_apache.sh
RUN chmod +x /usr/local/bin/reload_apache.sh

CMD ["reload_apache.sh"]
