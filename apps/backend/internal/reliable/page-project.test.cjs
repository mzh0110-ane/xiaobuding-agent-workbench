const { test } = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { join } = require('node:path');
const vm = require('node:vm');

function fixture() {
  const context = vm.createContext({ TextDecoder, Uint8Array, Error });
  vm.runInContext(readFileSync(join(__dirname, 'page-project.js'), 'utf8') + '\nglobalThis.project = ProjectUI;', context);
  return context.project;
}

test('project counts only server-confirmed verification and acceptance', () => {
  const ui = fixture();
  const counts = ui.counts({ items: [{ status: 'planned' }, { status: 'in_progress' }, { status: 'verified' }, { status: 'accepted' }, { status: 'unknown' }] });
  assert.equal(counts.total, 5); assert.equal(counts.verified, 2); assert.equal(counts.accepted, 1);
});

test('native planning provenance remains a suggestion until explicitly confirmed', () => {
  const ui = fixture();
  assert.equal(ui.suggested({ source: 'plan:project-plan-native-message' }), true);
  assert.equal(ui.suggested({ source: 'user' }), false); assert.equal(ui.suggested({ source: 'task' }), false);
});

test('project evidence accepts scoped evidence files and rejects traversal or absolute URLs', () => {
  const ui = fixture();
  for (const path of ['round-1/check-1.log', 'round-2-review.json']) assert.equal(ui.safeEvidencePath(path), true);
  for (const path of ['../config.toml', '/round-1/log', 'https://evil.invalid/a', 'round-1\\log', 'round-1/../secret', '']) assert.equal(ui.safeEvidencePath(path), false);
});

test('bounded patch loading cancels the response without displaying an oversized body', async () => {
  const ui = fixture(); let cancelled = false;
  const response = { ok: true, headers: { get: () => null }, body: { getReader: () => ({ read: async () => ({ done: false, value: new Uint8Array(512 * 1024 + 1) }), cancel: async () => { cancelled = true; }, releaseLock() {} }) } };
  await assert.rejects(ui.readPatch(response), /较大/); assert.equal(cancelled, true);
});

test('patch text remains bounded and decoded without HTML interpretation', async () => {
  const ui = fixture(); const bytes = new TextEncoder().encode('+<script>example</script>\n-旧内容'); let sent = false;
  const response = { ok: true, headers: { get: () => null }, body: { getReader: () => ({ read: async () => sent ? { done: true } : (sent = true, { value: bytes, done: false }), cancel: async () => {}, releaseLock() {} }) } };
  assert.equal(await ui.readPatch(response), '+<script>example</script>\n-旧内容');
});

test('small patch bytes with many lines cannot create an unbounded diff DOM', () => {
  const ui = fixture(), displayed = ui.patchLines('+line\n'.repeat(5000));
  assert.equal(displayed.lines.length, 3000); assert.equal(displayed.limited, true);
  assert.equal(ui.patchLines('+one\n-two').limited, false);
});

test('project short labels preserve the complete requirement for detail and dispatch', () => {
  const ui = fixture(), original = '完整需求'.repeat(100), item = { title: original };
  assert.ok(Array.from(ui.itemTitle(item)).length <= 53);
  assert.equal(item.title, original);
  assert.equal(ui.itemTitle({ ...item, short_title: '导出 PDF' }), '导出 PDF');
  assert.equal(ui.itemTitle({ title: 'Readable fallback', short_title: '  ' }), 'Readable fallback');
});

test('project editing never rewrites assigned or historical evidence', () => {
  const ui = fixture(), pending = { id: 'item-a', source: 'user', status: 'planned' };
  assert.equal(ui.itemEditable(pending, {}), true);
  assert.equal(ui.itemEditable({ ...pending, status: 'deferred' }, {}), true);
  assert.equal(ui.itemEditable({ ...pending, task_id: 'batch' }, {}), false);
  assert.equal(ui.itemEditable({ ...pending, source: 'task', id: 'batch-x' }, {}), false);
  assert.equal(ui.itemEditable(pending, { archived_at: 'saved' }), false);
  assert.equal(ui.itemEditable({ ...pending, status: 'verified' }, {}), false);
});

test('project ordering moves only saved IDs and retains every saved item', () => {
  const ui = fixture(), record = { items: [{ id: 'a', source: 'user' }, { id: 'batch-old', source: 'task' }, { id: 'b', source: 'plan:x' }, { id: 'c', source: 'user' }] };
  assert.equal(JSON.stringify(ui.movedItemOrder(record, 'b', -1)), '["b","a","c"]');
  assert.equal(JSON.stringify(ui.movedItemOrder(record, 'b', 1)), '["a","c","b"]');
  assert.equal(ui.movedItemOrder(record, 'a', -1), null);
  assert.equal(ui.movedItemOrder(record, 'batch-old', 1), null);
  assert.equal(record.items[0].id, 'a');
});

test('dirty project editor input survives refreshed records until explicit reset', () => {
  const ui = fixture(), item = { id: 'a', title: 'Original', acceptance: 'Opens', status: 'planned', depends_on: ['b'] };
  const initial = ui.reconcileItemDraft(null, item, 3);
  initial.short_title = 'My unsaved input'; initial.dirty = true;
  const refreshed = ui.reconcileItemDraft(initial, { ...item, title: 'New server title' }, 4);
  assert.equal(refreshed, initial);
  assert.equal(refreshed.short_title, 'My unsaved input');
  assert.equal(refreshed.base_revision, 3);
  const reset = ui.reconcileItemDraft(null, { ...item, title: 'New server title' }, 4);
  assert.equal(reset.title, 'New server title'); assert.equal(reset.base_revision, 4);
  reset.depends_on.push('c'); assert.equal(item.depends_on.length, 1);
});

test('project editor payload carries its original revision and cannot claim verification', () => {
  const ui = fixture(), draft = { id: 'a', title: ' Full requirement ', short_title: ' Label ', acceptance: ' Opens ', milestone: ' First ', depends_on: ['b'], disposition: 'deferred', reason: ' Waiting ', base_revision: 7, status: 'accepted', verified: true, evidence: ['fake'] };
  const payload = ui.itemEditPayload(draft);
  assert.equal(payload.expected_revision, 7);
  assert.equal(payload.item_edit.title, 'Full requirement');
  assert.equal(payload.item_edit.short_title, 'Label');
  assert.equal(payload.item_edit.reason, 'Waiting');
  assert.equal(Object.hasOwn(payload.item_edit, 'verified'), false);
  assert.equal(Object.hasOwn(payload.item_edit, 'status'), false);
  assert.equal(Object.hasOwn(payload.item_edit, 'evidence'), false);
});
