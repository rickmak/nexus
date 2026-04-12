# Nexus Roadmap

## Multi-User, Multi-Tenant Architecture

**Status:** Phase 1 Complete  
**Last Updated:** 2026-04-12  
**PR:** [#33 - Multi-User Architecture Preparation](https://github.com/IniZio/nexus/pull/33)

---

### Vision

Enable Nexus to operate as a **hybrid architecture** supporting:

1. **Personal Daemon** (`mode: "personal"`): Single-user daemon on laptop/PC (current)
2. **Pool Daemon** (`mode: "pool"`): Multi-user daemon serving many users (future)
3. **Federation**: Cross-daemon workspace sharing (future)

---

### Phase 1: Data Model Preparation ✅ COMPLETE

**Deliverables:**
- Auth package with `Identity` struct and `Provider` interface
- Self-managed token generation in daemon
- Workspace ownership fields (`OwnerUserID`, `TenantID`)
- Context propagation through RPC handlers
- Future-proof configuration (`AuthConfig`)

**Impact:** ~600 lines, zero behavior change

---

### Phase 2: Pool Mode with OIDC 🔮 PLANNED

**Deliverables:**
- `OIDCProvider` implementation
- Device Code flow for CLI
- Token refresh handling
- Workspace ownership enforcement
- Pool mode configuration

---

### Phase 3: Cross-Daemon Federation 🔮 PLANNED

**Deliverables:**
- Federation protocol
- Short-lived federation tokens
- Remote workspace proxy
- Cross-daemon sharing

---

### Phase 4: Multi-Tenancy & RBAC 🔮 FUTURE

**Deliverables:**
- Tenant isolation
- Role definitions (Admin, Developer, Viewer)
- Fine-grained permissions
- Audit logging

---

## Documentation

- **Design Spec:** `docs/superpowers/specs/2026-04-12-multi-user-multi-tenant-architecture-design.md`
- **Implementation Plan:** `docs/superpowers/plans/2026-04-12-multi-user-prep-implementation.md`
