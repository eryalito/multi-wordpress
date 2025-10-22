FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mwpfm .

FROM ubuntu:24.04

RUN apt update && apt upgrade -y && apt install -y apache2 wget php libapache2-mod-php php-mysql php-curl php-gd php-mbstring php-xml php-xmlrpc php-soap php-intl php-zip inotify-tools \
    && apt clean && rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/local/go/bin:${PATH}"

COPY --from=builder /app/mwpfm /usr/local/bin/mwpfm

COPY reload_apache.sh /usr/local/bin/reload_apache.sh
RUN chmod +x /usr/local/bin/reload_apache.sh

## Clean and configure default apache behavior
RUN rm -rf /var/www/html && rm -rf /etc/apache2/sites-enabled \
    && rm -rf /etc/apache2/sites-available \
    && mkdir -p /etc/apache2/sites-available \
    && mkdir -p /etc/apache2/sites-enabled \
    && mkdir -p /var/www/html \
    && mkdir -p /var/www/empty \
    && sed -i 's/Listen 80/Listen 8080/' /etc/apache2/ports.conf \
    && chown -R www-data:www-data /var/www/html /var/log/apache2 /var/run/apache2 /etc/apache2
    
COPY apache-default.conf /etc/apache2/sites-available/000-default.conf

RUN ln -s /etc/apache2/sites-available/000-default.conf /etc/apache2/sites-enabled/000-default.conf

USER www-data

CMD ["reload_apache.sh"]
