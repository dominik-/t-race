FROM golang:latest as builder

RUN mkdir /app

ADD . /app/

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o t-race .

FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/t-race .

# Command to run the executable
ENTRYPOINT ["./t-race"] 