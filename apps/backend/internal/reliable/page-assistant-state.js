'use strict';
const AssistantState = (() => {
  const modes = { codex: 'Codex', deepseek: 'DeepSeek Harness', zcode: 'ZCode', custom: '自定义协作' };
  function mode(plan = {}) { return Object.hasOwn(modes, plan.assistant_mode) ? plan.assistant_mode : 'custom'; }
  function executionKeys(plan = {}) { return plan.mode ? ['auditor', ...(plan.mode !== 'backend' ? ['frontend'] : []), ...(plan.mode !== 'frontend' ? ['backend'] : [])] : ['auditor', 'implementer']; }
  function models(provider) { return provider?.models?.length ? provider.models : provider ? [{ id: provider.model, name: provider.model, efforts: [] }] : []; }
  function modelRole(plan, selected) { return selected === 'zcode' && plan.auditor !== 'zcode' ? !plan.mode ? 'implementer' : plan.mode === 'backend' ? 'backend' : 'frontend' : 'auditor'; }
  function selection(plan = {}, providers = []) {
    const selected = mode(plan), provider = providers.find(item => item.id === (selected === 'custom' ? plan.auditor : selected));
    const role = modelRole(plan, selected), model = plan[role + '_model'] || provider?.model || '', option = models(provider).find(item => item.id === model);
    return { mode: selected, provider, models: models(provider), model, modelKnown: !!option, effort: plan[role + '_effort'] || '', efforts: option?.efforts || [], defaultEffort: option?.default_effort || '' };
  }
  function custom(plan = {}) {
    const next = { ...plan, assistant_mode: 'custom', reviewer: 'codex' };
    if (plan.reviewer !== 'codex') { next.reviewer_model = ''; next.reviewer_effort = ''; }
    return next;
  }
  function apply(plan, selected, providers = [], chosenModel, chosenEffort, readOnly = false) {
    if (!Object.hasOwn(modes, selected)) throw Error('请选择已有的 Agent 模式。');
    const next = custom(plan);
    if (selected === 'custom') return next;
    const provider = providers.find(item => item.id === selected && item.available);
    if (!provider) throw Error('所选原生工具目前不可用，请先检查原生登录和连接。');
    if (!readOnly && !providers.some(item => item.id === 'codex' && item.available)) throw Error('独立审查需要可用的 Codex 原生连接。');
    const model = chosenModel || provider.model || '', option = models(provider).find(item => item.id === model), effort = chosenEffort || '';
    if (!option) throw Error('所选模型不在这个工具的当前目录中，请刷新后再选择。');
    if (effort && !(option.efforts || []).includes(effort)) throw Error('当前模型不支持这个思考强度。');
    next.assistant_mode = selected;
    if(readOnly){next.auditor=selected;next.auditor_model=model;next.auditor_effort=effort;return next;}
    if (selected === 'zcode') {
      if (next.auditor !== 'codex') { next.auditor_model = ''; next.auditor_effort = ''; }
      next.auditor = 'codex';
    }
    for (const key of executionKeys(plan).filter(key => selected !== 'zcode' || key !== 'auditor')) { next[key] = selected; next[key + '_model'] = model; next[key + '_effort'] = effort; }
    return next;
  }
  function changeWorkMode(plan, workMode, providers, readOnly = false) {
    if (!['frontend', 'backend', 'fullstack'].includes(workMode)) throw Error('请选择前端、后端或前后端工作范围。');
    const next = { ...plan, mode: workMode }, selected = mode(plan);
    const choice = selection(plan, providers);
    return selected === 'custom' ? next : apply(next, selected, providers, choice.model, choice.effort, readOnly);
  }
  function assignmentSnapshot(plan = {}) {
    const saved = {};
    for (const role of ['auditor', 'implementer', 'reviewer', 'frontend', 'backend']) {
      if (typeof plan[role] === 'string') saved[role] = plan[role];
      for (const suffix of ['_model', '_effort']) {
        const key = role + suffix;
        if (typeof plan[key] === 'string' || typeof plan[role] === 'string') saved[key] = typeof plan[key] === 'string' ? plan[key] : '';
      }
    }
    return saved;
  }
  function restoreCustom(plan, saved) { return custom({ ...plan, ...assignmentSnapshot(saved || plan) }); }
  function blocksDispatch(unsaved, action, body = {}) {
    return !!unsaved && (['plan', 'start', 'replan', 'resume', 'recover-cancelled', 'accept', 'question', 'requirement', 'next-batch'].includes(action) || action === 'project-save' && body.auto_continue === true);
  }
  return { modes, mode, executionKeys, models, selection, apply, custom, changeWorkMode, assignmentSnapshot, restoreCustom, blocksDispatch };
})();
if (typeof module !== 'undefined') module.exports = AssistantState;
