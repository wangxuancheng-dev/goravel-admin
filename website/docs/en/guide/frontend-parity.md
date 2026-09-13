# Dual frontend parity (Vue / React)

This repo ships two admin UIs against the **same** `/api/admin` (and platform `/api/platform`) contract.

## Frontend roles

| Role | Directory | Notes |
|------|-----------|--------|
| **Primary UI** | `html-react/` (React 19) | Default production frontend; docs screenshots, Docker `BUILD_FRONTEND=1` → `public/admin`. **New features land on React first** |
| **Peer UI** | `html/` (Vue 3) | Same-repo Vue reference; separate CI type-check / test / build. Keep API/permission parity with React |

Prefer React (`html-react/`) first, then follow up Vue (`html/`). Same PR when practical; otherwise call out the follow-up in the PR.

## Change checklist

- [ ] API path / query / body matches backend
- [ ] Permission slugs / menu routes (React first is OK)
- [ ] Tenant header / query parity
- [ ] i18n keys (or backend `error_code`)
- [ ] Module flags aligned with backend
- [ ] Codegen: prefer React first (`CODE_GENERATOR_FRONTEND`, suggested default `react,vue`); Vue templates same batch or follow-up
- [ ] Shared util: minimal vitest on both sides

See also: [Architecture](/en/guide/architecture), [Production](/en/deploy/production).
