'use strict';
const AssistantUI = (() => {
  let current = null, newControls = null, taskControls = null, loading = false, saveError = null;
  const cards = new Map();
  const customAssignments = new Map();
  const titles = { codex: 'Codex', deepseek: 'DeepSeek Harness', zcode: 'ZCode', custom: '自定义协作' };
  const descriptions = { codex: 'ChatGPT 登录 · 由 Codex 规划与实施', deepseek: 'DeepSeek 账号 · 由 Harness 规划与实施', zcode: 'ZCode 实施 · Codex 规划与审查 · 复用已有 skill', custom: '按职责分配 Agent · 沿用当前协作流程' };
  function hasUnsaved(id) { const draft = id && drafts.get(id); return !!draft && !!(draft.dirty || draft.saving || draft.conflict); }
  function clearSaveError(id) {
    if (!saveError || (id && saveError.id !== id)) return;
    const previous = saveError; saveError = null;
    if ($('global-error') === previous.node && previous.node.textContent === previous.message) notice('global-error', '');
  }
  function modelSaveError(id, message) {
    notice('global-error', message); saveError = { id, node: $('global-error'), message };
  }
  function allowDispatch(id, action, body) {
    if (!AssistantState.blocksDispatch(hasUnsaved(id), action, body)) return true;
    notice('global-error', 'Agent 改选尚未保存或仍有冲突。请先保存切换，再安排新工作；已有草稿和进度保留。'); return false;
  }
  function customKey() { return current ? 'task:' + current.id : 'new'; }
  function rememberCustom(plan) {
    const key = customKey(), value = AssistantState.assignmentSnapshot(plan); customAssignments.set(key, value);
    try { sessionStorage.setItem('xiaobuding.customAssignment.' + key, JSON.stringify(value)); } catch {}
  }
  function savedCustom() {
    const key = customKey();
    if (customAssignments.has(key)) return customAssignments.get(key);
    try { const value = JSON.parse(sessionStorage.getItem('xiaobuding.customAssignment.' + key) || 'null'); if (value && typeof value === 'object') { const safe = AssistantState.assignmentSnapshot(value); customAssignments.set(key, safe); return safe; } } catch {}
    return config.default_plan || {};
  }
  function view() {
    if (!current) return { plan: $('new-plan')?._plan || {}, editable: !creating, readOnly: $('read-only').checked, draft: null };
    const settings = ModelState.settingsTask(current), draft = drafts.get(current.id);
    return { plan: draft?.plan || settings.model_plan || {}, editable: settings.settings_editable && !draft?.saving && !loading, readOnly: !!current.read_only, draft, scope: settings.settings_scope };
  }
  function field(id, name) {
    const label = el('label', 'assistant-control'), text = el('span', '', name), select = el('select');
    text.className = 'sr-only'; select.id = id; select.setAttribute('aria-label', name); label.append(text, select); return { label, select };
  }
  function controls(prefix, withSave) {
    const host = el('section', 'assistant-selection'); host.setAttribute('aria-label', '选择执行 Agent 与模型');
    const mode = field(prefix + '-assistant-mode', '执行模式'), model = field(prefix + '-assistant-model', '执行模型'), effort = field(prefix + '-assistant-effort', '思考强度');
    const advanced = button('按职责设置', 'assistant-advanced', () => AgentSettings.toggle(true, advanced)); advanced.prepend(DesktopUI.icon('sliders'));
    const status = el('p', 'assistant-selection-status'); status.setAttribute('role', 'status');
    const row = el('div', 'assistant-selection-row'); row.append(mode.label, model.label, effort.label, advanced);
    const save = withSave ? button('保存切换', 'primary assistant-save', null) : null;
    if (save) { save.onclick = saveCurrent; row.append(save); }
    host.append(row, status);
    const result = { host, mode: mode.select, model: model.select, effort: effort.select, advanced, status, save };
    mode.select.onchange = () => change(mode.select.value);
    model.select.onchange = () => change(mode.select.value, model.select.value, '');
    effort.select.onchange = () => change(mode.select.value, model.select.value, effort.select.value);
    return result;
  }
  function options(select, values, chosen) {
    const signature = JSON.stringify(values);
    if (select.dataset.options !== signature) { select.dataset.options = signature; select.replaceChildren(); for (const value of values) { const option = el('option', '', value.name); option.value = value.id; option.disabled = !!value.disabled; select.append(option); } }
    select.value = chosen;
  }
  function paint(controls, state) {
    const choice = AssistantState.selection(state.plan, config.providers || []), custom = choice.mode === 'custom';
    const modeOptions = Object.entries(titles).map(([id, name]) => ({ id, name, disabled: id !== 'custom' && !(config.providers || []).some(p => p.id === id && p.available) }));
    options(controls.mode, modeOptions, choice.mode);
    const models = custom ? [{ id: '', name: '按职责分别设置模型' }] : choice.models.map(item => ({ id: item.id, name: item.name || item.id }));
    if (!custom && choice.model && !choice.modelKnown) models.push({ id: choice.model, name: choice.model + '（目录未确认）', disabled: true });
    options(controls.model, models, custom ? '' : choice.model);
    const efforts = custom ? [{ id: '', name: '按职责设置思考强度' }] : [{ id: '', name: choice.defaultEffort ? '自动 · ' + (PlanUI.effortNames[choice.defaultEffort] || choice.defaultEffort) : '自动' }, ...choice.efforts.map(id => ({ id, name: PlanUI.effortNames[id] || id }))];
    if (!custom && choice.effort && !choice.efforts.includes(choice.effort)) efforts.push({ id: choice.effort, name: choice.effort + '（目录未确认）', disabled: true });
    options(controls.effort, efforts, custom ? '' : choice.effort);
    controls.mode.disabled = !state.editable; controls.model.disabled = !state.editable || custom; controls.effort.disabled = !state.editable || custom;
    controls.advanced.disabled = loading || !!state.draft?.saving;
    if (controls.save) { controls.save.disabled = !state.editable || !state.draft?.dirty || !!state.draft?.conflict; controls.save.hidden = !state.draft?.dirty; }
    controls.host.dataset.mode = choice.mode;
    controls.host.dataset.dirty = String(!!state.draft?.dirty || !!state.draft?.conflict);
    controls.status.textContent = state.draft?.conflict ? '改选已保留。其他窗口更新了安排，请载入最新安排后再保存。' : state.draft?.saving ? '正在保存后续执行安排…' : state.draft?.dirty ? '切换尚未保存。先点“保存切换”，再发送要求或继续任务。' : current ? '下次派发交接共享项目记录 · 当前执行保持原安排' : '保留真实检查与 Codex 独立审查 · 使用原生登录、技能与插件';
  }
  function renderCards(state) {
    const parent = $('assistant-cards'); if (!parent) return;
    if (!cards.size) {
      for (const [id, title] of Object.entries(titles)) {
        const card = button('', 'assistant-card', () => change(id)); card.dataset.assistantMode = id;
        const icon = el('span', 'assistant-card-icon'); icon.setAttribute('aria-hidden', 'true');
        const mark = el('img', 'assistant-brand-mark'); mark.alt = ''; mark.draggable = false;
        mark.src = id === 'custom' ? '/reliable/icon.png' : '/api/v1/reliable/brand/' + id + '.png';
        icon.append(mark);
        const caption = el('span', 'assistant-card-text'); caption.append(el('strong', '', id === 'custom' ? '自定义工作台' : title), el('small', '', descriptions[id]));
        card.append(icon, caption, el('span', 'assistant-card-state')); parent.append(card); cards.set(id, card);
      }
    }
    for (const [id, card] of cards) {
      const provider = (config.providers || []).find(item => item.id === id), available = id === 'custom' || !!provider?.available;
      card.disabled = !state.editable || !available; card.querySelector('.assistant-card-text small').textContent = id === 'zcode' && state.readOnly ? 'ZCode 原生只读规划 · 仅开放读取工具' : descriptions[id]; card.setAttribute('aria-pressed', String(AssistantState.mode(state.plan) === id));
      card.querySelector('.assistant-card-state').textContent = available ? AssistantState.mode(state.plan) === id ? '已选择' : '选择' : '待连接';
    }
  }
  function renderNew() {
    if (!$('new-plan')?._plan) return;
    if (!newControls) { newControls = controls('new', false); $('new-assistant-selection').append(newControls.host); }
    const plan = $('new-plan')._plan, state = { plan, editable: !creating, readOnly: $('read-only').checked, draft: null };
    paint(newControls, state); renderCards(state);
  }
  function sync(task) {
    if (current?.id !== task?.id) clearSaveError();
    current = task || null; loading = false;
    if (!current) { renderNew(); return; }
    const target = $('workspace-assistant-selection');
    if (!target) return;
    if (!taskControls || !taskControls.host.isConnected) { taskControls = controls('task', true); target.replaceChildren(taskControls.host); }
    paint(taskControls, view());
  }
  function change(mode, model, effort) {
    const state = view(); if (!state.editable) return;
    try {
      const same = AssistantState.mode(state.plan) === mode, choice = AssistantState.selection(state.plan, config.providers || []), selectedModel = model ?? (same ? choice.model : undefined), selectedEffort = effort ?? (same ? choice.effort : undefined);
      if (AssistantState.mode(state.plan) === 'custom' && mode !== 'custom') rememberCustom(state.plan);
      const next = mode === 'custom' ? AssistantState.restoreCustom(state.plan, same ? state.plan : savedCustom()) : AssistantState.apply(state.plan, mode, config.providers || [], selectedModel, selectedEffort, state.readOnly);
      if (mode === 'custom') rememberCustom(next);
      if (!current) { renderNewPlan(next); return; }
      const settings = ModelState.settingsTask(current), draft = ModelState.reconcileDraft(drafts.get(current.id), settings, config.default_plan);
      draft.plan = next; draft.dirty = true; drafts.set(current.id, draft);
      $('task-plan-editor').dataset.task = ''; detailSignature = ''; render();
    } catch (error) { notice(current ? 'global-error' : 'form-error', friendly(error.message)); sync(current); }
  }
  async function saveCurrent() {
    const owner = current?.id; if (!owner) return;
    const promise = savePlan(owner); sync(tasks.find(item => item.id === owner));
    await promise; if (current?.id === owner) sync(tasks.find(item => item.id === owner));
    TaskConversation.sync(tasks.find(item => item.id === selected));
  }
  function setLoading() { loading = true; if (taskControls) for (const node of taskControls.host.querySelectorAll('button,select')) node.disabled = true; }
  return { sync, renderNew, hasUnsaved, allowDispatch, modelSaveError, modelSaveSucceeded: clearSaveError, setLoading, setCreating: () => renderNew() };
})();
