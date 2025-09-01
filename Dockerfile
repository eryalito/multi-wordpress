FROM ubuntu:24.04

RUN apt update && apt upgrade -y && apt install -y apache2 wget

RUN wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz && \
    rm go1.24.0.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

COPY . .

CMD ["go", "run", "main.go"]
