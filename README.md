# Odin Discovery Controller

A Kubernetes controller that watches Services, Endpoints, and Ingresses for annotations and automatically registers DNS records with the Odin discovery backend service.

## Overview

The Odin Service Discovery Controller is a Kubernetes controller that:

- Watches Kubernetes Services, Endpoints, and Ingresses
- Detects resources annotated with `discovery.odin/address`
- Automatically registers DNS records with the Odin discovery backend
- Handles Service endpoint changes and Ingress load balancer updates
- Supports batch processing for efficient DNS record management

## Features

- **Automatic DNS Registration**: Automatically registers DNS records when Services or Ingresses are annotated
- **Endpoint Monitoring**: Tracks Service endpoints and updates DNS records accordingly
- **Batch Processing**: Efficiently batches DNS updates to reduce API calls
- **Namespace Scoping**: Supports both cluster-scoped and namespace-scoped operation
- **RBAC Ready**: Includes proper RBAC permissions for Kubernetes resources

## Quick Start

### Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Access to the Odin discovery backend service

### Installation

#### Using Helm Repository (Recommended)

Once the chart is published to GitHub Pages, you can install it via:

```bash
# Add the Helm repository
helm repo add odin-discovery-controller https://<your-org>.github.io/odin-discovery-controller/helm-repo
helm repo update

# Install the controller
helm install odin-discovery-controller odin-discovery-controller/odin-discovery-controller \
  --namespace odin-system \
  --create-namespace \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name"
```

#### Using Local Chart

```bash
helm install odin-discovery-controller ./charts/odin-discovery-controller \
  --namespace odin-system \
  --create-namespace \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name"
```

## Usage

### Annotating Services

Add the `discovery.odin/address` annotation to your Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    discovery.odin/address: "my-service.example.com"
spec:
  ports:
  - port: 80
    targetPort: 8080
```

### Annotating Ingresses

Add the annotation to your Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  annotations:
    discovery.odin/address: "my-app.example.com"
spec:
  rules:
  - host: my-app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80
```

### Multiple Addresses

You can specify multiple addresses as comma-separated values:

```yaml
annotations:
  discovery.odin/address: "service1.example.com,service2.example.com"
```

## Configuration

See the [Helm Chart README](charts/odin-discovery-controller/README.md) for complete configuration options.

### Key Configuration Parameters

- `controller.discoveryBackend`: Address of the Odin discovery backend service (required)
- `controller.orgId`: Organisation ID (required)
- `controller.accountName`: Account name (required)
- `controller.workers`: Number of worker threads (default: 5)
- `controller.batchSize`: Request batch size (default: 5)
- `namespace`: Namespace to watch (empty = all namespaces)

## Development

### Building

```bash
go build -o odin-discovery-controller ./cmd/main.go
```

### Running Locally

```bash
./odin-discovery-controller \
  --kubeconfig=/path/to/kubeconfig \
  --discoverybackend="discovery-service.example.com:8080" \
  --orgid="your-org-id" \
  --accountname="your-account-name" \
  --workers=5 \
  --batchsize=5
```

### Building Docker Image

```bash
docker build -t odin-discovery-controller:latest .
```

## Helm Chart Development

### Linting

```bash
cd charts/odin-discovery-controller
helm lint .
```

### Testing

```bash
# Dry-run installation
helm install test-release . --dry-run --debug

# Template rendering
helm template test-release .
```

### Updating Dependencies

```bash
cd charts/odin-discovery-controller
helm dependency update
```

## CI/CD

This repository includes GitHub Actions workflows for:

- **Docker Build**: Builds and publishes Docker images
- **Go Lint**: Runs golangci-lint checks
- **Helm Release**: Automatically releases Helm charts on version tags and publishes to GitHub Pages

### Helm Chart Release Process

When you create a git tag starting with `v` (e.g., `v0.1.0`), the release workflow will:

1. Update Chart.yaml with the version
2. Package the Helm chart
3. Create/update the Helm repository index
4. Create a GitHub Release with chart artifacts
5. Publish the chart repository to GitHub Pages

The chart will be available at: `https://<your-org>.github.io/odin-discovery-controller/helm-repo`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

Copyright © Dream Horizon. All rights reserved.

## Support

For issues and questions, please open an issue on GitHub.

## Links

- [Helm Chart Documentation](charts/odin-discovery-controller/README.md)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
