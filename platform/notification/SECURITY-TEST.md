# Notification visibility verification

The Notification endpoint intentionally declares no additional visibility.
OpenChoreo v1.2 gives every endpoint implicit `project` visibility; adding
`internal` would broaden it to every namespace, and adding `external` would
create public ingress. Scan consumes the endpoint through a `project`
dependency and receives its address as `NOTIFICATION_SERVICE_URL`.

Run the automated evidence check after both components report Ready:

```bash
./scripts/test-notification-visibility.sh
```

The script proves all of the following rather than treating one failed external
request as sufficient evidence:

1. the committed Notification workload does not request external visibility;
2. both generated deployments are Available;
3. Notification responds when its cluster Service is explicitly port-forwarded;
4. the public gateway does not route the Notification endpoint;
5. the generated Scan Deployment contains the injected dependency address;
6. a real externally submitted scan completes and creates a notification; and
7. OpenChoreo generated a NetworkPolicy for Notification.

Capture the successful output and these resources for the demo evidence:

```bash
kubectl get component,workload -n default
kubectl get deployment,service,httproute,networkpolicy -A \
  -l openchoreo.dev/component=notification-service -o wide
kubectl get deployment -A -l openchoreo.dev/component=scan-service -o yaml
```

Do not publish the last command's raw output if future dependencies introduce
secrets. A gateway 404 alone is not proof of isolation: it could also mean a
broken deployment, which is why the script separately checks internal health
and the end-to-end dependency.
