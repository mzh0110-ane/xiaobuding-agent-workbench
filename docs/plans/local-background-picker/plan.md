---
created: 2026-10-19
status: complete
requirements:
  - REQ-RELIABLE-PERSONALIZATION-001
  - REQ-RELIABLE-MODEL-001
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-personalization.md
legacy_specs: []
---

# Local Background Picker

Allow this user to replace the background directly from the existing appearance controls. Version 1.5.2 also replaces the bundled default with the latest supplied PNG, byte-for-byte. Existing theme, transparency, subject-preserving layout, agent settings and task evidence remain independent.

```text
Background settings
Image         [ Default / Custom / None ]
[ Choose local image ]
Images stay in this browser. Maximum 20 MB.
Save/error status
Transparency  [------o---] 80%
```

The controls remain in the phone task drawer. Reuse native file selection and browser Blob storage; do not introduce an upload service, a plugin or a second image library.

## Work order

- [x] [Deliver the local picker](task-01-local-background.md)

## Verification

The canonical browser suite retains its 14 original scenarios and adds four: native picker and reload without uploads or lost input; replacement/default/none and phone access; invalid/oversized/corrupt input; unavailable storage and missing custom image fallback. These own PERSONALIZATION-001.6 and .7; existing image, theme, keyboard and layout checks cover the shared behavior.

Verify actual candidate/production default image bytes against the supplied image and exercise custom-image persistence. Preserve the five production tasks and sealed candidate without applying it. Keep RED/GREEN, browser/runtime evidence and normal local commit receipt in `data/validation/upgrade-1.5.2` outside the source tree. Do not rerun unrelated core backend or upstream checks for this appearance-only increment.

## Results

Implementation and final 1.5.3 delivery passed.

The scope also includes a stale Codex catalog correction: opening assignment controls reuses authenticated refresh when the configured Codex model is unverified or its directory failed. A nineteenth browser scenario proves that fresh native reasoning options replace the placeholder while retaining input and selections; it does not invent support for an unlisted model. The installed native cache currently exposes low/medium/high/xhigh/max/ultra for gpt-6.1-sol. Actual model execution is outside this UI check.

The final scope targets 1.5.3. The additional native-history fix covers MODEL-001.7: retain only actually observed same-profile capabilities when the configured model is omitted; label them historical and let fresh native metadata replace old values. Two backend regression tests cover repeated restart/omission/replacement and profile isolation. The UI does not fabricate capabilities or promise entitlement. Initial 1.5.2 production verification reproduced cache omission after its candidate check; final delivery must therefore be verified again with the history path.

Final delivery: 1.5.3 candidate and production checks passed, including custom-image reload and ultra selection for backend, planning and review on gpt-6.1-sol. All 19 browser scenarios and four state tests passed. Catalog/history/selection race tests passed in 5.496 seconds; vet and lint passed with zero lint issues. Five production tasks and one pending seal were checked without applying changes (3.512 seconds). Metadata validated 351 decisions and 1354 specifications; owning findings are zero and 107 upstream findings are unchanged. No paid model request or full upstream/core regression rerun is claimed. Backup: E:\小布丁\backups\upgrade-1.5.3-20261019-193031. Evidence spans upgrade-1.5.2 and upgrade-1.5.3. The default PNG matches the original SHA-256 d8ead2a73e1083334dde7099f321463816bf1756c54c94314cfa024a0ca252b0.