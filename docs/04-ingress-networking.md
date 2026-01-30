# From Port-Forwarding to Ingress: Control Plane vs Data Plane Networking (kind)

## Context

After validating application connectivity using **Service + `kubectl port-forward`**, the next step was to expose the application using **Ingress**.

This introduced a series of questions:

* Why Ingress does not create Pods
* Why Ingress still requires Services
* Why NodePort still exists with Ingress
* How traffic actually flows from `localhost` to Pods
* Which component (kube-proxy, CNI, controller) is responsible for each step
* Why iptables rules appear even though no process is listening on ports

This document records the **full mental model** from Ingress down to kernel-level packet forwarding, and compares it with port-forwarding.

---

## 1. What Ingress Is (and What It Is Not)

### Ingress Is a Rule, Not a Process

An `Ingress` resource:

* Is a **Kubernetes API object**
* Stores HTTP routing rules (host/path → Service)
* Is persisted in **etcd**
* **Does not create Pods**
* **Does not listen on ports**

```yaml
kind: Ingress
```

Ingress is purely **declarative configuration**.

---

## 2. The Role of the Ingress Controller

Ingress rules are useless by themselves.

An **Ingress Controller** (e.g. ingress-nginx):

* Runs as a **Pod**
* Watches the Kubernetes API Server
* Reads Ingress resources
* Translates them into live proxy configuration (e.g. nginx.conf)
* Reloads nginx **gracefully** when rules change

Key separation:

| Component          | Responsibility         |
| ------------------ | ---------------------- |
| Ingress            | Desired routing rules  |
| Ingress Controller | Enforces routing rules |

---

## 3. Why Services Still Exist with Ingress

Ingress **never routes directly to Pods**.

Instead:

```
Ingress → Service → Pod
```

Reasons:

* Pods are ephemeral (IP changes, restarts, scaling)
* Services provide:

  * Stable virtual IP (ClusterIP)
  * Readiness-aware load balancing
  * Label-based selection

Ingress depends on Services as **stable backends**.

---

## 4. Service Is Not a Process

A critical realization:

> **A Service does not listen on a port and does not run a process.**

Instead:

* Services are implemented using **iptables / IPVS rules**
* These rules are programmed by **kube-proxy**
* Packet forwarding happens entirely in the **Linux kernel**

There is no user-space proxy involved.

---

## 5. NodePort Is Not an Alternative to Ingress

A common misconception is that NodePort and Ingress are mutually exclusive.

They are not.

### Layered responsibilities:

| Layer                   | Purpose                        |
| ----------------------- | ------------------------------ |
| NodePort / LoadBalancer | How traffic enters the cluster |
| Ingress Controller      | HTTP termination & routing     |
| Ingress                 | Routing rules                  |
| Service                 | Stable backend                 |
| Pod                     | Application                    |

Ingress **always sits behind some external entry mechanism**.

---

## 6. Why ingress-nginx Shows `type: LoadBalancer`

In a kind / local environment:

```text
TYPE: LoadBalancer
EXTERNAL-IP: <pending>
PORT(S): 80:31290, 443:30946
```

Explanation:

* `LoadBalancer` expresses **intent**
* No cloud provider exists → External IP stays `<pending>`
* Kubernetes still allocates **NodePorts**
* NodePort is the **actual data-plane entry**

LoadBalancer Services are **implemented on top of NodePort**.

---

## 7. Two Ways Traffic Can Reach the Ingress Controller

### A. NodePort Path

```
Client → NodeIP:31290 → kube-proxy (iptables) → ingress-nginx Service → Pod
```

* Implemented via `KUBE-NODEPORTS` rules
* Used when accessing `<node-ip>:31290`

---

### B. hostPort Path (Used in This Setup)

In this environment:

```
localhost:8080 → Docker port mapping → node:80
→ CNI hostPort DNAT → ingress-nginx Pod:80
```

Evidence from iptables:

```text
CNI-DN-* --dport 80 → DNAT → 10.244.0.5:80
```

Key insight:

* **This path does NOT use NodePort**
* It is implemented by the **CNI portmap plugin**
* The destination Pod is the ingress controller

---

## 8. kube-proxy vs CNI: Clear Responsibility Split

| Component  | Responsibility                 |
| ---------- | ------------------------------ |
| kube-proxy | Service & NodePort iptables    |
| CNI        | Pod networking, hostPort rules |
| iptables   | Actual packet forwarding       |

iptables chains observed:

* `KUBE-*` → kube-proxy
* `CNI-*` → CNI plugin

---

## 9. End-to-End Traffic Flow (Current Setup)

```
Client
 → localhost:8080
 → Docker port mapping
 → kind node:80
 → CNI hostPort DNAT
 → ingress-nginx Pod (nginx)
 → Ingress rules
 → url-shortener Service (ClusterIP)
 → Pod:8080
```

No port-forwarding, no API server proxying — **pure data plane**.

---

## 10. How This Differs from Port-Forwarding

### Port-Forwarding Characteristics

```bash
kubectl port-forward svc/url-shortener 8080:80
```

Traffic path:

```
localhost
 → kubectl
 → API Server
 → kubelet
 → Pod
```

Key properties:

* Uses **control plane**
* Debug-only
* No ingress controller
* No kernel-level forwarding
* Not representative of production traffic

---

### Ingress Characteristics

```
localhost → node → kernel → ingress → service → pod
```

Key properties:

* Uses **data plane**
* No API server involvement in traffic
* Scales naturally
* Matches real production architecture
* Host/path-based routing

---

## 11. Control Plane vs Data Plane Summary

| Aspect              | Port-Forwarding | Ingress |
| ------------------- | --------------- | ------- |
| Uses API Server     | Yes             | No      |
| Uses kernel NAT     | No              | Yes     |
| Requires controller | No              | Yes     |
| Production-ready    | No              | Yes     |
| Debug-friendly      | Yes             | Less    |

---

## 12. Key Takeaways

* Ingress is configuration, not a server
* Services are virtual abstractions, not processes
* kube-proxy and CNI program kernel forwarding rules
* NodePort and LoadBalancer are entry mechanisms, not alternatives to Ingress
* hostPort and NodePort are separate paths
* Port-forwarding validates correctness, Ingress validates architecture
