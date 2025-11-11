# Odin Service Discovery Controller

A Helm chart for deploying the Odin Service Discovery Controller to Kubernetes.

## Introduction

The Odin Service Discovery Controller watches Kubernetes Services, Endpoints, and Ingresses for annotations and automatically registers DNS records with the Odin discovery backend service.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Access to the Odin discovery backend service

## Installation

### Basic Installation

```bash
helm install odin-discovery-controller ./helm/odin-discovery-controller \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name" \
  --set image.repository="your-registry/odin-discovery-controller" \
  --set image.tag="1.0.3"
```

### Installation with Custom Namespace

To watch resources in a specific namespace only:

```bash
helm install odin-discovery-controller ./helm/odin-discovery-controller \
  --namespace odin-system \
  --create-namespace \
  --set namespace="default" \
  --set rbac.clusterScoped=false \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name"
```

## Parameters

### Image Configuration

| Name                | Description                                                                                               | Value                         |
| ------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `image`             | Configuration for the container image                                                                     |                               |
| `image.registry`    | Container image registry                                                                                  | `docker.io`                   |
| `image.repository`  | Container image repository                                                                                | `odinhq/discovery-controller` |
| `image.pullPolicy`  | Image pull policy                                                                                         | `IfNotPresent`                |
| `image.tag`         | Container image tag (overrides the chart appVersion)                                                      | `0.0.1`                       |
| `image.pullSecrets` | Docker registry secret names as an array of strings or maps with name field                               | `[]`                          |
| `nameOverride`      | String to partially override common.names.fullname template with a string (will prepend the release name) | `""`                          |
| `fullnameOverride`  | String to fully override common.names.fullname template with a string                                     | `""`                          |

### Service Account

| Name                                          | Description                                                                                                            | Value  |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------ |
| `serviceAccount`                              | Configuration for the service account                                                                                  |        |
| `serviceAccount.create`                       | Specifies whether a service account should be created                                                                  | `true` |
| `serviceAccount.annotations`                  | Annotations to add to the service account                                                                              | `{}`   |
| `serviceAccount.name`                         | The name of the service account to use. If not set and create is true, a name is generated using the fullname template | `""`   |
| `serviceAccount.automountServiceAccountToken` | Automount API credentials for a service account                                                                        | `true` |

### Pod Security Context

| Name                         | Description                        | Value  |
| ---------------------------- | ---------------------------------- | ------ |
| `podSecurityContext`         | Pod security context configuration |        |
| `podSecurityContext.fsGroup` | Group ID for the pod's filesystem  | `2000` |

### Container Security Context

| Name                                     | Description                                | Value     |
| ---------------------------------------- | ------------------------------------------ | --------- |
| `securityContext`                        | Security context for the container         |           |
| `securityContext.capabilities.drop`      | List of capabilities to drop               | `["ALL"]` |
| `securityContext.readOnlyRootFilesystem` | Whether to use a read-only root filesystem | `true`    |
| `securityContext.runAsNonRoot`           | Whether to run as non-root user            | `true`    |
| `securityContext.runAsUser`              | User ID to run the container as            | `1000`    |

### Controller Configuration

| Name                          | Description                                  | Value   |
| ----------------------------- | -------------------------------------------- | ------- |
| `controller`                  | Controller runtime configuration             |         |
| `controller.workers`          | Number of worker threads for processing      | `5`     |
| `controller.discoveryBackend` | Discovery backend service address (required) | `""`    |
| `controller.batchSize`        | Request batch size                           | `5`     |
| `controller.orgId`            | Organisation ID (required)                   | `""`    |
| `controller.accountName`      | Account name (required)                      | `""`    |
| `controller.batchPushTimeout` | Batch push timeout in milliseconds           | `500ms` |

### Namespace Configuration

| Name        | Description                                                                                                                                      | Value |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ----- |
| `namespace` | Namespace to watch. If set, the controller will only watch resources in that namespace. If empty, it will watch all namespaces (cluster-scoped). | `""`  |

### RBAC Configuration

| Name                 | Description                                                                                                                      | Value  |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `rbac`               | RBAC (Role-Based Access Control) configuration                                                                                   |        |
| `rbac.create`        | Specifies whether RBAC resources should be created                                                                               | `true` |
| `rbac.clusterScoped` | If true, creates ClusterRole and ClusterRoleBinding (cluster-scoped). If false, creates Role and RoleBinding (namespace-scoped). | `true` |

### Pod Configuration

| Name             | Description     | Value |
| ---------------- | --------------- | ----- |
| `podAnnotations` | Pod annotations | `{}`  |
| `podLabels`      | Pod labels      | `{}`  |

### Node Configuration

| Name           | Description                       | Value |
| -------------- | --------------------------------- | ----- |
| `nodeSelector` | Node selector for pod assignment  | `{}`  |
| `tolerations`  | Tolerations for pod assignment    | `[]`  |
| `affinity`     | Affinity rules for pod assignment | `{}`  |

### Resource Configuration

| Name                        | Description                  | Value |
| --------------------------- | ---------------------------- | ----- |
| `resources`                 | Resource limits and requests |       |
| `resources.limits.cpu`      | CPU limit                    | `{}`  |
| `resources.limits.memory`   | Memory limit                 | `{}`  |
| `resources.requests.cpu`    | CPU request                  | `{}`  |
| `resources.requests.memory` | Memory request               | `{}`  |

### Additional Configuration

| Name  | Description                      | Value |
| ----- | -------------------------------- | ----- |
| `env` | Additional environment variables | `[]`  |

## Usage

### Annotating Resources

The controller watches for the annotation `discovery.odin/address` on Services and Ingresses.

#### Service Example

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

Multiple addresses can be specified as comma-separated values:

```yaml
annotations:
  discovery.odin/address: "service1.example.com,service2.example.com"
```

#### Ingress Example

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

## Uninstallation

```bash
helm uninstall odin-discovery-controller
```

## Development

### Linting the Chart

```bash
helm lint ./helm/odin-discovery-controller
```

### Dry Run Installation

```bash
helm install odin-discovery-controller ./helm/odin-discovery-controller \
  --dry-run --debug \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name"
```

### Template Rendering

```bash
helm template odin-discovery-controller ./helm/odin-discovery-controller \
  --set controller.discoveryBackend="discovery-service.example.com:8080" \
  --set controller.orgId="your-org-id" \
  --set controller.accountName="your-account-name"
```

## Notes

- The controller uses in-cluster configuration by default (empty kubeconfig path)
- When `namespace` is set, the controller only watches that namespace and requires namespace-scoped RBAC (`rbac.clusterScoped=false`)
- When `namespace` is empty, the controller watches all namespaces and requires cluster-scoped RBAC
- The controller automatically handles Service endpoint changes and Ingress load balancer address updates
