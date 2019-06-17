# Gather dependencies and build the executable
FROM golang:1.10.3 as builder

WORKDIR /go/src/agones.dev
RUN git clone https://github.com/googleforgames/agones.git

WORKDIR /go/src/agones.dev/agones/examples/allocator-service
ADD ./main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .


# Create the final image that will run the allocator service
FROM alpine:3.8
RUN apk add --update ca-certificates
RUN adduser -D service

COPY --from=builder /go/src/agones.dev/agones/examples/allocator-service \
                    /home/service

RUN chown -R service /home/service && \
    chmod o+x /home/service/service

USER service
ENTRYPOINT /home/service/service
