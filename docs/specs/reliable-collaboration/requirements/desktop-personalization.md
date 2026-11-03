---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Desktop Personalization Requirements

## Overview

The local workspace uses the requested native permission defaults and lets its user personalize the existing desktop without changing tool accounts, task evidence, or collaboration responsibilities.

### REQ-RELIABLE-ACCESS-001: Application-scoped native full access

- **AC-RELIABLE-ACCESS-001.1:** Codex and DeepSeek Harness processes started by this application shall default to their supported full-access permission mode without interactive approval prompts, including investigation, implementation, and independent review dispatches.
- **AC-RELIABLE-ACCESS-001.2:** These defaults shall apply to this application's child-process arguments or temporary invocation overlay. They shall not rewrite either native tool's global configuration, login, account, or unrelated sessions.
- **AC-RELIABLE-ACCESS-001.3:** Investigation and independent review shall retain read-only responsibilities. The controller shall compare admitted project/candidate source around those roles and reject detected source changes instead of treating their output as valid read-only evidence. Full native access shall not remove independent sessions, source/verification checks, the three-round limit, or explicit human acceptance. This is source-change verification, not an operating-system guarantee against all filesystem writes.
- **AC-RELIABLE-ACCESS-001.4:** Application invocations shall reuse the selected Codex and Harness native environments' already installed skills and plugins through their existing discovery mechanisms. Xiaobuding shall not create or install a separate skills/plugins collection or rewrite the native registrations. Effective inheritance shall be verified for both tools rather than inferred from a configured home path.

### REQ-RELIABLE-PERSONALIZATION-001: Local desktop appearance

- **AC-RELIABLE-PERSONALIZATION-001.1:** The existing avatar shall render larger on desktop and phone layouts while preserving its aspect ratio and leaving navigation and task controls unobstructed.
- **AC-RELIABLE-PERSONALIZATION-001.2:** Appearance controls shall offer `none`, `portrait`, and an available `custom` background mode. `portrait` shall use a local copy of the supplied original image without image generation or alteration of the source asset; `none` shall remove that background. With no valid saved preference, the initial mode shall be `portrait`.
- **AC-RELIABLE-PERSONALIZATION-001.3:** A labeled slider shall adjust background transparency from 0% to 100%, where 0% is fully visible and 100% is invisible. A newly selected portrait without a saved value shall start at 80% transparency; changing transparency shall not change foreground opacity.
- **AC-RELIABLE-PERSONALIZATION-001.4:** The chosen background mode and transparency shall persist locally across reloads. Unsupported modes, missing or malformed values, and non-finite values shall receive safe defaults; finite out-of-range values shall be clamped. Unavailable browser storage shall not prevent the controls or task workspace from working.
- **AC-RELIABLE-PERSONALIZATION-001.5:** Light, dark, and system themes and phone layouts shall retain readable content, visible focus, and usable controls with either background mode and at both transparency endpoints. The decorative image shall not intercept input, add page overflow, or change task or account data.
- **AC-RELIABLE-PERSONALIZATION-001.6:** The existing appearance region shall provide a native local image picker for PNG, JPEG and WebP images up to 20 MiB. A valid decoded image shall be selectable, replaceable and restorable after reload without discarding task input or changing theme/transparency. Unsupported, oversized or undecodable images shall leave the current image usable and show a readable status.
- **AC-RELIABLE-PERSONALIZATION-001.7:** Custom images shall remain in this browser's local storage without a server upload or new plugin. Image persistence failure shall permit temporary use with an explicit message rather than claim successful saving. A missing or inaccessible saved custom image shall fall back to the default image while keeping the workspace usable. Replacing or unloading an image shall release unused object URLs.

## Exclusions

Native account migration, global tool permission changes, separate Xiaobuding skills/plugin installations, changes to default models or assignments, image generation, image editing, background upload services, task deletion, weakened evidence/application rules, and claims that full-access read-only roles have operating-system write isolation are excluded.
