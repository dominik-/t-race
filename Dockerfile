FROM golang:latest as builder

RUN mkdir /app

ADD . /app/

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o t-race .

FROM alpine:latest  

RUN apk --no-cache add ca-certificates
RUN apk add iproute2

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/t-race .

EXPOSE 7000
EXPOSE 8000
EXPOSE 9000

# Command to run the executable
ENTRYPOINT ["./t-race"] 