---
title: "Traefik Kubernetes Gateway API Documentation"
description: "Learn how to use the Kubernetes Gateway API as a provider for configuration discovery in Traefik Proxy. Read the technical documentation."
---

# Traefik & Kubernetes with Gateway API

The Kubernetes Gateway provider is a Traefik implementation of the [Gateway API](https://gateway-api.sigs.k8s.io/)
specification from the Kubernetes Special Interest Groups (SIGs).

This provider supports Standard version [v1.6.1](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v1.6.1) of the Gateway API specification.

It fully supports all `HTTPRoute` core and some extended features, like `BackendTLSPolicy`, `GRPCRoute`, and `TLSRoute` resources from the [Standard channel](https://gateway-api.sigs.k8s.io/concepts/versioning/?h=#release-channels), as well as `TCPRoute` from the [Experimental channel](https://gateway-api.sigs.k8s.io/concepts/versioning/?h=#release-channels).

For more details, check out the conformance [report](https://github.com/kubernetes-sigs/gateway-api/tree/main/conformance/reports/v1.6.1/traefik-traefik).

!!! info "Using The Helm Chart"

    When using the Traefik [Helm Chart](../../../../getting-started/kubernetes.md#install-traefik), the RBAC (Role-Based Access Control) are automatically managed for you.

## Requirements

{% include-markdown "includes/kubernetes-requirements.md" %}

1. Install/update the Kubernetes Gateway API CRDs.

    ```bash
    # Install Gateway API CRDs from the Standard channel.
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.6.1/standard-install.yaml
    ```

    !!! warning "`experimentalChannel` requires the Experimental channel CRDs"

        The `experimentalChannel` option makes Traefik watch `TCPRoute` and `TLSRoute` resources,
        which the Standard channel CRDs do not provide.
        Enabling it without installing the Experimental channel CRDs
        (`experimental-install.yaml`) prevents the Kubernetes Gateway provider from starting:
        no Gateway API resource is served at all, not only `TCPRoute` and `TLSRoute`.
        Traefik keeps running and the other providers are unaffected.

2. If you are not using the Helm Chart, install/update the Traefik [RBAC](../../../dynamic-configuration/kubernetes-gateway-rbac.yml) for Gateway API.

    ```bash
    # Install Traefik RBACs for Gateway API.
    kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.7/docs/content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml
    ```

## Configuration Example

You can enable the `kubernetesGateway` provider as detailed below:

```yaml tab="File (YAML)"
providers:
  kubernetesGateway: {}
  # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
# ...
```

```bash tab="CLI"
--providers.kubernetesgateway=true
```

```yaml tab="Helm Chart Values"
## Values file
providers:
  kubernetesGateway:
    enabled: true
```

## Configuration Options

<!-- markdownlint-disable MD013 -->

| Field                                                                 | Description                                                                                                                                                                                                                                                                                                                                                                          | Default | Required |
|:----------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------|:---------|
| <a id="opt-providers-providersThrottleDuration" href="#opt-providers-providersThrottleDuration" title="#opt-providers-providersThrottleDuration">`providers.providersThrottleDuration`</a> | Minimum amount of time to wait for, after a configuration reload, before taking into account any new configuration refresh event.<br />If multiple events occur within this time, only the most recent one is taken into account, and all others are discarded.<br />**This option cannot be set per provider, but the throttling algorithm applies to each of them independently.** | 2s      | No       |
| <a id="opt-providers-kubernetesGateway-endpoint" href="#opt-providers-kubernetesGateway-endpoint" title="#opt-providers-kubernetesGateway-endpoint">`providers.kubernetesGateway.endpoint`</a> | Server endpoint URL.<br />More information [here](#endpoint).                                                                                                                                                                                                                                                                                                                        | ""      | No       |
| <a id="opt-providers-kubernetesGateway-experimentalChannel" href="#opt-providers-kubernetesGateway-experimentalChannel" title="#opt-providers-kubernetesGateway-experimentalChannel">`providers.kubernetesGateway.experimentalChannel`</a> | Toggles support for the Experimental Channel resources ([Gateway API release channels documentation](https://gateway-api.sigs.k8s.io/concepts/versioning/#release-channels)).<br />(ex: `TCPRoute`)<br />Requires the Experimental channel CRDs to be installed in the cluster, see [Requirements](#requirements).                                                    | false   | No       |
| <a id="opt-providers-kubernetesGateway-token" href="#opt-providers-kubernetesGateway-token" title="#opt-providers-kubernetesGateway-token">`providers.kubernetesGateway.token`</a> | Bearer token used for the Kubernetes client configuration. Accepts either the token value directly or a path to a file containing the token.                                                                                                                                                                                                                                         | ""      | No       |
| <a id="opt-providers-kubernetesGateway-certAuthFilePath" href="#opt-providers-kubernetesGateway-certAuthFilePath" title="#opt-providers-kubernetesGateway-certAuthFilePath">`providers.kubernetesGateway.certAuthFilePath`</a> | Path to the certificate authority file.<br />Used for the Kubernetes client configuration.                                                                                                                                                                                                                                                                                           | ""      | No       |
| <a id="opt-providers-kubernetesGateway-namespaces" href="#opt-providers-kubernetesGateway-namespaces" title="#opt-providers-kubernetesGateway-namespaces">`providers.kubernetesGateway.namespaces`</a> | Array of namespaces to watch.<br />If left empty, watch all namespaces.                                                                                                                                                                                                                                                                                                              | []      | No       |
| <a id="opt-providers-kubernetesGateway-labelSelector" href="#opt-providers-kubernetesGateway-labelSelector" title="#opt-providers-kubernetesGateway-labelSelector">`providers.kubernetesGateway.labelSelector`</a> | Allow filtering on `GatewayClass` only. If left empty, Traefik processes all GatewayClass objects in the configured namespaces.<br />See [label-selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) for details.                                                                                                                   | ""      | No       |
| <a id="opt-providers-kubernetesGateway-throttleDuration" href="#opt-providers-kubernetesGateway-throttleDuration" title="#opt-providers-kubernetesGateway-throttleDuration">`providers.kubernetesGateway.throttleDuration`</a> | Minimum amount of time to wait between two Kubernetes events before producing a new configuration.<br />This prevents a Kubernetes cluster that updates many times per second from continuously changing your Traefik configuration.<br />If empty, every event is caught.                                                                                                           | 0s      | No       |
| <a id="opt-providers-kubernetesGateway-nativeLBByDefault" href="#opt-providers-kubernetesGateway-nativeLBByDefault" title="#opt-providers-kubernetesGateway-nativeLBByDefault">`providers.kubernetesGateway.nativeLBByDefault`</a> | Defines whether to use Native Kubernetes load-balancing mode by default. For more information, please check out the `traefik.io/service.nativelb` service annotation documentation.                                                                                                                                                                                                  | false   | No       |
| <a id="opt-providers-kubernetesGateway-statusAddress-hostname" href="#opt-providers-kubernetesGateway-statusAddress-hostname" title="#opt-providers-kubernetesGateway-statusAddress-hostname">`providers.kubernetesGateway.`<br />`statusAddress.hostname`</a> | Hostname copied to the Gateway `status.addresses`.                                                                                                                                                                                                                                                                                                                                   | ""      | No       |
| <a id="opt-providers-kubernetesGateway-statusAddress-ip" href="#opt-providers-kubernetesGateway-statusAddress-ip" title="#opt-providers-kubernetesGateway-statusAddress-ip">`providers.kubernetesGateway.`<br />`statusAddress.ip`</a> | IP address copied to the Gateway `status.addresses`, and currently only supports one IP value (IPv4 or IPv6).                                                                                                                                                                                                                                                                        | ""      | No       |
| <a id="opt-providers-kubernetesGateway-statusAddress-service-namespace" href="#opt-providers-kubernetesGateway-statusAddress-service-namespace" title="#opt-providers-kubernetesGateway-statusAddress-service-namespace">`providers.kubernetesGateway.`<br />`statusAddress.service.namespace`</a> | The namespace of the Kubernetes service to copy status addresses from.<br />When using third parties tools like External-DNS, this option can be used to copy the service `loadbalancer.status` (containing the service's endpoints IPs) to the Gateway `status.addresses`.                                                                                                          | ""      | No       |
| <a id="opt-providers-kubernetesGateway-statusAddress-service-name" href="#opt-providers-kubernetesGateway-statusAddress-service-name" title="#opt-providers-kubernetesGateway-statusAddress-service-name">`providers.kubernetesGateway.`<br />`statusAddress.service.name`</a> | The name of the Kubernetes service to copy status addresses from.<br />When using third parties tools like External-DNS, this option can be used to copy the service `loadbalancer.status` (containing the service's endpoints IPs) to the Gateway `status.addresses`.                                                                                                               | ""      | No       |
| <a id="opt-providers-kubernetesGateway-crossProviderNamespaces" href="#opt-providers-kubernetesGateway-crossProviderNamespaces" title="#opt-providers-kubernetesGateway-crossProviderNamespaces">`providers.kubernetesGateway.crossProviderNamespaces`</a> | List of namespaces from which Gateway API routes (`HTTPRoute`, `TCPRoute`, `TLSRoute`) are allowed to declare a `backendRef` of kind `TraefikService`.<br />When unset, all namespaces are allowed. When set to `[]`, every such backendRef is rejected and the route is dropped.                                                                                                    | []      | No       |
| <a id="opt-providers-kubernetesGateway-gateways" href="#opt-providers-kubernetesGateway-gateways" title="#opt-providers-kubernetesGateway-gateways">`providers.kubernetesGateway.gateways`</a> | Restricts the provider to the given `Gateway` resources, expressed as `namespace/name`.<br />When unset, every `Gateway` of the managed `GatewayClass` resources is handled.<br />More information [here](#gateways).                                                                                                                     | []      | No       |
| <a id="opt-providers-kubernetesGateway-gatewayClasses" href="#opt-providers-kubernetesGateway-gatewayClasses" title="#opt-providers-kubernetesGateway-gatewayClasses">`providers.kubernetesGateway.gatewayClasses`</a> | Restricts the provider to the `Gateway` resources of the given `GatewayClass` resources.<br />When unset, every managed `GatewayClass` is handled.<br />More information [here](#gatewayclasses).                                                                                                                     | []      | No       |
| <a id="opt-providers-kubernetesGateway-disableGatewayClassStatus" href="#opt-providers-kubernetesGateway-disableGatewayClassStatus" title="#opt-providers-kubernetesGateway-disableGatewayClassStatus">`providers.kubernetesGateway.disableGatewayClassStatus`</a> | Disables the `GatewayClass` status update, leaving it to an external controller.<br />`Gateway`, listener and route statuses are still written by Traefik.<br />More information [here](#disablegatewayclassstatus).                                                                                        | false   | No       |
| <a id="opt-providers-kubernetesGateway-gatewayIsolation" href="#opt-providers-kubernetesGateway-gatewayIsolation" title="#opt-providers-kubernetesGateway-gatewayIsolation">`providers.kubernetesGateway.gatewayIsolation`</a> | Resolves the address of each `Gateway` from a `Service` an external controller provisions for it, and restricts the routers of that `Gateway` to that address.<br />More information [here](#gatewayisolation).                                                                                        | false   | No       |
| <a id="opt-providers-kubernetesGateway-qps" href="#opt-providers-kubernetesGateway-qps" title="#opt-providers-kubernetesGateway-qps">`providers.kubernetesGateway.qps`</a> | Defines the maximum QPS to the Kubernetes API server. Setting this to a negative value will disable client-side ratelimiting.                                                                                                                                                                                                                                                        | 50 | No       |
| <a id="opt-providers-kubernetesGateway-burst" href="#opt-providers-kubernetesGateway-burst" title="#opt-providers-kubernetesGateway-burst">`providers.kubernetesGateway.burst`</a> | Defines the maximum burst of requests to the Kubernetes API server.                                                                                                                                                                                                                                                                                                                  | 100 | No |

<!-- markdownlint-enable MD013 -->

### `endpoint`

The Kubernetes server endpoint URL.

When deployed into Kubernetes, Traefik reads the environment variables
`KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` or `KUBECONFIG` to
construct the endpoint.

The access token is looked up
in `/var/run/secrets/kubernetes.io/serviceaccount/token`
and the SSL CA certificate in `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`.
Both are mounted automatically when deployed inside Kubernetes.

The endpoint may be specified to override the environment variable values
inside a cluster.

When the environment variables are not found, Traefik tries to connect to
the Kubernetes API server with an external-cluster client.
In this case, the endpoint is required.
Specifically, it may be set to the URL used by `kubectl proxy` to connect to a
Kubernetes cluster using the granted authentication
and authorization of the associated kubeconfig.

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    endpoint: "http://localhost:8080"
    # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  endpoint = "http://localhost:8080"
  # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.endpoint=http://localhost:8080
```

### `gateways`

Restricts the provider to a fixed set of `Gateway` resources, expressed as
`namespace/name`. Every other `Gateway` is ignored: no routing configuration is
produced for it, and its status is left untouched.

This is meant for topologies where a Traefik instance is dedicated to one
`Gateway` (or to a group of them) rather than serving every `Gateway` of the
cluster, typically when an external controller provisions one data plane per
`Gateway`.

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    gateways:
      - default/my-gateway
    # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  gateways = ["default/my-gateway"]
  # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.gateways=default/my-gateway
```

### `gatewayClasses`

Restricts the provider to the `Gateway` resources of a fixed set of
`GatewayClass` resources, by name.

It serves the same purpose as [`gateways`](#gateways), for an instance serving a
whole `GatewayClass` rather than named `Gateway` resources. Unlike a `Gateway`
list, it does not have to be updated, and the instance restarted, every time a
`Gateway` is created or deleted.

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    gatewayClasses:
      - my-gateway-class
    # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  gatewayClasses = ["my-gateway-class"]
  # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.gatewayclasses=my-gateway-class
```

### `disableGatewayClassStatus`

Stops Traefik from writing the `GatewayClass` status, so that an external
controller can own it. The `Gateway`, listener and route statuses are still
written by Traefik.

This is required when an external controller publishes a supported feature set
that spans more than the data plane, for instance because it implements the
infrastructure part of the Gateway API itself. Two writers on the same
`GatewayClass` status would otherwise overwrite each other.

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    disableGatewayClassStatus: true
    # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  disableGatewayClassStatus = true
  # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.disablegatewayclassstatus=true
```

### `gatewayIsolation`

Gives each `Gateway` served by the instance an address of its own, and restricts
the routers it contributes to that address. Without it, the `Gateway` resources
sharing an instance answer at the same address, so a request no router of its own
`Gateway` matches falls through to a router another `Gateway` contributed to the
same entry point.

Traefik resolves the address of a `Gateway` from the `Service` an external
controller provisions for it, found by the labels it carries:

| Label | Value |
|---|---|
| <a id="opt-gateway-networking-k8s-iogateway-name" href="#opt-gateway-networking-k8s-iogateway-name" title="#opt-gateway-networking-k8s-iogateway-name">`gateway.networking.k8s.io/gateway-name`</a> | the `Gateway` name |
| <a id="opt-gateway-traefik-iogateway-namespace" href="#opt-gateway-traefik-iogateway-namespace" title="#opt-gateway-traefik-iogateway-namespace">`gateway.traefik.io/gateway-namespace`</a> | the `Gateway` namespace |

Only IP addresses are used: a hostname cannot be compared with the address a
connection was accepted on. A `Gateway` whose `Service` has no address yet is
reported as not programmed.

The routers carry a [`DstIP`](../../../routing-configuration/http/routing/rules-and-priority.md#dstip)
matcher, which sees the address the connection was accepted on. Behind a
destination NAT, a `Service` of type `LoadBalancer` for instance, that is the
translated address, so the entry points have to be configured with
[`originalDestination`](../../entrypoints.md#originaldestination) and the pod has
to run on the host network, where the connection tracking that keeps the original
address lives.

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    gatewayIsolation: true
    # ...
```

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  gatewayIsolation = true
  # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.gatewayisolation=true
```

## Routing Configuration

See the dedicated section in [routing](../../../../reference/routing-configuration/kubernetes/gateway-api.md).

!!! tip "Routing Configuration"

    When using the Kubernetes Gateway API provider, Traefik uses the Gateway API
    CRDs to retrieve its routing configuration.
    Check out the Gateway API concepts [documentation](https://gateway-api.sigs.k8s.io/concepts/api-overview/),
    and the dedicated [routing section](../../../../reference/routing-configuration/kubernetes/gateway-api.md)
    in the Traefik documentation.

{% include-markdown "includes/traefik-for-business-applications.md" %}
