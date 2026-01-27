# Kubernetes Deployment Notes (with Graceful Shutdown & Docker)

## Context

This document summarizes what I learned while preparing a Go-based URL shortener for Kubernetes deployment.

Before writing any Kubernetes manifests, I first ensured that the application itself behaves correctly in a containerized and orchestrated environment. This involved:

1. Implementing **graceful shutdown**
2. Building a **production-friendly Docker image**
3. Deploying the service using a **Kubernetes Deployment and Service**

These steps are tightly coupled and build on top of each other.

---

## 1. Why Graceful Shutdown Is Required

### Problem

In Kubernetes, Pods are **not terminated abruptly** by default.

When Kubernetes needs to:

* Roll out a new version
* Scale down replicas
* Evict a Pod

It first sends a **SIGTERM** signal to the container.

If the application:

* Immediately exits, or
* Ignores SIGTERM,

Then in-flight requests may be dropped, causing user-visible errors.

---

### Goal

Ensure that when the application receives a termination signal:

* It stops accepting new requests
* It allows existing requests to complete
* It exits cleanly within a bounded time

This behavior is required for **zero-downtime deployments**.

---

### Implementation Process

In `main.go`, I implemented graceful shutdown by:

1. Running the HTTP server in a goroutine
2. Listening for `SIGINT` and `SIGTERM`
3. Calling `http.Server.Shutdown(ctx)` with a timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

Key points:

* The timeout is **not an idle timeout**
* It only starts counting **after a shutdown signal is received**
* Kubernetes will forcibly kill the container if it exceeds its own termination grace period

---

### Outcome

With graceful shutdown:

* Kubernetes can safely terminate Pods
* Readiness probes can stop traffic before shutdown
* Rolling updates no longer drop active requests

---

## 2. Dockerfile: Packaging the Application Correctly

### Purpose of the Dockerfile

The Dockerfile defines **how the application is built and executed** in a container.

Its goals are to:

* Produce a small, reproducible image
* Separate build-time and runtime concerns
* Ensure the image behaves the same locally and in Kubernetes

---

### Why Multi-Stage Builds

A multi-stage build allows me to:

* Use a full Go environment to compile the binary
* Ship only the compiled binary into the final image

This avoids:

* Shipping the Go compiler
* Large image sizes
* Unnecessary attack surface

---

### Build Process (Alpine-based)

1. **Builder stage**

   * Uses `golang:alpine`
   * Downloads Go module dependencies
   * Compiles a static Linux binary

2. **Runtime stage**

   * Uses `alpine`
   * Installs CA certificates
   * Copies only the compiled binary

Key build flags:

* `CGO_ENABLED=0` for portability
* `GOOS=linux` to match container runtime

---

### Runtime Behavior

The final container:

* Starts the compiled Go binary directly
* Reads configuration from environment variables
* Exposes port `8080` for Kubernetes to route traffic

This image is then loaded into the local Kubernetes cluster (kind) for deployment.

---

## 3. Kubernetes Deployment: Declaring Desired State

### What a Deployment Does

A Deployment describes the **desired state** of an application, including:

* How many replicas should exist
* What image to run
* How health is checked
* What resources are allocated

Kubernetes continuously reconciles the actual state to match this declaration.

---

### Object Hierarchy

```
Deployment
 └── ReplicaSet
     └── Pod
         └── Container (Go API)
```

Only the Deployment is managed directly; everything else is handled by Kubernetes.

---

## 4. High Availability with Replicas

```yaml
replicas: 2
```

This ensures:

* Two Pods are always running
* Kubernetes automatically replaces failed Pods

High availability is achieved **without any special logic in the application**.

---

## 5. Labels and Selectors

Labels connect Kubernetes resources:

* The Deployment selects Pods by label
* Services route traffic to Pods using the same labels

If labels do not match, traffic will never reach the application.

---

## 6. Container Configuration

```yaml
image: url-shortener:dev
imagePullPolicy: IfNotPresent
```

This configuration is optimized for local development with kind:

* Images are loaded manually into the cluster
* Kubernetes does not attempt to pull from a remote registry

---

## 7. Health Probes and Traffic Control

### Readiness Probe

* Controls **whether traffic is sent to the Pod**
* Used during startup, shutdown, and dependency failures

### Liveness Probe

* Detects application crashes or deadlocks
* Triggers container restarts

These probes integrate tightly with graceful shutdown:

* Readiness fails first → traffic stops
* SIGTERM arrives → shutdown begins
* Liveness is not triggered during normal termination

---

## 8. Resource Management

```yaml
resources:
  requests:
    cpu: "50m"
    memory: "64Mi"
  limits:
    cpu: "200m"
    memory: "128Mi"
```

Resource constraints:

* Enable predictable scheduling
* Prevent noisy-neighbor problems
* Are required for Horizontal Pod Autoscaling (HPA)

---

## 9. Local Verification via Port Forwarding

Before introducing Ingress, traffic is verified using:

```bash
kubectl port-forward svc/url-shortener 8080:80
```

This confirms:

* Docker image is valid
* Pods are healthy
* Service selectors and ports are correct

Ingress is intentionally added **after** this step to reduce debugging complexity.

---

## 10. Key Takeaways

* Graceful shutdown is mandatory for Kubernetes workloads
* Dockerfiles define runtime behavior as much as application code
* Multi-stage builds improve security and image size
* Deployments manage replicas and lifecycle declaratively
* Health probes and shutdown logic must work together

---

## Next Steps

* Add Ingress for HTTP routing
* Attach readiness checks to real dependencies (Redis)
* Enable Horizontal Pod Autoscaling (HPA)