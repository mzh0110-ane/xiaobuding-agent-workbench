---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Large Resource Project Requirements

## Purpose

Users continue implementation in a teaching application containing a large question-image library without splitting their project, deleting resources, uploading to GitHub, or preparing Git commands. Reliable collaboration owns this contract because it owns candidate isolation, source evidence, and acceptance.

The observed project contains 153344 tracked entries, including an initial 149752 binary-resource candidates and 3592 remaining source/text entries. Inspection of actual sizes additionally identified large question-bank JSON/JSONL under data/artifacts and EPUB/GZip inputs. The implemented classification accounts for 150081 frozen inputs and 3263 remaining source/text entries. These are admission-inventory observations, not a statement that the project's tests pass. The resource-aware exception extends AC-RELIABLE-LOOP-001.7 and AC-RELIABLE-INSPECT-001.3; small legacy candidates retain their original snapshot algorithm.

### REQ-RELIABLE-RESOURCES-001: Resource-aware isolated collaboration

- **AC-RELIABLE-RESOURCES-001.1:** A clean existing Git project exceeding the full-snapshot tracked-file ceiling, with at most 250000 total entries and 30000 mutable source entries shall be eligible for implementation when its source fits the existing 256 MiB aggregate and 16 MiB per-file limits. Every admitted entry shall be accounted for as mutable source or a frozen resource; classification shall not silently discard entries.
- **AC-RELIABLE-RESOURCES-001.2:** Preparation shall initially materialize mutable source and required input resources without copying the entire resource library. Source files, including SVG, HTML, JavaScript, JSON, scripts and extensionless files, shall remain editable under the audited scope. The original's contents, staged work, history and ignore rules shall retain their existing protections.
- **AC-RELIABLE-RESOURCES-001.3:** An Agent or check may require a deferred resource. The controller shall materialize an independent, verified copy from the frozen local baseline before that resource is used. Unknown, oversized, unsafe or unavailable resources shall produce an explicit reason and shall not be interpreted as successful reads or passing checks. No shared writable file link to the original shall be created.
- **AC-RELIABLE-RESOURCES-001.4:** Existing frozen resources shall remain read-only for the batch. Changed, deleted-after-materialization, type-substituted or redirected resources, a changed resource manifest, or a candidate patch modifying a frozen resource shall block verification or application. New generated assets remain ordinary scoped candidate changes.
- **AC-RELIABLE-RESOURCES-001.5:** Resource hydration alone shall not count as a code change or consume an implementation round. Audit/review may acknowledge only resources verified as available for that invocation. All passing evidence shall bind the resource baseline, mutable source, real checks and fresh-session independent review, while retaining the shared three-round implementation budget.
- **AC-RELIABLE-RESOURCES-001.6:** Cancellation and restart shall retain preparation ownership, the task identity, model choices, source contract and round count. Retry shall not blindly remove or replace an existing candidate. Existing small-project and legacy sealed-candidate behavior shall remain valid.
- **AC-RELIABLE-RESOURCES-001.7:** The existing activity and error surfaces shall distinguish source preparation, deferred resource counts, materialization and genuine verification. Retry of an unchanged admission failure shall not claim progress or reset the budget. The user shall not have to edit Git ignore rules or delete the question bank.

## Exclusions

Automatic acceptance, GitHub publication, editing existing deferred binary resources, OS-enforced protection against a fully privileged native Agent, arbitrary-size repositories, automatic migration of ordinary non-Git folders into the resource profile, and a guarantee of zero model errors remain excluded. Tasks whose purpose is to edit an existing frozen binary resource require a separate explicit scope rather than silently making it writable.
