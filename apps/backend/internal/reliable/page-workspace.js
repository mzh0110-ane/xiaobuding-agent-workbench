'use strict';
const WorkspaceUI = (() => {
  const tabs = ['conversation', 'activity', 'results', 'project'], views = new Map();
  let taskID = '', currentTab = 'conversation', initialized = false, changing = false;
  const compact = matchMedia('(max-width:1100px)');
  let contextPreferences = { wide: false, compact: false };
  function reflectContext() {
    const control = $('task-context-toggle'), agentOpen = document.documentElement.dataset.agentSettings === 'open';
    const open = !!taskID && contextPreferences[compact.matches ? 'compact' : 'wide'] && !agentOpen;
    document.body.dataset.contextPanel = open ? 'open' : 'closed';
    control.hidden = !taskID; control.disabled = agentOpen;
    control.setAttribute('aria-expanded', String(open));
    control.title = open ? '收起本批概览' : '查看状态、恢复、验收与阶段规则';
  }
  function toggleContext() {
    const key = compact.matches ? 'compact' : 'wide';
    contextPreferences[key] = !contextPreferences[key];
    try { localStorage.setItem('xiaobuding.contextPanel', JSON.stringify(contextPreferences)); } catch {}
    reflectContext();
  }
  function view(id) {
    if (!views.has(id)) {
      let saved = ''; try { saved = sessionStorage.getItem('xiaobuding.workspace.' + id) || ''; } catch {}
      views.set(id, { tab: tabs.includes(saved) ? saved : 'conversation', scroll: {} });
    }
    return views.get(id);
  }
  function remember() { if (taskID && !changing) view(taskID).scroll[currentTab] = $('workspace-content').scrollTop; }
  function prepare(task) { if ((task?.id || '') !== taskID) { remember(); changing = true; } }
  function show(next, focus = false, restoring = false) {
    if (!tabs.includes(next)) return;
    if (!restoring) remember(); currentTab = next;
    if (taskID) {
      view(taskID).tab = next;
      try { sessionStorage.setItem('xiaobuding.workspace.' + taskID, next); } catch {}
    }
    for (const tab of tabs) {
      const active = tab === next, button = $('tab-' + tab);
      button.setAttribute('aria-selected', String(active)); button.tabIndex = active ? 0 : -1;
      $('panel-' + tab).hidden = !active;
    }
    $('workspace-content').scrollTop = taskID ? view(taskID).scroll[next] || 0 : 0;
    if (next === 'project') ProjectUI.show();
    if (next === 'results') renderProof(tasks.find(item => item.id === selected));
    if (focus) $('tab-' + next).focus();
  }
  function width(value) {
    document.body.dataset.workspaceWidth = value === 'portrait' ? 'portrait' : 'wide';
    const wide = document.body.dataset.workspaceWidth === 'wide', control = $('workspace-width');
    control.textContent = wide ? '展示人物' : '宽屏工作';
    control.setAttribute('aria-pressed', String(wide));
    control.title = wide ? '留出右侧人物背景的位置' : '让工作区使用更多空间';
  }
  function init() {
    if (initialized) return; initialized = true;
    try {
      const savedContext = JSON.parse(localStorage.getItem('xiaobuding.contextPanel') || '{}');
      for (const key of ['wide', 'compact']) if (typeof savedContext[key] === 'boolean') contextPreferences[key] = savedContext[key];
    } catch {}
    $('task-context-toggle').onclick = toggleContext;
    compact.addEventListener('change', reflectContext);
    new MutationObserver(reflectContext).observe(document.documentElement, { attributes: true, attributeFilter: ['data-agent-settings'] });
    document.addEventListener('keydown', event => {
      if (event.key !== 'Escape' || document.body.dataset.contextPanel !== 'open' || !event.target.closest?.('#task-summary, #task-context-toggle')) return;
      event.preventDefault(); toggleContext(); $('task-context-toggle').focus();
    });
    let saved; try { saved = localStorage.getItem('xiaobuding.workspaceWidth'); } catch {}
    width(saved);
    $('workspace-width').onclick = () => {
      width(document.body.dataset.workspaceWidth === 'wide' ? 'portrait' : 'wide');
      try { localStorage.setItem('xiaobuding.workspaceWidth', document.body.dataset.workspaceWidth); } catch {}
    };
    for (const tab of tabs) $('tab-' + tab).onclick = () => show(tab);
    $('workspace-tabs').addEventListener('keydown', event => {
      if (!event.target.matches('[data-workspace-tab]') || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
      event.preventDefault(); const index = tabs.indexOf(event.target.dataset.workspaceTab);
      const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowLeft' ? -1 : 1) + tabs.length) % tabs.length;
      show(tabs[next], true);
    });
    $('workspace-content').addEventListener('scroll', remember, { passive: true });
  }
  function sync(task) {
    init(); document.body.classList.toggle('has-task', !!task);
    const attention = ({ blocked: '需要处理', interrupted: '进度已保留', exhausted: '继续下一批', awaiting_acceptance: '等待验收' })[task?.state];
    $('task-context-toggle').dataset.attention = String(!!attention);
    $('task-context-label').textContent = attention || '本批概览';
    $('task-context-toggle').setAttribute('aria-label', '本批概览' + (attention ? '，' + attention : ''));
    if ((task?.id || '') === taskID) { reflectContext(); return; }
    remember(); taskID = task?.id || '';
    show(taskID ? view(taskID).tab : 'conversation', false, true);
    changing = false;
    reflectContext();
  }
  return { prepare, sync, show, current: () => currentTab };
})();
