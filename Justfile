IMAGE := "kuberhealthy/namespace-pod-check"
TAG := "latest"

# Build the namespace pod check container locally.
build:
	podman build -f Containerfile -t {{IMAGE}}:{{TAG}} .

# Run the unit tests for the namespace pod check.
test:
	go test ./...

# Build the namespace pod check binary locally.
binary:
	go build -o bin/namespace-pod-check ./cmd/namespace-pod-check
