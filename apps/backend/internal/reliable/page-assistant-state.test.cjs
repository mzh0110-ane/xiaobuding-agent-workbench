const { test } = require('node:test');
const assert = require('node:assert/strict');
const AssistantState = require('./page-assistant-state.js');

const providers = [
  { id: 'codex', available: true, model: 'gpt-6.1-sol', models: [{ id: 'gpt-6.1-sol', efforts: ['low', 'high'], default_effort: 'low' }] },
  { id: 'deepseek', available: true, model: 'deepseek-flash', models: [{ id: 'deepseek-flash', efforts: ['high', 'max'], default_effort: 'high' }] },
];
const original = { mode: 'fullstack', auditor: 'codex', frontend: 'deepseek', backend: 'codex', reviewer: 'codex', frontend_paths: 'web', backend_paths: 'server', reviewer_model: 'gpt-6.1-sol', reviewer_effort: 'high' };

test('legacy assignments remain custom until the user explicitly chooses a preset', () => {
  assert.equal(AssistantState.mode({ ...original, frontend: 'codex' }), 'custom');
  assert.equal(AssistantState.mode({ assistant_mode: 'unknown' }), 'custom');
  assert.equal(AssistantState.mode({ assistant_mode: 'deepseek' }), 'deepseek');
});

test('DeepSeek preset shares one execution selection while preserving scope and independent Codex review', () => {
  const next = AssistantState.apply(original, 'deepseek', providers, 'deepseek-flash', 'max');
  assert.deepEqual(original.frontend_paths, 'web');
  for (const key of ['auditor', 'frontend', 'backend']) {
    assert.equal(next[key], 'deepseek'); assert.equal(next[key + '_model'], 'deepseek-flash'); assert.equal(next[key + '_effort'], 'max');
  }
  assert.equal(next.reviewer, 'codex'); assert.equal(next.reviewer_model, original.reviewer_model); assert.equal(next.reviewer_effort, original.reviewer_effort);
  assert.equal(next.mode, original.mode); assert.equal(next.frontend_paths, 'web'); assert.equal(next.backend_paths, 'server');
  assert.equal(original.frontend_model, undefined);
});

test('switching Codex back to custom keeps user selections rather than resetting the project', () => {
  const codex = AssistantState.apply(original, 'codex', providers, 'gpt-6.1-sol', 'high');
  const custom = AssistantState.apply(codex, 'custom', providers);
  assert.equal(custom.assistant_mode, 'custom'); assert.equal(custom.frontend_model, 'gpt-6.1-sol');
  assert.equal(custom.backend_effort, 'high'); assert.equal(custom.mode, 'fullstack');
});

test('single-domain and legacy presets change only active execution roles', () => {
  const frontend = AssistantState.apply({ ...original, mode: 'frontend', backend_model: 'keep-inactive' }, 'codex', providers, 'gpt-6.1-sol', 'low');
  assert.equal(frontend.backend_model, 'keep-inactive'); assert.equal(frontend.auditor, 'codex');
  const legacy = AssistantState.apply({ auditor: 'codex', implementer: 'deepseek', reviewer: 'codex' }, 'deepseek', providers);
  assert.equal(legacy.implementer, 'deepseek'); assert.equal(legacy.mode, undefined);
});

test('unavailable tools, missing models and unsupported effort never fall back silently', () => {
  assert.throws(() => AssistantState.apply(original, 'deepseek', [{ ...providers[1], available: false }, providers[0]]), /不可用/);
  assert.throws(() => AssistantState.apply(original, 'codex', providers, 'missing'), /目录/);
  assert.throws(() => AssistantState.apply(original, 'codex', providers, 'gpt-6.1-sol', 'max'), /思考强度/);
  assert.throws(() => AssistantState.apply(original, 'codex', [providers[1]]), /不可用/);
});

test('a missing chosen model stays explicit in the selector instead of choosing the first model', () => {
  const result = AssistantState.selection({ ...original, assistant_mode: 'codex', auditor_model: 'temporarily-missing', auditor_effort: 'high' }, providers);
  assert.equal(result.model, 'temporarily-missing'); assert.equal(result.modelKnown, false); assert.equal(result.effort, 'high');
});

test('manual role settings mark the preset custom and repair legacy review routing only when requested', () => {
  const next = AssistantState.custom({ ...original, assistant_mode: 'deepseek' });
  assert.equal(next.assistant_mode, 'custom'); assert.equal(next.reviewer, 'codex');
  const migrated = AssistantState.custom({ auditor: 'deepseek', implementer: 'codex', reviewer: 'deepseek', reviewer_model: 'deepseek-flash', reviewer_effort: 'max' });
  assert.equal(migrated.reviewer, 'codex'); assert.equal(migrated.reviewer_model, ''); assert.equal(migrated.reviewer_effort, '');
});

test('read-only inspection can choose an available native tool without requiring an unused review connection', () => {
  const next = AssistantState.apply(original, 'deepseek', [providers[1]], 'deepseek-flash', 'high', true);
  assert.equal(next.auditor, 'deepseek'); assert.equal(next.reviewer, 'codex');
});

test('changing new-task work domains reapplies a named preset to every newly active role', () => {
  for (const starting of ['frontend', 'backend']) {
    const single = AssistantState.apply({ ...original, mode: starting }, 'codex', providers, 'gpt-6.1-sol', 'high');
    const full = AssistantState.changeWorkMode(single, 'fullstack', providers);
    for (const role of ['auditor', 'frontend', 'backend']) { assert.equal(full[role], 'codex'); assert.equal(full[role + '_model'], 'gpt-6.1-sol'); assert.equal(full[role + '_effort'], 'high'); }
    assert.equal(full.frontend_paths, 'web'); assert.equal(full.backend_paths, 'server');
  }
});

test('restoring custom assignments restores role selections without reviving an old scope or work mode', () => {
  const saved = { ...original, auditor_model: 'gpt-6.1-sol', frontend_model: 'deepseek-flash', frontend_effort: 'max', backend_model: 'gpt-6.1-sol', backend_effort: 'high' };
  const snapshot = AssistantState.assignmentSnapshot(saved);
  const preset = AssistantState.apply({ ...original, mode: 'backend', frontend_paths: 'current/web', backend_paths: 'current/server' }, 'codex', providers, 'gpt-6.1-sol', 'low');
  const restored = AssistantState.restoreCustom(preset, snapshot);
  assert.equal(restored.assistant_mode, 'custom'); assert.equal(restored.frontend, 'deepseek'); assert.equal(restored.frontend_model, 'deepseek-flash'); assert.equal(restored.frontend_effort, 'max');
  assert.equal(restored.backend_model, 'gpt-6.1-sol'); assert.equal(restored.backend_effort, 'high'); assert.equal(restored.mode, 'backend');
  assert.equal(restored.frontend_paths, 'current/web'); assert.equal(restored.backend_paths, 'current/server'); assert.equal(snapshot.mode, undefined); assert.equal(snapshot.frontend_paths, undefined);
});

test('custom snapshots reject unrelated persisted keys and preserve deliberate later custom changes', () => {
  const later = { ...original, backend: 'deepseek', backend_model: 'deepseek-flash', backend_effort: 'max' };
  const snapshot = AssistantState.assignmentSnapshot({ ...later, mode: 'frontend', allowed_paths: '.', token: 'private', assistant_mode: 'codex' });
  assert.equal(snapshot.token, undefined); assert.equal(snapshot.allowed_paths, undefined); assert.equal(snapshot.assistant_mode, undefined);
  const restored = AssistantState.restoreCustom({ ...original }, snapshot);
  assert.equal(restored.backend, 'deepseek'); assert.equal(restored.backend_effort, 'max');
});

test('restoring an old custom plan with omitted native defaults clears preset-specific model and effort', () => {
  const snapshot = AssistantState.assignmentSnapshot({ mode: 'fullstack', auditor: 'codex', frontend: 'deepseek', backend: 'codex', reviewer: 'codex' });
  const preset = AssistantState.apply(original, 'deepseek', providers, 'deepseek-flash', 'max');
  const restored = AssistantState.restoreCustom(preset, snapshot);
  assert.equal(restored.backend, 'codex'); assert.equal(restored.backend_model, ''); assert.equal(restored.backend_effort, '');
  assert.equal(restored.auditor_model, ''); assert.equal(restored.reviewer_effort, '');
});

test('unsaved assignment guards block only work dispatch and enabling automatic continuation', () => {
  for (const action of ['plan', 'start', 'replan', 'resume', 'recover-cancelled', 'accept', 'question', 'requirement', 'next-batch']) assert.equal(AssistantState.blocksDispatch(true, action), true, action);
  assert.equal(AssistantState.blocksDispatch(true, 'project-save', { auto_continue: true }), true);
  for (const action of ['cancel', 'stop', 'archive', 'unarchive', 'open', 'open-project', 'open-evidence', 'read', 'models', 'project-save']) assert.equal(AssistantState.blocksDispatch(true, action), false, action);
  assert.equal(AssistantState.blocksDispatch(true, 'project-save', { auto_continue: false }), false);
  assert.equal(AssistantState.blocksDispatch(false, 'accept'), false);
});
