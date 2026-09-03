# k8s-remedy-operator

A Kubernetes operator that detects common production failure signatures and applies safe, automated remediation — with the guardrails a real production system needs: dry-run mode, rate limiting, rollback, and full audit trails.

Built to demonstrate senior-level Kubernetes controller design: custom resources, the reconciliation loop, leader election, observability integration, and safe automation of operational actions.

---

## Table of Contents

- [Motivation](#motivation)
- [Features](#features)
- [Architecture Overview](#architecture-overview)
- [Directory Structure](#directory-structure)
- [Custom Resources](#custom-resources)
- [Safety Model](#safety-model)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Observability](#observability)
- [Roadmap](#roadmap)
- [Design Docs](#design-docs)

---

## Motivation

SREs spend a large share of their time on repetitive, well-understood remediation: bouncing a crash-looping pod, rotating a near-expiry cert, evicting a node under memory pressure. This operator encodes that operational knowledge as code, running inside the cluster it protects, so common incidents are detected and remediated automatically — safely, observably, and reversibly — before they page a human.

---

## Features

### Detection

- [ ] **Crash-loop detection** — watches `Pod` restart counts and back-off state, flags pods exceeding a configurable restart threshold within a rolling window.
- [ ] **Memory pressure detection** — tracks pod/container memory usage against requests/limits (via metrics-server or Prometheus) and flags sustained pressure before OOMKill.
- [ ] **Certificate expiry detection** — scans `Secret`s of type `kubernetes.io/tls` (and cert-manager `Certificate` resources) for upcoming expiry.
- [ ] **Dependency timeout detection** — consumes readiness/liveness probe failures and upstream error-rate signals (via Prometheus queries) to flag services failing due to downstream dependency issues, not their own code.
- [ ] **Pluggable detector interface** — new failure signatures can be added by implementing a single `Detector` interface, no core changes required.

### Remediation

- [ ] **Playbook-as-code** — remediation actions defined as declarative YAML "playbooks" (trigger condition → action → verification step), version-controlled alongside application manifests.
- [ ] **Built-in actions**: pod restart, deployment rollback to last-known-good revision, HPA scale-adjust, cert renewal trigger, node cordon/drain.
- [ ] **Custom action plugins** — actions implemented as a Go interface, loadable without modifying the controller core.
- [ ] **Pre/post verification** — each remediation defines a health check that must pass after the action; failed verification triggers rollback of the remediation itself.

### Safety Rails

- [ ] **Dry-run mode** — cluster-wide or per-playbook flag that logs and emits events for the action the operator _would_ take, without executing it.
- [ ] **Rate limiting** — per-resource and global limits on remediation actions per time window, preventing remediation storms during widespread outages.
- [ ] **Circuit breaker** — automatically disables a playbook after N consecutive failed remediations or verification failures, requiring manual re-enable.
- [ ] **Blast-radius caps** — hard ceiling on percentage of a workload's replicas that can be touched by automated action in a single window.
- [ ] **Rollback** — every remediation action records enough state (previous replica count, previous image tag, previous config) to be reversed with a single command.
- [ ] **Human-in-the-loop mode** — optional approval gate (via annotation, Slack, or webhook) required before higher-risk actions execute.
- [ ] **Full audit log** — every detection, decision, and action (or non-action, with reason) recorded as both a Kubernetes `Event` and a structured log line.

### Observability & Integration

- [ ] **Prometheus metrics** — exposes counts of detections, remediations attempted/succeeded/failed/rolled-back, and current circuit-breaker states.
- [ ] **Alertmanager integration** — can both consume firing alerts as a detection source and emit its own alerts when a playbook is disabled or a remediation fails.
- [ ] **Grafana dashboard** (provided) — remediation activity, success rate, and rate-limit utilization at a glance.
- [ ] **Structured logging** — JSON logs with correlation IDs so a single incident's detection → decision → action → verification chain can be traced end to end.

### Platform

- [ ] **CRD-based configuration** — `RemediationPolicy` and `Playbook` custom resources; no ConfigMap-editing or restarts required to change behavior.
- [ ] **Leader election** — safe to run multiple replicas for HA; only the leader executes actions.
- [ ] **Namespace scoping** — can be restricted to specific namespaces or run cluster-wide.
- [ ] **RBAC least-privilege** — ships with a minimal `ClusterRole` scoped to exactly the verbs/resources each enabled detector/action needs.

---

## Architecture Overview

```
                     ┌────────────────────────────────────────┐
                     │              Kubernetes API            │
                     └───────────────┬────────────────────────┘
                                     │ watch/list
                     ┌───────────────▼──────────────────────────┐
                     │            Remedy Operator (Pod)         │
                     │                                          │
   Prometheus  ─────▶│  ┌────────────┐   ┌──────────────────┐   │
   Alertmanager─────▶│  │ Detectors  │──▶│  Decision Engine  │  │
                     │  └────────────┘   │  (rate limit,     │  │
                     │                   │   circuit breaker,│  │
                     │                   │   dry-run check)  │  │
                     │                   └─────────┬─────────┘  │
                     │                              │           │
                     │                    ┌─────────▼─────────┐ │
                     │                    │   Action Executor │ │
                     │                    │  (built-in/plugin)│ │
                     │                    └─────────┬─────────┘ │
                     │                              │           │
                     │                    ┌─────────▼─────────┐ │
                     │                    │ Verifier/Rollback │ │
                     │                    └───────────────────┘ │
                     └──────────────────────────────────────────┘
                                     │
                          Events, Metrics, Audit Log
```

Core reconciliation follows the standard controller pattern: **Observe** (detectors watch cluster + metrics state) → **Diff** (decision engine compares against `RemediationPolicy` thresholds) → **Act** (executor runs the playbook action) → **Verify** (post-check confirms health, or triggers rollback).

---

## Directory Structure

```
k8s-remedy-operator/
├── cmd/
│   └── manager/
│       └── main.go                 # Entry point: manager setup, leader election, flag parsing
│
├── api/
│   └── v1alpha1/
│       ├── remediationpolicy_types.go   # RemediationPolicy CRD schema
│       ├── playbook_types.go            # Playbook CRD schema
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go     # controller-gen generated
│
├── internal/
│   ├── controller/
│   │   ├── remediationpolicy_controller.go   # Main reconciliation loop
│   │   ├── playbook_controller.go
│   │   └── suite_test.go                     # envtest-based controller tests
│   │
│   ├── detectors/
│   │   ├── detector.go              # Detector interface + registry
│   │   ├── crashloop/
│   │   │   └── crashloop.go
│   │   ├── memorypressure/
│   │   │   └── memorypressure.go
│   │   ├── certexpiry/
│   │   │   └── certexpiry.go
│   │   └── dependencytimeout/
│   │       └── dependencytimeout.go
│   │
│   ├── actions/
│   │   ├── action.go                # Action interface + registry
│   │   ├── restart/
│   │   ├── rollback/
│   │   ├── scale/
│   │   ├── certrenew/
│   │   └── cordon/
│   │
│   ├── engine/
│   │   ├── decision.go              # Rate limiting, circuit breaker, dry-run logic
│   │   ├── ratelimiter.go
│   │   ├── circuitbreaker.go
│   │   └── blastradius.go
│   │
│   ├── verifier/
│   │   └── verifier.go              # Post-action health verification
│   │
│   ├── audit/
│   │   └── audit.go                 # Structured audit logging + K8s Event emission
│   │
│   └── metrics/
│       └── metrics.go               # Prometheus metric definitions
│
├── config/
│   ├── crd/
│   │   └── bases/                   # Generated CRD manifests
│   ├── rbac/
│   │   ├── role.yaml                # Least-privilege ClusterRole
│   │   └── role_binding.yaml
│   ├── manager/
│   │   └── manager.yaml             # Operator Deployment manifest
│   ├── samples/
│   │   ├── remediationpolicy_sample.yaml
│   │   └── playbook_crashloop.yaml
│   └── prometheus/
│       ├── servicemonitor.yaml
│       └── alerting_rules.yaml
│
├── playbooks/                       # Example playbook-as-code library
│   ├── crashloop-restart.yaml
│   ├── memory-pressure-scale.yaml
│   ├── cert-expiry-renew.yaml
│   └── dependency-timeout-circuitbreak.yaml
│
├── dashboards/
│   └── grafana/
│       └── remedy-operator-overview.json
│
├── test/
│   ├── e2e/
│   │   └── remediation_e2e_test.go  # Kind-based end-to-end tests
│   └── chaos/
│       └── failure_injection.go     # Deliberately induces crash-loops/OOMs for testing
│
├── docs/
│   ├── design-doc.md                # Full design doc: goals, tradeoffs, alternatives considered
│   ├── safety-model.md              # Deep dive on rate limiting, circuit breaker, blast radius
│   └── runbook-authoring-guide.md   # How to write a new Playbook CR
│
├── hack/
│   └── kind-cluster.yaml            # Local dev cluster config
│
├── Makefile                         # build, test, deploy, generate targets
├── Dockerfile
├── go.mod
├── go.sum
├── PROJECT                          # kubebuilder project marker
└── README.md
```

---

## Custom Resources

**`RemediationPolicy`** — cluster or namespace-scoped policy defining which detectors are enabled, their thresholds, and global safety limits (rate limits, blast-radius caps, dry-run flag).

**`Playbook`** — a single trigger → action → verification definition. Multiple playbooks can be bound to one policy.

```yaml
apiVersion: remedy.io/v1alpha1
kind: Playbook
metadata:
  name: crashloop-restart
spec:
  trigger:
    detector: crashloop
    threshold:
      restartCount: 5
      window: 10m
  action:
    type: restart
    maxAffectedPercent: 20
  verification:
    healthCheck: readinessProbe
    timeout: 2m
  safety:
    dryRun: false
    requireApproval: false
```

---

## Safety Model

This is the part of the project that actually matters for a production system — automating remediation without safety rails is how you turn a small incident into an outage. See [`docs/safety-model.md`](docs/safety-model.md) for the full writeup, but at a glance:

| Mechanism               | Purpose                                                                         |
| ----------------------- | ------------------------------------------------------------------------------- |
| Dry-run mode            | Validate detection + decision logic against real traffic with zero blast radius |
| Rate limiting           | Cap actions per resource/time window to prevent remediation storms              |
| Circuit breaker         | Auto-disable a playbook after repeated failures                                 |
| Blast-radius cap        | Limit % of a workload touched by one automated decision                         |
| Verification + rollback | Every action is checked and reversible                                          |
| Human-in-the-loop       | Optional approval gate for high-risk actions                                    |

---

## Getting Started

```bash
# Clone
git clone https://github.com/mjthecoder65/k8s-remedy-operator.git
cd k8s-remedy-operator

# Local dev cluster
kind create cluster --config hack/kind-cluster.yaml

# Install CRDs
make install

# Run the operator locally against the cluster
make run

# Apply a sample policy + playbook
kubectl apply -f config/samples/
```

---

## Configuration

Operator-wide flags (set via Deployment env vars or CLI flags):

| Flag                  | Default | Description                                                                           |
| --------------------- | ------- | ------------------------------------------------------------------------------------- |
| `--dry-run`           | `true`  | Global override; forces all playbooks into dry-run regardless of per-playbook setting |
| `--metrics-addr`      | `:8080` | Prometheus metrics endpoint                                                           |
| `--leader-elect`      | `true`  | Enable leader election for HA deployments                                             |
| `--global-rate-limit` | `10/5m` | Max total remediation actions cluster-wide per window                                 |

---

## Observability

- Prometheus metrics exposed at `/metrics` (detections, remediations, verification outcomes, circuit-breaker state)
- Sample Grafana dashboard in `dashboards/grafana/`
- Every remediation emits a Kubernetes `Event` on the affected resource, visible via `kubectl describe`

---

## Roadmap

- [ ] Phase 1 — Core controller scaffold, CRDs, crash-loop detector + restart action, dry-run mode
- [ ] Phase 2 — Rate limiter, circuit breaker, blast-radius cap, audit logging
- [ ] Phase 3 — Memory pressure + cert expiry + dependency timeout detectors
- [ ] Phase 4 — Prometheus/Alertmanager integration, Grafana dashboard
- [ ] Phase 5 — Chaos test suite (deliberate failure injection), e2e test suite on Kind
- [ ] Phase 6 — Human-in-the-loop approval flow (Slack/webhook)
- [ ] Phase 7 — Design doc + load test writeup + demo recording

---

## Design Docs

- [`docs/design-doc.md`](docs/design-doc.md) — goals, non-goals, alternatives considered, tradeoffs
- [`docs/safety-model.md`](docs/safety-model.md) — deep dive on the safety mechanisms above
- [`docs/runbook-authoring-guide.md`](docs/runbook-authoring-guide.md) — how to write a new `Playbook`
