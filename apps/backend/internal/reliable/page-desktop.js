'use strict';
const DesktopUI = (() => {
  const paths = {
    plus: 'M12 5v14M5 12h14', search: 'M21 21l-5-5M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
    refresh: 'M20 7v5h-5M4 17v-5h5M6 7a7 7 0 0 1 12-2l2 3M4 16l2 3a7 7 0 0 0 12-2',
    folder: 'M3 7V5h6l2 2h10v13H3V7Z', more: 'M5 12h.01M12 12h.01M19 12h.01',
    sun: 'M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1 1M18 18l1 1M5 19l1-1M18 6l1-1M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0',
    sidebar: 'M3 4h18v16H3V4ZM9 4v16', external: 'M14 3h7v7M21 3l-9 9M10 3H3v18h18v-7',
    'arrow-up': 'M12 19V5M6 11l6-6 6 6', shield: 'M12 3l8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3ZM8 12l3 3 5-6',
    sliders: 'M4 7h5M13 7h7M4 17h9M17 17h3M9 4v6M17 14v6', chevron: 'M9 5l7 7-7 7',
    code: 'M8 5l-6 7 6 7M16 5l6 7-6 7M14 3l-4 18', play: 'M8 4l12 8-12 8V4Z',
    archive: 'M3 3h18v5H3V3ZM5 8v13h14V8M10 12h4', restore: 'M4 10a8 8 0 1 1 1 8M4 4v6h6',
    file: 'M6 3h8l4 4v14H6V3ZM14 3v5h4M9 12h6M9 16h6',
    image: 'M3 3h18v18H3V3ZM3 16l6-6 5 5 3-3 4 4M15 7h.01',
  };
  const archiveStates = ['inspected', 'accepted', 'cancelled', 'exhausted', 'blocked', 'interrupted', 'awaiting_acceptance'];
  let view = 'active', query = '', menu = null, menuTask = '', menuTrigger = null, handlers;
  const mobile = window.matchMedia('(max-width:760px)');
  function icon(name) {
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('viewBox', '0 0 24 24'); svg.setAttribute('class', 'icon'); svg.setAttribute('aria-hidden', 'true');
    const path = document.createElementNS(svg.namespaceURI, 'path'); path.setAttribute('d', paths[name] || paths.file); svg.append(path); return svg;
  }
  function hydrate(root = document) { for (const node of root.querySelectorAll('[data-icon]')) node.replaceChildren(icon(node.dataset.icon)); }
  function folderName(path) { return String(path || '').replace(/[\\/]+$/, '').split(/[\\/]/).at(-1) || '项目'; }
  function shortTool(id) { return id === 'deepseek' ? 'Harness' : id === 'codex' ? 'Codex' : id || '未选择'; }
  function planSummary(plan) { return PlanUI.keys(plan).map(key => shortTool(plan[key]) + ' ' + ({ frontend: '前端', backend: '后端', auditor: '规划', reviewer: '审查', implementer: '实施' })[key]).join(' · '); }
  function visibleTasks(all) { return all.filter(task => !!task.archived_at === (view === 'archived') && (!query || task.search_match || (task.goal + ' ' + task.project).toLocaleLowerCase().includes(query))); }
  function counts(all) {
    $('active-count').textContent = all.filter(task => !task.archived_at).length;
    const archived = all.filter(task => task.archived_at), pending = archived.filter(task => task.state === 'awaiting_acceptance').length;
    $('archived-count').textContent = archived.length + (pending ? ' · ' + pending + ' 待验收' : '');
    $('archive-hint').hidden = view !== 'archived'; $('history-title').textContent = view === 'archived' ? '已归档的任务' : '最近任务';
    $('task-list').setAttribute('aria-labelledby', 'filter-' + view);
  }
  function setView(next) {
    view = next;
    for (const id of ['active', 'archived']) $('filter-' + id).setAttribute('aria-selected', String(id === view));
    closeMenu(false); handlers.changed();
  }
  function closeMenu(restore = false) {
    if (!menu) return;
    menu.remove(); menu = null; menuTask = '';
    if (menuTrigger) menuTrigger.setAttribute('aria-expanded', 'false');
    if (restore) (menuTrigger?.isConnected ? menuTrigger : $('new-task')).focus();
    menuTrigger = null;
  }
  function menuButton(label, action, symbol, disabled = false) {
    const item = el('button'); item.type = 'button'; item.setAttribute('role', 'menuitem'); item.dataset.action = action; item.disabled = disabled;
    item.append(icon(symbol), el('span', '', label));
    item.onclick = () => { const id = menuTask; closeMenu(true); handlers.action(id, action); };
    menu.append(item); return item;
  }
  function showMenu(id, trigger, point) {
    const task = handlers.tasks().find(item => item.id === id); if (!task) return;
    closeMenu(false); menuTask = id; menuTrigger = trigger; trigger.setAttribute('aria-expanded', 'true');
    menu = el('div', 'context-menu'); menu.id = 'task-context-menu'; menu.setAttribute('role', 'menu'); menu.setAttribute('aria-label', '任务操作');
    menu.append(el('p', 'menu-title', task.goal));
    menuButton('打开原项目文件夹', 'open-project', 'folder');
    const work = menuButton('打开工作副本', 'open', 'code', !task.worktree); if (!task.worktree) work.title = '独立工作副本尚未准备好';
    menuButton('查看证据文件夹', 'open-evidence', 'file'); menu.append(el('div', 'menu-divider'));
    menuButton(task.archived_at ? '恢复到任务列表' : '归档任务', task.archived_at ? 'unarchive' : 'archive', task.archived_at ? 'restore' : 'archive', !task.archived_at && !archiveStates.includes(task.state));
    const note = task.archived_at ? '恢复只改变列表位置，不会自动启动任务。' : !archiveStates.includes(task.state) ? '当前正在执行，结束或停止后才能归档。' : task.state === 'awaiting_acceptance' ? '仍待验收。归档不会应用修改，可恢复后验收。' : '归档仅收纳，不删除项目或证据。';
    menu.append(el('p', 'menu-note', note)); document.body.append(menu);
    const box = menu.getBoundingClientRect(), anchor = trigger.getBoundingClientRect();
    menu.style.left = Math.max(8, Math.min(point?.x ?? anchor.left, innerWidth - box.width - 8)) + 'px';
    menu.style.top = Math.max(8, Math.min(point?.y ?? anchor.bottom + 5, innerHeight - box.height - 8)) + 'px';
    menu.querySelector('button:not(:disabled)')?.focus();
  }
  function syncMenu() {
    if (!menu) return;
    const task = handlers.tasks().find(item => item.id === menuTask);
    if (!task) { closeMenu(false); return; }
    const archive = menu.querySelector('[data-action="archive"]'); if (archive) archive.disabled = !archiveStates.includes(task.state);
  }
  function drawer(open, returnFocus = false) {
    const sidebar = $('sidebar'), focusWasInside = sidebar.contains(document.activeElement); sidebar.classList.toggle('is-open', open && mobile.matches);
    $('sidebar-toggle').setAttribute('aria-expanded', String(open && mobile.matches));
    $('sidebar-backdrop').hidden = !open || !mobile.matches;
    sidebar.inert = mobile.matches && !open; document.querySelector('.main-pane').inert = mobile.matches && open;
    if (open && mobile.matches) $('new-task').focus(); else if ((returnFocus || focusWasInside) && mobile.matches) $('sidebar-toggle').focus();
  }
  function keyboard(event) {
    if (menu) {
      const items = [...menu.querySelectorAll('button:not(:disabled)')], index = items.indexOf(document.activeElement);
      if (event.key === 'Escape') { event.preventDefault(); closeMenu(true); return; }
      if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
        event.preventDefault(); const next = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (index + (event.key === 'ArrowUp' ? -1 : 1) + items.length) % items.length;
        items[next]?.focus(); return;
      }
      if (event.key === 'Tab') closeMenu(true);
    }
    const row = event.target.closest?.('[data-task-row]');
    if (row && (event.key === 'ContextMenu' || (event.shiftKey && event.key === 'F10'))) { event.preventDefault(); showMenu(row.dataset.taskRow, event.target.closest('button') || row); return; }
    if (mobile.matches && $('sidebar').classList.contains('is-open')) {
      if (event.key === 'Escape') { event.preventDefault(); drawer(false, true); }
      if (event.key === 'Tab') {
        const focusable = [...$('sidebar').querySelectorAll('button:not(:disabled),input:not(:disabled),select:not(:disabled),a,summary')].filter(node => node.getClientRects().length && !node.closest('details:not([open]) > :not(summary)')), first = focusable[0], last = focusable.at(-1);
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
      }
    }
  }
  function init(callbacks) {
    handlers = callbacks; hydrate(); AppearanceUI.init();
    $('task-search').oninput = event => { query = event.target.value.trim().toLocaleLowerCase(); handlers.changed(); };
    for (const id of ['active', 'archived']) $('filter-' + id).onclick = () => setView(id);
    $('theme').value = document.documentElement.dataset.theme || 'system';
    $('theme').onchange = event => { const value = event.target.value; document.documentElement.dataset.theme = value; try { localStorage.setItem('xiaobuding.theme', value); } catch {} };
    $('sidebar-toggle').onclick = () => drawer(!$('sidebar').classList.contains('is-open'));
    $('sidebar-backdrop').onclick = () => drawer(false, true); mobile.addEventListener('change', () => drawer(false)); drawer(false);
    $('task-more').onclick = event => showMenu(handlers.selected(), event.currentTarget);
    $('task-list').addEventListener('contextmenu', event => {
      const row = event.target.closest('[data-task-row]'); if (!row) return;
      event.preventDefault(); showMenu(row.dataset.taskRow, row.querySelector('[data-task-select]'), { x: event.clientX, y: event.clientY });
    });
    document.addEventListener('keydown', keyboard);
    document.addEventListener('pointerdown', event => { if (menu && !menu.contains(event.target) && !event.target.closest('[data-task-menu],#task-more')) closeMenu(false); });
    $('main-content').addEventListener('scroll', () => closeMenu(false), { passive: true });
    $('task-list').addEventListener('scroll', () => closeMenu(false), { passive: true }); window.addEventListener('resize', () => closeMenu(false));
    $('task-search').addEventListener('keydown', event => { if (event.key === 'Escape') { event.target.value = ''; query = ''; handlers.changed(); } });
  }
  return { init, icon, hydrate, folderName, planSummary, visibleTasks, counts, showMenu, syncMenu, closeMenu, drawer, filterKey: () => view + ':' + query };
})();
