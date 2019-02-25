FROM golang:1.11 as builder
COPY . /app
WORKDIR /app
RUN go build -o crono
CMD /app/crono

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/crono .
CMD ./crono