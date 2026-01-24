# url-shortener
A production-style URL shortener built in Go and deployed on Kubernetes.

## Features
- Create short URLs
- Redirect to original URLs
- Redis-backed storage
- Health & readiness probes
- Kubernetes-ready deployment

## Architecture
Client → Ingress → Service → API (Go) → Redis

## Tech Stack
- Go
- Redis
- Docker
- Kubernetes

## Status
Work in progress