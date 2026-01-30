太好了，這一步**真的值得寫成一份完整筆記**。
下面是我幫你整理的 **全新一份筆記**，專門記錄：

* 你做過的 **Ingress 實驗流程**
* 每一步「發生了什麼」
* 你真正學到的 **系統層結論**
* 並且把 *control plane vs data plane*、*hostPort vs Service*、*Ingress vs port-forward* 全部收斂

風格、深度我刻意對齊你之前的筆記，**可以直接放 GitHub，不會太像教科書**。

---

# 📘 Kubernetes Learning Notes 05

## Ingress Experiments: From Rules to Packets (kind)

---

## Motivation

After validating application connectivity using `kubectl port-forward`, I moved on to using **Ingress** to better understand how Kubernetes exposes HTTP services in a production-like way.

During this process, I intentionally ran multiple experiments to answer questions such as:

* Does Ingress create Pods?
* What happens if Ingress rules are deleted?
* What happens if the ingress controller is deleted?
* Why do Services still exist?
* How does traffic actually flow at the kernel level?
* Who is responsible for routing: kube-proxy, CNI, or the controller?

This note documents the **experiments, observations, and final mental model**.

---

## 1. Installing Ingress vs Creating Ingress Rules

### Key Distinction

There are **two completely separate actions**:

1. **Installing an ingress controller**
2. **Creating Ingress resources (rules)**

```text
Ingress Controller = running software (Pods)
Ingress = configuration (rules stored in etcd)
```

### Important Realization

```bash
kubectl apply -f ingress.yaml
```

❌ does NOT create any Pods
✅ only creates routing rules in the API server

The ingress controller Pods were created earlier by applying the official `ingress-nginx` deployment manifest, which includes a Deployment.

---

## 2. Experiment 1: Deleting the Ingress Resource

### Action

```bash
kubectl delete ingress url-shortener-ingress
```

### Observation

* ingress-nginx controller Pod **continued running**
* Traffic to `localhost:8080` returned **404 / default backend**
* No Pods were restarted

### Conclusion

* Ingress is **pure configuration**
* Deleting rules does not affect the controller lifecycle
* Controller still listens, but has no routing rules

---

## 3. Experiment 2: Deleting the Ingress Controller Pod

### Action

```bash
kubectl delete pod -n ingress-nginx ingress-nginx-controller-xxxxx
```

### Observation

* Pod was deleted
* Deployment immediately created a new Pod
* During the gap:

  * traffic briefly failed
* Once the new Pod was Running:

  * traffic resumed automatically
  * no need to re-apply ingress.yaml

### Conclusion

* Ingress rules persist independently in etcd
* Controllers enforce rules dynamically
* **Control plane state survives data plane restarts**

---

## 4. Understanding How Traffic Actually Enters the Cluster

### Initial Entry Point

```text
localhost:8080
 → Docker port mapping
 → kind node:80
```

This step is **outside Kubernetes**.

---

## 5. hostPort Path (Actual Path Used)

Using iptables inspection, the actual routing path was discovered:

```text
node:80
 → CNI hostPort DNAT
 → ingress-nginx Pod:80
```

Evidence from iptables:

```text
CNI-DN-* --dport 80 → DNAT → 10.244.0.5:80
```

### Key Insight

* CNI (portmap plugin) handles `hostPort`
* Traffic is DNAT-ed **directly to the Pod**
* **Service is completely bypassed in this path**

---

## 6. NodePort Path (Present but Not Used)

The ingress-nginx Service exposed NodePorts:

```text
80:31290
443:30946
```

These rules exist:

```text
KUBE-NODEPORTS → Service ClusterIP → Pod
```

However:

* Traffic sent to `localhost:8080` does **not** use NodePort
* NodePort would only be used if accessing `nodeIP:31290`

---

## 7. Service Is Not a Process

A major conceptual shift:

> **Services do not listen on ports.**

Instead:

* Services are implemented via **iptables rules**
* Rules are written by **kube-proxy**
* Packet forwarding happens entirely in the kernel

No user-space proxy is involved.

---

## 8. CNI vs kube-proxy Responsibilities

| Component  | Responsibility                |
| ---------- | ----------------------------- |
| CNI        | Pod networking, hostPort DNAT |
| kube-proxy | Service & NodePort iptables   |
| iptables   | Actual data plane forwarding  |

Observed chains:

* `CNI-*` → hostPort handling
* `KUBE-*` → Service / NodePort handling

---

## 9. Ingress `host:` Does NOT Mean “Listening on Localhost”

Ingress rule:

```yaml
host: localhost
```

### Real Meaning

* Matches **HTTP Host header**
* Does NOT bind to an IP
* Does NOT control where traffic comes from

Example:

```bash
curl http://127.0.0.1:8080
```

Fails unless:

```bash
curl -H "Host: localhost" http://127.0.0.1:8080
```

Ingress operates **after** traffic has already reached the controller.

---

## 10. Why Ingress Still Uses Services

Ingress controller routing flow:

```text
Ingress Controller (nginx)
 → backend Service
 → backend Pod
```

Service is required because:

* Pods are ephemeral
* IPs change
* readiness must be respected
* load balancing is needed

Ingress never routes directly to application Pods.

---

## 11. Comparison: Port-Forward vs Ingress

### Port-Forwarding

```text
localhost
 → kubectl
 → API Server
 → kubelet
 → Pod
```

* Uses control plane
* Debug-only
* Not production-like
* No kernel-level NAT

---

### Ingress

```text
localhost
 → node
 → kernel (iptables)
 → ingress controller
 → Service
 → Pod
```

* Pure data plane
* No API server involvement in traffic
* Production architecture
* Scalable and resilient

---

## 12. Final Mental Model

```text
Ingress        = desired HTTP routing rules
Ingress Ctrl   = enforcement engine
Service        = stable abstraction
NodePort       = external entry option
hostPort       = Pod-centric shortcut
kube-proxy     = rule programmer
CNI            = Pod network setup
iptables       = real packet mover
```

---

## Key Takeaways

* Ingress does not create Pods
* Controllers enforce rules, rules persist independently
* Services are virtual abstractions, not processes
* hostPort and NodePort are different entry paths
* Kubernetes networking is **declarative + event-driven**
* Real traffic flows through the kernel, not the API server