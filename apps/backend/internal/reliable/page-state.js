'use strict';
const ModelState = (() => {
  function stageResumable(task) {
    if (!task?.model_plan?.mode || task.round < 1 || !task.audit || !task.worktree || (task.requirement_revision || 0) !== (task.audited_revision || 0)) return false;
    if (task.recovery_pending || task.recovery_evidence?.length || task.seed_pending) return false;
    const stage = task.stage_recovery;
    if (stage && stage.round === task.round && stage.status !== 'completed') return true;
    const last = task.runs?.at(-1);
    return !!last && last.role === 'implementer' && last.round === task.round && ['failed','interrupted'].includes(last.status) && !!last.domain;
  }
  function settingsTask(task) {
    const future = !!task.local_checkpoint || ['inspected','accepted','cancelled','exhausted','awaiting_acceptance'].includes(task.state) || (['blocked','interrupted'].includes(task.state) && task.round >= 3 && !stageResumable(task));
    return { ...task, settings_scope: future ? 'future' : 'active', settings_has_future: !!task.future_plan, settings_editable: !task.successor_task_id && !task.archived_at && !['cancelling','applying'].includes(task.state), model_plan: future ? task.future_plan?.plan || task.model_plan : task.model_plan, plan_revision: future ? task.future_plan?.revision || 0 : task.plan_revision };
  }
  function reconcileDraft(draft, task, defaults) {
    const revision = task.plan_revision || 0;
    const scope = task.settings_scope || 'active';
    if (draft?.dirty && draft.scope && draft.scope !== scope) return { ...draft, revision, scope, needsReload: !!task.settings_has_future, conflict: !!task.settings_has_future };
    if (draft?.dirty) return { ...draft, scope, conflict: !!draft.needsReload || draft.revision !== revision };
    const saved = task.model_plan || {};
    const plan = saved.mode ? { ...defaults, ...saved } : Object.fromEntries(['auditor', 'implementer', 'reviewer'].map(role => [role, saved[role] || defaults[role]]));
    if (!Object.hasOwn(saved, 'assistant_mode')) delete plan.assistant_mode;
    if (!saved.mode) for (const [key, value] of Object.entries(saved)) if (value) plan[key] = value;
    return { plan, revision, scope, dirty: false, conflict: false };
  }
  function taskTitle(task = {}) {
    const first = String(task.goal || '').trim().split(/\r?\n/)[0].replace(/\s+/g, ' ');
    const title = first || String(task.project || '').replace(/[\\/]+$/, '').split(/[\\/]/).at(-1) || '项目协作';
    const characters = Array.from(title);
    return characters.length > 28 ? characters.slice(0, 28).join('') + '…' : title;
  }
  function goalText(task = {}) { return String(task.goal || '').replace(/NEXT BATCH REQUIREMENT:\s*/g, '下一批要求：\n'); }
  return { reconcileDraft, settingsTask, taskTitle, goalText, stageResumable };
})();
if (typeof module !== 'undefined') module.exports = ModelState;
