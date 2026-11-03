const { test } = require('node:test');
const assert = require('node:assert/strict');
const ModelState = require('./page-state.js');
const { reconcileDraft } = ModelState;
const defaults = { auditor: 'deepseek', implementer: 'codex', reviewer: 'deepseek' };

test('legacy preset metadata does not inherit a new default and saved future preset survives reconciliation', () => {
  const team = { ...defaults, mode: 'fullstack', frontend: 'codex', backend: 'codex', assistant_mode: 'codex' };
  const historical = reconcileDraft(null, { model_plan: { ...team, assistant_mode: undefined } }, team);
  assert.equal(historical.plan.assistant_mode, undefined);
  const legacyDomain = { ...team }; delete legacyDomain.assistant_mode;
  assert.equal(reconcileDraft(null, { model_plan: legacyDomain }, team).plan.assistant_mode, undefined);
  const futurePlan = { ...team, assistant_mode: 'deepseek', frontend: 'deepseek', backend: 'deepseek', auditor: 'deepseek' };
  const task = ModelState.settingsTask({ state: 'accepted', model_plan: legacyDomain, future_plan: { revision: 2, plan: futurePlan } });
  assert.equal(reconcileDraft(null, task, team).plan.assistant_mode, 'deepseek');
  assert.equal(task.settings_scope, 'future');
});

test('third-round operational writer recovery keeps current model settings editable', () => {
  const task = { state: 'blocked', round: 3, worktree: 'E:/copy', audit: {}, model_plan: { ...defaults, mode: 'fullstack' }, plan_revision: 4, stage_recovery: { round: 3, domain: 'backend', status: 'retryable' } };
  assert.equal(ModelState.settingsTask(task).settings_scope, 'active');
  assert.equal(ModelState.stageResumable(task), true);
  assert.equal(ModelState.stageResumable({ ...task, requirement_revision: 2, audited_revision: 1 }), false);
  assert.equal(ModelState.settingsTask({ ...task, stage_recovery: { round: 3, status: 'completed' } }).settings_scope, 'future');
});

test('polling preserves an unsaved stage selection and its original revision', () => {
  const draft = { plan: { ...defaults, reviewer: 'codex' }, revision: 3, dirty: true };
  const task = { plan_revision: 3, model_plan: defaults, state: 'reviewing' };
  const next = reconcileDraft(draft, task, defaults);
  assert.deepEqual(next?.plan, draft.plan);
  assert.equal(next.revision, 3);
  assert.equal(next.dirty, true);
});

test('a newer remote arrangement surfaces a conflict without discarding local input', () => {
  const draft = { plan: { ...defaults, implementer: 'deepseek' }, revision: 2, dirty: true };
  const next = reconcileDraft(draft, { plan_revision: 4, model_plan: defaults }, defaults);
  assert.deepEqual(next?.plan, draft.plan);
  assert.equal(next.conflict, true);
  assert.equal(next.revision, 2);
});

test('a clean editor follows persisted changes and legacy tasks receive defaults', () => {
  const next = reconcileDraft(null, { plan_revision: 5, model_plan: { ...defaults, auditor: 'codex' } }, defaults);
  assert.equal(next?.plan.auditor, 'codex');
  assert.equal(next.revision, 5);
  assert.deepEqual(reconcileDraft(null, { model_plan: { auditor: '', reviewer: '' } }, defaults).plan, defaults);
});

test('saved domain, native model and effort survive reload without upgrading legacy tasks', () => {
  const team = { mode: 'fullstack', frontend: 'deepseek', backend: 'codex', auditor: 'codex', reviewer: 'codex' };
  const saved = { ...team, mode: 'frontend', frontend_model: 'deepseek-v4-pro', frontend_effort: 'max' };
  const restored = reconcileDraft(null, { plan_revision: 2, model_plan: saved }, team);
  assert.equal(restored.plan.mode, 'frontend');
  assert.equal(restored.plan.frontend_model, 'deepseek-v4-pro');
  assert.equal(restored.plan.frontend_effort, 'max');
  const legacy = reconcileDraft(null, { model_plan: defaults }, team);
  assert.equal(legacy.plan.mode, undefined);
});

test('resting task settings project a separate editable future revision', () => {
  assert.equal(typeof ModelState.settingsTask,'function');
  const task={state:'inspected',read_only:true,plan_revision:5,model_plan:defaults};
  const next=ModelState.settingsTask(task);
  assert.equal(next.settings_scope,'future');assert.equal(next.plan_revision,0);assert.equal(next.settings_editable,true);
  assert.equal(task.plan_revision,5);
  const saved=ModelState.settingsTask({...task,future_plan:{revision:2,plan:{...defaults,auditor:'codex'}}});
  assert.equal(saved.model_plan.auditor,'codex');assert.equal(saved.plan_revision,2);
  assert.equal(ModelState.settingsTask({...task,archived_at:'saved'}).settings_editable,false);
});

test('an unsaved active plan becomes a future draft when a task completes',()=>{
  assert.equal(typeof ModelState.settingsTask,'function');
  const draft={plan:{...defaults,auditor:'codex'},revision:5,scope:'active',dirty:true};
  const next=reconcileDraft(draft,ModelState.settingsTask({state:'inspected',plan_revision:5,model_plan:defaults}),defaults);
  assert.equal(next.plan.auditor,'codex');assert.equal(next.revision,0);assert.equal(next.scope,'future');assert.equal(next.conflict,false);
  const remote=reconcileDraft(draft,ModelState.settingsTask({state:'inspected',plan_revision:5,model_plan:defaults,future_plan:{revision:1,plan:defaults}}),defaults);
  assert.equal(remote.conflict,true);
  assert.equal(reconcileDraft(remote,ModelState.settingsTask({state:'inspected',plan_revision:5,model_plan:defaults,future_plan:{revision:1,plan:defaults}}),defaults).conflict,true);
});

test('a pending accepted checkpoint keeps model edits in the future scope through interrupted receipt completion', () => {
  const source = { state: 'blocked', round: 1, plan_revision: 5, model_plan: defaults, local_checkpoint: { complete: false } };
  const pending = ModelState.settingsTask(source);
  assert.equal(pending.settings_scope, 'future'); assert.equal(pending.settings_editable, true); assert.equal(pending.plan_revision, 0);
  const future = { ...defaults, auditor: 'codex' };
  const interrupted = ModelState.settingsTask({ ...source, state: 'interrupted', local_checkpoint: { complete: true, commit: 'saved' }, future_plan: { revision: 2, plan: future } });
  assert.equal(interrupted.settings_scope, 'future'); assert.equal(interrupted.settings_editable, true);
  assert.equal(interrupted.plan_revision, 2); assert.deepEqual(interrupted.model_plan, future);
  assert.equal(ModelState.settingsTask({ ...source, state: 'applying' }).settings_editable, false);
  assert.equal(source.plan_revision, 5); assert.deepEqual(source.model_plan, defaults);
});

// @covers AC-RELIABLE-RENEWAL-001.4/.5/.7
test('a superseded batch keeps historical assignments read-only while its child remains editable', () => {
  const parent = { state: 'exhausted', round: 3, successor_task_id: 'child', model_plan: defaults };
  const projected = ModelState.settingsTask(parent);
  assert.equal(projected.settings_editable, false);
  assert.equal(projected.settings_scope, 'future');
  assert.deepEqual(projected.model_plan, defaults);
  assert.equal(parent.successor_task_id, 'child');
  const child = ModelState.settingsTask({ state: 'queued', round: 0, parent_task_id: 'parent', model_plan: defaults, plan_revision: 1 });
  assert.equal(child.settings_editable, true);
  assert.equal(child.settings_scope, 'active');
  assert.deepEqual(child.model_plan, defaults);
});
