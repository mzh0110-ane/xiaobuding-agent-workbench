const { test } = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { join } = require('node:path');
const vm = require('node:vm');

class Node {
  constructor(tag, className = '', text = '') {
    Object.assign(this, { tag, className, text, children: [], dataset: {}, open: false });
    this.classList = { add: value => { this.className += ' ' + value; } };
  }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.children = children; }
  querySelectorAll() { return descendants(this).filter(node => node.tag === 'details' && node.dataset.disclosure); }
}
function descendants(node) { return node.children.flatMap(child => [child, ...descendants(child)]); }
function text(node) { return [node.text, ...descendants(node).map(child => child.text)].join('\n'); }
function fixture(api = async () => []) {
  const root = new Node('div');
  const context = vm.createContext({
    $: () => root, el: (tag, cls, value) => new Node(tag, cls, value), api, selected: 'task',
    evidenceURL: (id, path) => id + '/' + path, friendly: value => value,
    roleTitle: role => role, PlanUI: { effortNames: {} },
  });
  vm.runInContext(readFileSync(join(__dirname, 'page-artifacts.js'), 'utf8') + '\n' + readFileSync(join(__dirname, 'page-proof.js'), 'utf8') + '\nglobalThis.proof = TaskProof;', context);
  return { root, context, render: task => { context.selected = task.id; context.proof.render(task); } };
}
function passing(overrides = {}) {
  return {
    id: 'task', state: 'awaiting_acceptance', round: 2, snapshot: '0123456789abcdef0123456789abcdef',
    checks: [{ command: 'node --test', exit_code: 0, log: 'check-1.log' }],
    review: { verdict: 'pass', summary: 'The checked behavior passed.', findings: [] }, ...overrides,
  };
}
const flush = () => new Promise(resolve => setImmediate(resolve));

test('passing evidence describes one batch and keeps review limitations prominent', () => {
  const f = fixture();
  f.render(passing({ review: { verdict: 'pass', summary: 'Known limitation: image recognition remains unavailable.', findings: [{ severity: 'P2', file: 'app.js', reason: 'The preview requires a manual refresh.' }] } }));
  assert.match(text(f.root), /本批核验已通过，等待你验收/);
  assert.match(text(f.root), /整个项目是否达到你的预期/);
  const review = descendants(f.root).find(node => node.className.includes('result-limitations'));
  assert.match(text(review), /image recognition remains unavailable/);
  assert.match(text(review), /manual refresh/);
  assert.match(text(f.root), /本批第 2 轮/);
  assert.match(text(f.root), /0123456789abcdef/);
});

test('failed checks, empty verification and newer requirements cannot claim a passing batch', () => {
  for (const change of [
    { checks: [{ command: 'test', exit_code: 1, log: 'check-1.log' }] },
    { checks: [{ command: 'test', exit_code: 0, verification_problem: 'Ran zero tests', log: 'check-1.log' }] },
    { checks: [] },
    { requirement_revision: 2, audited_revision: 1 },
  ]) {
    const f = fixture(); f.render(passing(change));
    const status = descendants(f.root).find(node => node.className === 'result-status');
    assert.doesNotMatch(status.text, /已通过/);
  }
});

test('each check opens its own recorded round log and unsafe log paths stay unavailable', () => {
  const f = fixture();
  f.render(passing({ checks: [
    { command: 'test', exit_code: 0, log: 'check-1.log' },
    { command: 'lint', exit_code: 1, log: 'check-2.log' },
    { command: 'verify', exit_code: 0, log: '../other.log' },
  ] }));
  const links = descendants(f.root).filter(node => node.tag === 'a');
  assert.deepEqual(links.map(link => link.href), ['task/round-2/check-1.log', 'task/round-2/check-2.log']);
  assert.match(text(f.root), /本项没有可用的日志路径/);
});

test('available patches remain accessible without a review and missing files are not invented', async () => {
  const f = fixture(async () => ['round-2/changes.patch', 'round-1-review.json']);
  f.render(passing({ review: undefined })); await flush();
  const quick = descendants(f.root).find(node => node.className === 'proof-links');
  assert.deepEqual(descendants(quick).filter(node => node.tag === 'a').map(node => node.href), ['task/round-2/changes.patch']);
  const missing = fixture(); missing.render(passing()); await flush();
  const empty = descendants(missing.root).find(node => node.className === 'proof-links');
  assert.equal(descendants(empty).filter(node => node.tag === 'a').length, 0);
  assert.match(text(empty), /本轮尚未生成/);
});

test('a delayed evidence response cannot replace another task result', async () => {
  const responses = [];
  const f = fixture(() => new Promise(resolve => responses.push(resolve)));
  f.render(passing()); f.render(passing({ id: 'new-task' }));
  responses[0](['round-2/changes.patch']); await flush();
  assert.equal(descendants(f.root).filter(node => node.href === 'task/round-2/changes.patch').length, 0);
  responses[1]([]); await flush();
  assert.match(text(f.root), /本轮尚未生成/);
});

test('read-only results identify their limits and preserve a collapsed audit on refresh', () => {
  const f = fixture(), task = { id: 'task', read_only: true, state: 'inspected', audit: { summary: 'Only source investigation.', facts: ['src/app.js exists'], inferences: ['A model inference'], unknowns: ['Unverified behavior'], read_files: ['src/app.js'] } };
  f.render(task);
  assert.match(text(f.root), /没有执行代码修改、实际测试或独立验收/);
  const audit = f.root.querySelectorAll().find(node => node.dataset.disclosure === 'audit');
  assert.equal(audit.open, true); audit.open = false;
  f.render(task);
  assert.equal(f.root.querySelectorAll().find(node => node.dataset.disclosure === 'audit').open, false);
});
