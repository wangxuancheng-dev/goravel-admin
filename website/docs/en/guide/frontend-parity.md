# Dual frontend parity (Vue / React)

This repo ships two admin UIs against the **same** `/api/admin` (and platform `/api/platform`) contract.

## Primary vs peer

| Role | Directory | Notes |
|------|-----------|--------|
| **Primary shipping UI** | `html/` (Vue 3) | Default production frontend; docs screenshots, Docker `BUILD_FRONTEND=1`, codegen Vue templates |
| **Peer showcase** | `html-react/` (React 19) | Same-repo React showcase; separate CI type-check / test / build. **New features land on Vue first** |

Prefer Vue (`html/`) first, then follow up React (`html-react/`). Same PR when practical; otherwise call out the follow-up in the PR.

## Change checklist

- [ ] API path / query / body matches backend
- [ ] Permission slugs / menu routes (Vue first is OK)
- [ ] Tenant header / query parity
- [ ] i18n keys (or backend `error_code`)
- [ ] Module flags aligned with backend
- [ ] Codegen: Vue first; React templates same batch or follow-up
- [ ] Shared util: minimal vitest on both sides

See also: [Architecture](/en/guide/architecture), [Production](/en/deploy/production).
