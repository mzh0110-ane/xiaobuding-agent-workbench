const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = __dirname + '/page-workbench-state.js';
function state() {
  assert.ok(fs.existsSync(source), 'the workbench state controller is not implemented');
  const context = vm.createContext({ Map, Promise, Error, URL });
  vm.runInContext(fs.readFileSync(source, 'utf8') + ';globalThis.subject=WorkbenchState;', context);
  return context.subject;
}
test('project grouping uses backend identity and opens newest uncontinued batch', () => {
  const groups = state().groups([
    { id: 'old', project_id: 'one', project: 'E:/same', created: '1', successor_task_id: 'new' },
    { id: 'other', project_id: 'two', project: 'F:/same', created: '2' },
    { id: 'new', project_id: 'one', project: 'E:/same', created: '3' },
  ]);
  assert.equal(groups.length, 2);
  assert.equal(groups[0].current.id, 'new');
  assert.equal(groups[0].tasks.length, 2);
  assert.notEqual(groups[0].id, groups[1].id);
});
test('conditional reader coalesces pending fetches and reuses a 304 snapshot', async () => {
  let calls = 0, release;
  const seen = [];
  const gate = new Promise(resolve => { release = resolve; });
  const reader = state().reader(async (path, options) => {
    calls++; seen.push(options.headers);
    if (calls === 1) { await gate; return { ok: true, status: 200, headers: { get: () => 'rev1' }, json: async () => ({ id: 'one' }) }; }
    return { ok: false, status: 304 };
  });
  const first = reader.read('tasks/one'), duplicate = reader.read('tasks/one');
  release();
  const [a, b] = await Promise.all([first, duplicate]);
  assert.equal(calls, 1); assert.equal(a, b);
  assert.equal(await reader.read('tasks/one'), a);
  assert.equal(seen[1]['If-None-Match'], 'rev1');
});
test('conditional reader recovers after failure without caching failed state', async () => {
  let calls = 0;
  const reader = state().reader(async () => {
    if (++calls === 1) throw Error('offline');
    return { ok: true, status: 200, headers: { get: () => 'good' }, json: async () => ({ recovered: true }) };
  });
  await assert.rejects(reader.read('tasks/one'), /offline/);
  assert.equal((await reader.read('tasks/one')).recovered, true);
  assert.equal(calls, 2);
});
test('task reminders only announce a transition into an actionable state', () => {
  const subject = state();
  assert.equal(subject.shouldNotify(undefined, 'blocked'), false);
  assert.equal(subject.shouldNotify('blocked', 'blocked'), false);
  assert.equal(subject.shouldNotify('implementing', 'checking'), false);
  assert.equal(subject.shouldNotify('reviewing', 'awaiting_acceptance'), true);
  assert.equal(subject.shouldNotify('auditing', 'blocked'), true);
});
test('preview links are restricted to an explicit loopback HTTP port', () => {
  const subject = state();
  assert.equal(subject.previewURL('http://127.0.0.1:4321/'), 'http://127.0.0.1:4321/');
  for (const value of ['javascript:alert(1)', 'https://example.com/', 'http://127.0.0.1/', 'http://user@127.0.0.1:4321/', 'http://127.0.0.1.evil:4321/']) assert.equal(subject.previewURL(value), '');
});
test('a cached background task cannot hide a fresh completion or archive summary', () => {
  const subject = state();
  assert.equal(typeof subject.navigationItem, 'function', 'summary and detail merging must preserve background state');
  const summary = { id: 'a', updated: '2', state: 'awaiting_acceptance', archived_at: 'now', summary_only: true };
  const cached = { id: 'a', updated: '1', state: 'implementing', goal: 'complete requirement' };
  assert.equal(subject.navigationItem(summary, cached, 'b').state, 'awaiting_acceptance');
  assert.equal(subject.navigationItem(summary, cached, 'b').archived_at, 'now');
  assert.equal(subject.navigationItem(summary, cached, 'a').goal, 'complete requirement');
});
test('loaded history keeps earlier observed windows when new messages advance the latest page', () => {
  const subject = state();
  assert.equal(typeof subject.mergeMessages, 'function');
  const messages = (start, end) => Array.from({ length: end - start }, (_, n) => ({ id: 'm' + (start + n), value: start + n }));
  const got = subject.mergeMessages(messages(0, 35), messages(35, 60), messages(45, 70));
  assert.equal(got.length, 70); assert.equal(got[35].id, 'm35'); assert.equal(got[69].id, 'm69');
});
test('backfilled history is ordered by its original position despite overlapping pages', () => {
  const subject = state();
  const messages = (start, end) => Array.from({ length: end - start }, (_, n) => ({ id: 'm' + (start + n), _position: start + n }));
  const got = subject.mergeMessages(messages(50, 75), messages(0, 60), messages(75, 100));
  assert.equal(got.length, 100); assert.equal(got[0].id, 'm0'); assert.equal(got[74].id, 'm74');
});
test('history cursor skips already covered positions but keeps a real gap available', () => {
  const subject = state(); assert.equal(typeof subject.earlierCursor, 'function');
  const positions = (a, b) => Array.from({ length: b - a }, (_, n) => ({ _position: a + n }));
  assert.equal(subject.earlierCursor([...positions(0, 60), ...positions(75, 100)], 75), 75);
  assert.equal(subject.earlierCursor(positions(0, 100), 75), 0);
  assert.equal(subject.earlierCursor(positions(10, 100), 75), 10);
});
test('preview request ownership rejects an earlier status after a newer start or stop', () => {
  const subject = state(); assert.equal(typeof subject.requestGate, 'function');
  const gate = subject.requestGate(), refresh = gate.begin(), start = gate.begin();
  assert.equal(gate.owns(refresh), false); assert.equal(gate.owns(start), true);
});
test('summary navigation follows requested pages and exposes remaining history', async () => {
  const subject = state(); assert.equal(typeof subject.summaryPages, 'function');
  const paths = [];
  const result = await subject.summaryPages(async value => { paths.push(value); return value.includes('offset=2') ? { tasks: [{ id: 'c' }], total: 3, next_offset: -1 } : { tasks: [{ id: 'a' }, { id: 'b' }], total: 3, next_offset: 2 }; }, 2, '完整需求');
  assert.equal(result.tasks.length, 3); assert.equal(result.next_offset, -1); assert.match(paths[1], /offset=2/); assert.match(paths[0], /q=/);
});
