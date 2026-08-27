FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata curl

COPY connect-linux-amd64 /app/connect-app
RUN chmod +x /app/connect-app
COPY schema.sql /app/schema.sql

EXPOSE 8080
ENV PORT=8080

CMD ["/app/connect-app"]
