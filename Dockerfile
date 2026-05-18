FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/workout ./cmd

FROM alpine:3.22

WORKDIR /app
RUN adduser -D -H app \
    && mkdir -p /app/data /app/uploads \
    && chown -R app:app /app

COPY --from=build /out/workout /app/workout

USER app
ENV HOST=0.0.0.0
ENV PORT=8080
ENV APP_DATA_PATH=/app/data/app.json
ENV UPLOAD_DIR=/app/uploads

EXPOSE 8080
CMD ["/app/workout"]
