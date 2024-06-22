FROM golang:1.22

# org.opencontainers labels
LABEL org.opencontainers.image.source=https://github.com/slashtechno/cross-blogger
LABEL org.opencontainers.image.description="A Docker image to use Blogger as a headless CMS for static site generators"

WORKDIR /app
COPY . .
RUN go build -o /app/cross-blogger

ENTRYPOINT ["/app/cross-blogger"]