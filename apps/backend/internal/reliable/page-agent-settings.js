'use strict';
const AgentSettings = (() => {
  let panel, body, tabs, context, comparison, backdrop, task = null, opened = false, opener = null, role = '', tabSignature = '';
  const rolesByContext = new Map(), modal = matchMedia('(max-width:1100px)');
  function make(tag, cls, text) { return el(tag, cls, text); }
  function selectionSummary(current, key, plan, providers) {
    const activeStates = ['queued','preparing','auditing','implementing','checking','reviewing'];
    const running = current && (!current.state || activeStates.includes(current.state)) ? [...(current.runs || [])].reverse().find(run => ['running','preparing'].includes(run.status)) : null;
    const actual = running ? { role: running.domain || running.role, provider: running.provider_name || running.provider_id || '未记录工具', model: running.model || '未记录模型', effort: running.effort || '', status: running.status } : null;
    const provider = (providers || []).find(item => item.id === plan?.[key]);
    return { current: actual, actual: actual?.role === key ? actual : null, upcoming: { role: key, provider: provider?.name || plan?.[key] || '未选择工具', model: plan?.[key + '_model'] || provider?.model || '沿用原工具设置', effort: plan?.[key + '_effort'] || '' } };
  }
  function showSelection(prefix) {
    const summary = selectionSummary(task, role, $(prefix)?._plan, config.providers || []), dirty = !!task && !!drafts.get(task.id)?.dirty;
    comparison.replaceChildren();
    const actual = make('div', 'agent-selection-current'), upcoming = make('div', 'agent-selection-upcoming');
    actual.append(make('strong', '', '当前实际执行'));
    if (summary.current) {
      const value = summary.current;
      actual.append(make('p', '', (PlanUI.titles[value.role] || value.role) + ' · ' + value.provider), make('p', 'hint', value.model + ' · 思考：' + (PlanUI.effortNames[value.effort] || value.effort || '执行记录未提供')));
      if (value.status === 'preparing') actual.append(make('small', 'hint', '已安排，正在准备派发。'));
    } else actual.append(make('p', 'hint', task ? '当前没有 Agent 正在运行。' : '开始后显示真实派发记录。'));
    upcoming.append(make('strong', '', (dirty ? '未保存的改选' : task ? '下次派发的安排' : '开始时的安排') + ' · ' + (PlanUI.titles[role] || role)));
    upcoming.append(make('p', '', summary.upcoming.provider + ' · ' + summary.upcoming.model), make('p', 'hint', '思考：' + (PlanUI.effortNames[summary.upcoming.effort] || summary.upcoming.effort || '自动，派发时确定')));
    upcoming.append(make('small', 'hint', dirty ? '保存后才用于后续派发，当前执行保持原设置。' : task ? '已完成的执行记录保留；改选不会即时替换正在运行的 Agent。' : '使用现有原生账号、技能和插件。'));
    comparison.append(actual, upcoming);
  }
  function availableRoles() {
    const parent = document.getElementById(task ? 'task-plan' : 'new-plan');
    if (!parent?._plan) return [];
    return (!task && $('read-only').checked) || (task?.read_only && ModelState.settingsTask(task).settings_scope !== 'future') ? ['auditor'] : PlanUI.keys(parent._plan);
  }
  function reflectMode() {
    if (!panel) return;
    const isModal = opened && modal.matches;
    document.querySelector('.app-layout').inert = isModal;
    backdrop.hidden = !isModal;
    panel.setAttribute('role', modal.matches ? 'dialog' : 'complementary');
    if (isModal) panel.setAttribute('aria-modal', 'true'); else panel.removeAttribute('aria-modal');
    document.documentElement.dataset.agentSettings = opened ? 'open' : 'closed';
    for (const id of ['agent-settings-toggle','agent-settings-sidebar']) $(id).setAttribute('aria-expanded', String(opened));
  }
  function toggle(next = !opened, trigger = null) {
    if (next && !opened) { opener = trigger || document.activeElement; DesktopUI.closeMenu(false); DesktopUI.drawer(false); }
    opened = next; panel.hidden = !opened; reflectMode();
    if (opened) {
      sync(task); $('agent-settings-close').focus({ preventScroll: true });
      if ((config.providers || []).some(p => (p.tool_kind === 'codex' || p.id === 'codex') && (p.catalog_error || p.catalog_source === 'codex_cache_history' || p.models?.some(m => m.name?.includes('已配置，未验证'))))) void refreshModels(true);
    } else {
      const target = opener?.isConnected && opener.getClientRects().length && !opener.closest('[inert]') ? opener : $('agent-settings-toggle');
      target.focus({ preventScroll: true });
    }
  }
  function chooseRole(next) {
    role = next; rolesByContext.set(task?.id || 'new', role); sync(task);
  }
  function renderTabs(keys, prefix) {
    const signature = prefix + ':' + keys.join(',');
    if (signature !== tabSignature) {
      tabSignature = signature; tabs.replaceChildren();
      for (const key of keys) {
        const button = make('button', '', ({frontend:'前端',backend:'后端',auditor:'规划',reviewer:'审查',implementer:'实施'})[key]);
        button.type = 'button'; button.dataset.agentRole = key; button.id = 'agent-role-' + key; button.setAttribute('role','tab'); button.setAttribute('aria-controls',prefix + '-role-' + key); button.onclick = () => chooseRole(key); tabs.append(button);
      }
    }
    for (const button of tabs.children) { const active = button.dataset.agentRole === role; button.setAttribute('aria-selected', String(active)); button.tabIndex = active ? 0 : -1; }
    const fields = $(prefix);
    for (const row of fields?.querySelectorAll('.assignment-row') || []) {
      const key = [...row.querySelectorAll('select')].map(select => select.id.slice(prefix.length + 1)).find(id => keys.includes(id));
      row.hidden = key !== role; row.id = prefix + '-role-' + key; row.setAttribute('role','tabpanel'); row.setAttribute('aria-labelledby','agent-role-' + key);
    }
  }
  function sync(current) {
    if (!panel) return;
    for (const id of ['agent-settings-toggle','agent-settings-sidebar']) $(id).disabled = false;
    if (current?.id !== task?.id) { task = current || null; role = rolesByContext.get(task?.id || 'new') || (task?.read_only ? 'auditor' : ''); }
    else task = current || null;
    $('new-agent-settings').hidden = !!task; $('task-agent-settings').hidden = !task;
    context.textContent = task ? (task.read_only ? '项目追问与后续工作' : '项目协作') + ' · ' + DesktopUI.folderName(task.project) : '新任务 · 开始时使用这些安排';
    const keys = availableRoles(), prefix = task ? 'task-plan' : 'new-plan';
    if (!keys.includes(role)) role = task?.read_only && keys.includes('auditor') ? 'auditor' : keys[0] || '';
    renderTabs(keys, prefix);
    markDirty(!!task && !!drafts.get(task.id)?.dirty);
  }
  function markDirty(dirty) {
    if (!panel) return;
    for (const id of ['agent-settings-toggle','agent-settings-sidebar']) { $(id).dataset.dirty = String(dirty); $(id).setAttribute('aria-label', dirty ? 'Agent 设置，有未保存的改选' : 'Agent 设置'); }
    if (comparison) showSelection(task ? 'task-plan' : 'new-plan');
    AssistantUI.sync(task);
  }
  function keyboard(event) {
    if (!opened) return;
    if (event.key === 'Escape') { event.preventDefault(); event.stopImmediatePropagation(); toggle(false); return; }
    const tab = event.target.closest?.('[data-agent-role]');
    if (tab && ['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) {
      const keys = availableRoles(), index = keys.indexOf(role), next = event.key === 'Home' ? 0 : event.key === 'End' ? keys.length - 1 : (index + (event.key === 'ArrowLeft' ? -1 : 1) + keys.length) % keys.length;
      event.preventDefault(); chooseRole(keys[next]); $('agent-role-' + role).focus(); return;
    }
    if (!modal.matches || event.key !== 'Tab') return;
    const controls = [...panel.querySelectorAll('button:not(:disabled),input:not(:disabled),select:not(:disabled),a,summary,[tabindex="0"]')].filter(node => node.getClientRects().length && node.tabIndex !== -1 && !node.closest('details:not([open]) > :not(summary)'));
    const first = controls[0], last = controls.at(-1);
    if (event.shiftKey && (document.activeElement === first || !panel.contains(document.activeElement))) { event.preventDefault(); last?.focus(); }
    else if (!event.shiftKey && (document.activeElement === last || !panel.contains(document.activeElement))) { event.preventDefault(); first?.focus(); }
  }
  function init() {
    panel = make('aside','agent-settings-drawer'); panel.id = 'agent-settings-drawer'; panel.hidden = true; panel.setAttribute('aria-labelledby','agent-settings-title');
    const heading = make('div','agent-settings-heading'), title = make('h2','','Agent 设置'); title.id = 'agent-settings-title';
    const close = make('button','icon-button',''); close.type = 'button'; close.id = 'agent-settings-close'; close.setAttribute('aria-label','收起 Agent 设置'); close.append(DesktopUI.icon('chevron')); close.onclick = () => toggle(false);
    const refresh = $('refresh-models'); heading.append(title,refresh,close);
    context = make('p','agent-settings-context'); tabs = make('div','agent-role-tabs'); tabs.setAttribute('role','tablist'); tabs.setAttribute('aria-label','选择 Agent 职责');
    body = make('div','agent-settings-body'); const newHost = make('fieldset'), taskHost = make('div'); newHost.id = 'new-agent-settings'; taskHost.id = 'task-agent-settings';
    for (const [id, host] of [['assignment-settings',newHost],['task-plan-settings',taskHost]]) { const section = $(id); section.open = true; host.append(section); }
    comparison = make('section', 'agent-selection-comparison'); comparison.setAttribute('aria-label', '实际执行与后续安排');
    body.addEventListener('change', () => showSelection(task ? 'task-plan' : 'new-plan'));
    body.append(newHost,taskHost); panel.append(heading,context,comparison,tabs,body);
    backdrop = make('button','agent-settings-backdrop'); backdrop.type = 'button'; backdrop.hidden = true; backdrop.setAttribute('aria-label','关闭 Agent 设置'); backdrop.tabIndex = -1; backdrop.onclick = () => toggle(false);
    document.body.append(backdrop,panel);
    for (const id of ['agent-settings-toggle','agent-settings-sidebar']) $(id).onclick = event => toggle(!opened, event.currentTarget);
    document.addEventListener('keydown',keyboard,true); modal.addEventListener('change',reflectMode); sync(null); reflectMode();
  }
  function setLoading() { if (!panel) return; if (opened) toggle(false); for (const id of ['agent-settings-toggle','agent-settings-sidebar']) $(id).disabled = true; }
  return { init, sync, toggle, markDirty, selectionSummary, setLoading, setCreating: busy => { $('new-agent-settings').disabled = busy; } };
})();
