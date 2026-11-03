---
id: "01-local-background"
title: "Deliver local image selection and the replacement default"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-PERSONALIZATION-001
  - REQ-RELIABLE-MODEL-001
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-personalization.md
---

# Deliver the Local Background Picker

Implement native selection, decoded image validation, one browser-local custom Blob, temporary object URLs, persistence status and safe recovery in the existing background panel. Use the latest supplied image as the bundled default without editing it. Follow the plan's ASCII control grouping and existing desktop/phone surfaces.

Acceptance is owned by PERSONALIZATION-001.2 and .6-.7. The four added browser scenarios cover picker/reload/privacy, replacement and phone access, invalid inputs, and storage-failure recovery. Shared theme/transparency/task state behavior remains covered by the retained scenarios and four state tests.

Run `node scripts/reliable/desktop-browser-check.cjs`, `node --test apps/backend/internal/reliable/page-state.test.cjs`, the existing Windows build and bounded documentation checks. Then verify the actual candidate and production image/picker, the stopped task snapshot and pending sealed evidence, without invoking a model or applying a candidate. Store delivery results in the plan after completing them.

The scope also includes a stale Codex catalog correction: opening assignment controls reuses authenticated refresh when the configured Codex model is unverified or its directory failed. A nineteenth browser scenario proves that fresh native reasoning options replace the placeholder while retaining input and selections; it does not invent support for an unlisted model. The installed native cache currently exposes low/medium/high/xhigh/max/ultra for gpt-6.1-sol. Actual model execution is outside this UI check.

The final scope targets 1.5.3. The additional native-history fix covers MODEL-001.7: retain only actually observed same-profile capabilities when the configured model is omitted; label them historical and let fresh native metadata replace old values. Two backend regression tests cover repeated restart/omission/replacement and profile isolation. The UI does not fabricate capabilities or promise entitlement. Initial 1.5.2 production verification reproduced cache omission after its candidate check; final delivery must therefore be verified again with the history path.

Final delivery: 1.5.3 candidate and production checks passed, including custom-image reload and ultra selection for backend, planning and review on gpt-6.1-sol. All 19 browser scenarios and four state tests passed. Catalog/history/selection race tests passed in 5.496 seconds; vet and lint passed with zero lint issues. Five production tasks and one pending seal were checked without applying changes (3.512 seconds). Metadata validated 351 decisions and 1354 specifications; owning findings are zero and 107 upstream findings are unchanged. No paid model request or full upstream/core regression rerun is claimed. Backup: E:\小布丁\backups\upgrade-1.5.3-20261019-193031. Evidence spans upgrade-1.5.2 and upgrade-1.5.3. The default PNG matches the original SHA-256 d8ead2a73e1083334dde7099f321463816bf1756c54c94314cfa024a0ca252b0.