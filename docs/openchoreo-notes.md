# Docker, Kubernetes, and OpenChoreo notes

## Docker fundamentals

An image is an immutable application package: filesystem layers, runtime
metadata, and a default command. A container is a running (or stopped) instance
of that image with a small writable layer. Rebuilding an image does not change
an existing container. Each Dockerfile instruction can add a cacheable layer,
so stable dependency downloads should appear before frequently changed source
code. This keeps rebuilds quick and avoids shipping development files.

A multi-stage build uses one image to compile and another to run. For the Scan
Service, the builder contains Go and module tooling; the runtime contains only
the compiled binary and trusted CA certificates. The result has fewer packages,
a smaller attack surface, and no compiler in production. Useful commands are:

```bash
docker build -t securecloud-scan:v0.1.0 services/scan
docker run --rm -p 8080:8080 securecloud-scan:v0.1.0
docker logs <container>
docker exec -it <container> /bin/sh
```

`docker logs` reads stdout/stderr, which is why services emit structured JSON
there. `docker exec` runs a process inside an existing container, but it is not
guaranteed to work in hardened or distroless images that intentionally contain
no shell. In that case I inspect metadata with `docker inspect`, use health
endpoints, or attach a purpose-built debug container.

## Kubernetes fundamentals

A **Pod** is Kubernetes' smallest scheduling unit. It holds one or more tightly
coupled containers that share networking and volumes. Pods are disposable; an
application should not depend on a specific pod name or IP.

A **Deployment** describes the desired replica count and pod template. Its
controller creates ReplicaSets, replaces unhealthy pods, and rolls out a new
template. A **Service** gives a changing group of pods a stable virtual address
and selects them by labels. Clients call the Service, not individual pods.

A **Namespace** scopes names and provides an administrative boundary for RBAC,
quotas, and network policy. OpenChoreo stores developer intent in a control-plane
namespace and runs generated workloads in data-plane namespaces. A **ConfigMap**
holds non-secret configuration such as a log level. A **Secret** is intended for
sensitive values, although its data is merely base64-encoded unless encryption
at rest and access controls are configured. Secrets must never be committed as
literal values; workloads reference them by name and key.

The basic investigation loop is:

```bash
kubectl get pods,deployments,services -A
kubectl describe pod <pod> -n <namespace>
kubectl logs deployment/<deployment> -n <namespace> --tail=100
kubectl exec -it <pod> -n <namespace> -- <command>
```

`get` answers what exists and its current status. `describe` shows configuration
and controller events, which often reveal failed scheduling, image pulls, or
probes. `logs` shows application output. `exec` is for focused runtime checks;
it should not become a way to make manual, untracked production changes.

## OpenChoreo resource chain

The `Component` expresses the developer's intent and selects a platform-owned
component type. Its `Workload` supplies the container, endpoints, configuration,
and declared dependencies. OpenChoreo snapshots this intent into an immutable
`ComponentRelease`. A `ReleaseBinding` selects that release for an environment.
The data plane then receives ordinary Kubernetes resources such as a Deployment,
Service, HTTPRoute, and NetworkPolicy. In short:

```text
Component + Workload -> ComponentRelease -> ReleaseBinding
                     -> Deployment + Service + HTTPRoute + NetworkPolicy
```

This separation lets developers describe what should run without hand-writing
every Kubernetes object, while the platform team controls how component types
render and enforce policy. I would trace a deployment with:

```bash
kubectl get component,workload -n default
kubectl get componentrelease,releasebinding -n default
kubectl get deployment,service,httproute -A \
  -l openchoreo.dev/component=scan-service
```

The Scan Service consumes Notification through an endpoint dependency. In
OpenChoreo v1.2 the dependency names the component and endpoint, uses `project`
visibility, and binds the resolved address to `NOTIFICATION_SERVICE_URL`. The
Notification endpoint needs no extra visibility entry: project visibility is
implicit and narrower than `internal` or `external`. OpenChoreo can then inject
the environment variable and permit the declared traffic without committing a
DNS name. This is both portable configuration and an explicit security boundary.
