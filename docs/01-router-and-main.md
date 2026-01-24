# What I Learned: Introducing chi Router in a Go Service

## Context

This project is a production-style URL shortener built with Go and Kubernetes.
At an early stage, the service only exposed basic endpoints such as `/healthz`, but the goal was to gradually evolve it into a Kubernetes-ready service with proper routing, observability, and lifecycle management.


## 1. Why I Introduced a Router Library (chi)

Initially, the service used Go’s built-in `http.ServeMux`, which is perfectly fine for very small demos. However, as the service grew toward a production-style system, several limitations became clear:

* Manual path parsing becomes necessary for dynamic routes (e.g. `/{code}`)

* Middleware support (logging, timeouts, recovery) is limited and clunky

* Routing logic and server startup logic tend to get mixed together

I chose `chi` because:

* It is lightweight and built directly on top of `net/http`

* It supports clean, expressive routing for dynamic paths

* It provides a composable middleware model suitable for production services

* It is commonly used in infrastructure- and backend-oriented Go services

The goal was not to add features faster, but to **structure the service in a production-friendly way**.


## 2. Separation of Responsibilities

After introducing `chi`, I restructured the project to clearly separate concerns:

```text
cmd/api/main.go        → Application entry point (composition root)
internal/http/router  → HTTP routing and middleware
internal/http/*       → HTTP handlers (health, redirect, etc.)
```

### Key principle

`main.go` should only:

* Read configuration (env vars)

* Initialize dependencies

* Wire components together

* Start and manage the HTTP server lifecycle

It should **not** contain routing rules or handler logic.


## 3. Why main.go Had to Change

Before introducing `chi`, `main.go` was responsible for:

* Creating the router

* Registering routes

* Starting the HTTP server

This tightly coupled application startup with HTTP details.

After refactoring, `main.go` became a **composition root**:

```go
srv := &http.Server{
    Addr:    ":" + port,
    Handler: httpRouter,
}
srv.ListenAndServe()
```

This design makes the system easier to:

* Extend (e.g. add Redis, metrics, graceful shutdown)

* Test (router can be tested independently)

* Modify (router implementation can change without touching startup logic)


## 4. Middleware as a First-Class Concept

Using `chi` made middleware a natural part of the system design rather than an afterthought.
I added production-friendly defaults early:

* Request ID (for tracing)

* Real IP handling

* Panic recovery

* Request timeout

* Structured access logging

This mirrors how real-world services are built and aligns well with Kubernetes environments.


## 5. Kubernetes-Oriented Endpoints

With the router in place, I added Kubernetes-specific endpoints:

* `/healthz` — liveness probe (process-level health)

* `/readyz` — readiness probe (later tied to Redis connectivity)

These endpoints are intentionally simple at first and are designed to evolve as dependencies are added.


## 6. Key Takeaways

* A router library is not about convenience; it is about architecture

* Keeping main.go minimal reduces long-term complexity

* Early separation of routing, handlers, and startup logic pays off as the system grows

* Designing with Kubernetes in mind influences even early code structure