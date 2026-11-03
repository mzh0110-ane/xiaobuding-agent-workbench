const { test } = require('node:test');
const assert = require('node:assert/strict');
const { existsSync, readFileSync } = require('node:fs');
const { join } = require('node:path');
const vm = require('node:vm');

function fixture(file = 'page-budget.js', name = 'StorageBudgetUI') {
  const path = join(__dirname, file), context = vm.createContext({ matchMedia: () => ({}) });
  const source = existsSync(path) ? readFileSync(path, 'utf8') : '';
  vm.runInContext(source + `\nglobalThis.subject = typeof ${name} === 'undefined' ? {} : ${name};`, context);
  return context.subject;
}

test('storage budget payload converts bounded GB values to exact backend byte limits', () => {
  const ui = fixture();
  assert.equal(typeof ui.payload, 'function', 'storage budget controls do not yet validate the API payload');
  const payload = ui.payload({ revision: 3, cache: '8', temp: '2.5', free: '0.0625' });
  assert.equal(payload.expected_revision, 3);
  assert.equal(payload.cache_limit_bytes, 8 * 1024 ** 3);
  assert.equal(payload.temp_limit_bytes, 2.5 * 1024 ** 3);
  assert.equal(payload.minimum_free_bytes, 64 * 1024 ** 2);
});

test('storage budget validation rejects blanks, infinity, hidden formats and out-of-range limits', () => {
  const ui = fixture();
  assert.equal(typeof ui.payload, 'function');
  for (const value of ['', ' ', 'Infinity', 'NaN', '0', '-1', '0.01', '1024.1', '0x10', '1e3', '8,0']) {
    assert.throws(() => ui.payload({ revision: 0, cache: value, temp: '2', free: '2' }), /缓存|额度/);
  }
  assert.throws(() => ui.payload({ revision: -1, cache: '8', temp: '2', free: '2' }), /读取|版本/);
  assert.equal(ui.payload({ revision: 0, cache: '1024', temp: '2', free: '2' }).cache_limit_bytes, 1024 ** 4);
});

test('a refreshed budget preserves dirty input and its original revision until explicit reset', () => {
  const ui = fixture();
  assert.equal(typeof ui.reconcile, 'function');
  const first = ui.reconcile(null, { revision: 4, cache_limit_bytes: 8 * 1024 ** 3, temp_limit_bytes: 2 * 1024 ** 3, minimum_free_bytes: 2 * 1024 ** 3 });
  first.cache = '12'; first.dirty = true;
  const latest = { revision: 5, cache_limit_bytes: 9 * 1024 ** 3, temp_limit_bytes: 2 * 1024 ** 3, minimum_free_bytes: 2 * 1024 ** 3 };
  assert.equal(ui.reconcile(first, latest), first);
  assert.equal(first.cache, '12'); assert.equal(first.revision, 4);
  assert.equal(ui.reconcile(null, latest).revision, 5);
});

test('budget request ownership rejects delayed older loads', () => {
  const ui = fixture();
  assert.equal(typeof ui.requestGate, 'function');
  const gate = ui.requestGate(), older = gate.begin(), newer = gate.begin();
  assert.equal(gate.owns(older), false); assert.equal(gate.owns(newer), true);
});

test('late budget responses cannot replace a newer result or release its busy state', async () => {
  const ui = fixture(), gate = ui.requestGate(), applied = []; let releaseOld, releaseNew, settled = 0;
  const old = gate.run(() => new Promise(resolve => { releaseOld = resolve; }), value => applied.push(value), error => applied.push(error.message), () => { settled++; });
  const current = gate.run(() => new Promise(resolve => { releaseNew = resolve; }), value => applied.push(value), error => applied.push(error.message), () => { settled++; });
  releaseOld('old'); await old;
  assert.equal(applied.length, 0); assert.equal(settled, 0);
  releaseNew('new'); await current;
  assert.deepEqual(applied, ['new']); assert.equal(settled, 1);
});

test('a stale budget failure cannot display an error over a newer successful request', async () => {
  const ui = fixture(), gate = ui.requestGate(), applied = []; let rejectOld, settled = 0;
  const old = gate.run(() => new Promise((_, reject) => { rejectOld = reject; }), value => applied.push(value), error => applied.push(error.message), () => { settled++; });
  await gate.run(() => Promise.resolve('latest'), value => applied.push(value), error => applied.push(error.message), () => { settled++; });
  rejectOld(Error('stale failure')); await old;
  assert.deepEqual(applied, ['latest']); assert.equal(settled, 1);
});

test('agent drawer keeps actual running dispatch separate from selected upcoming role settings', () => {
  const ui = fixture('page-agent-settings.js', 'AgentSettings');
  assert.equal(typeof ui.selectionSummary, 'function');
  const running = { role: 'implementer', domain: 'frontend', provider_name: 'DeepSeek Harness', model: 'actual-model', effort: 'high', status: 'running' };
  const task = { runs: [running] }, plan = { frontend: 'codex', frontend_model: 'future-model', frontend_effort: 'max' };
  const summary = ui.selectionSummary(task, 'frontend', plan, [{ id: 'codex', name: 'Codex', model: 'default-model' }]);
  assert.equal(summary.actual.model, 'actual-model'); assert.equal(summary.actual.effort, 'high');
  assert.equal(summary.upcoming.model, 'future-model'); assert.equal(summary.upcoming.effort, 'max');
  assert.equal(running.model, 'actual-model');
  assert.equal(ui.selectionSummary({ runs: [{ ...running, status: 'completed' }] }, 'frontend', plan, []).actual, null);
  assert.equal(ui.selectionSummary(task, 'backend', plan, []).actual, null);
});
