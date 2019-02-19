FROM golang:1.11.1 AS build

# Copy the source files to the proper path
RUN mkdir -p /go/src/github.com/su225/k8s-clustering/
WORKDIR /go/src/github.com/su225/k8s-clustering
COPY . .

# Compile and build the binary of the service
# Disable CGO since it is not required.
RUN CGO_ENABLED=0 GOOS=linux \
    go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o k8s-clustering

# Now get the binary from the build container and build
# the actual container
FROM alpine:3.9

# Create the directory for the application and create the
# user with that home directory and switch the user to non-root
RUN mkdir node && adduser -S -D -H -h /node nodeuser
USER nodeuser

# Copy the binary and launch it on port 8080
COPY --from=build /go/src/github.com/su225/k8s-clustering/k8s-clustering /node
CMD /node/k8s-clustering
EXPOSE 8888