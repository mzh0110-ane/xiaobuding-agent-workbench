'use strict';
const WorkbenchUI = (() => {
  const previousStates = new Map();
  let notificationInit = false;
  function guidance(parent, task) {
    if (['queued', 'preparing', 'auditing', 'implementing', 'checking', 'reviewing', 'applying', 'cancelling'].includes(task.state)) {
      const previous = el('details', 'guidance-raw'); previous.append(el('summary', '', '上一轮记录（当前流程正在继续）'), el('pre', '', task.reason)); parent.append(previous); return;
    }
    const data = task.recovery || { summary: friendly(task.reason), next_action: 'none' };
    const box = el('section', 'guidance-banner'); box.dataset.severity = 'warning';
    const completedSourceChange = data.code === 'source_changed' && task.round >= 3;
    const heading = completedSourceChange ? (/outside audited scope/i.test(task.reason || '') ? '有文件超出本批审查范围，任务已暂停。' : '本批源码或范围发生变化，任务已暂停。') : data.summary;
    box.append(el('strong', '', heading));
    if (data.code === 'source_changed') {
      const match = String(task.reason || '').match(/change outside audited scope:\s*([^\r\n]+)/i);
      if (match) box.append(el('p', '', '需要核对的文件：' + match[1]));
      box.append(el('p', '', task.worktree ? '已有修改保存在工作副本，不会自动应用到原项目。请点击上方“查看工作副本”，核对文件变化和来源。' : '已有任务记录保留，请先核对涉及文件的变化和来源。'));
      if (task.round >= 3) box.append(el('p', '', '本批已用三次自动尝试。核对后，如上方提供“保留修改，开始下一批”，可从那里继续；仍会检查继承的修改。'));
      else box.append(el('p', '', '核对后可展开下方“核对当前副本并重新规划”，按提示确认；现有授权范围不会自动扩大。'));
    }
    if (task.read_only && ['blocked', 'interrupted'].includes(task.state)) {
      box.append(el('p', '', '这是只读调查，原任务暂停不代表后续提问失败；已回答的内容可在“对话”中查看。处理原因后可点上方“处理后继续”。'));
    }
    if (['replan', 'checks'].includes(data.next_action)) {
      box.append(button(data.next_action === 'replan' ? (completedSourceChange ? '查看项目记录' : '查看差异与项目范围') : '查看失败检查', '', () => WorkspaceUI.show(data.next_action === 'replan' ? 'project' : 'results')));
    }
    if (data.next_action === 'replan' && task.round < 3 && !task.successor_task_id && !task.archived_at) replan(box, task);
    if (data.next_action === 'login') box.append(el('p', '', '回到对应原生工具完成登录后，再点上方继续。小布丁复用原有账号。'));
    const raw = el('details', 'guidance-raw'); raw.append(el('summary', '', '原始诊断记录'), el('pre', '', data.detail || task.reason)); box.append(raw); parent.append(box);
  }
  function replan(parent, task) {
    const details = el('details', 'guidance-replan'); details.append(el('summary', '', '核对当前副本并重新规划')); const body = el('div'); details.append(body); parent.append(details);
    details.ontoggle = async () => {
      if (!details.open || details.dataset.loaded) return; details.dataset.loaded = 'true'; body.textContent = '正在核对变化与原始授权范围…';
      try {
        const value = await api('tasks/' + task.id + '/replan'); if (!details.isConnected) return;
        body.replaceChildren(el('p', 'hint', '原授权范围：' + (value.allowed_paths || []).join('、')));
        const list = el('ul'); for (const file of (value.changed_files || []).slice(0, 100)) list.append(el('li', '', file)); body.append(list);
        body.append(el('p', 'hint', '共 ' + (value.total_changes || 0) + ' 处变化，列表最多显示 100 项，可在工作副本核对完整文件。此操作保留已用轮次，不扩大原授权范围。'));
        if (!value.eligible) { body.append(el('p', 'hint', friendly(value.reason))); return; }
        const form = el('form'), caption = el('label', 'replan-confirmation'), check = el('input'); check.type = 'checkbox'; check.required = true;
        caption.append(check, document.createTextNode('我已核对这些变化，确认以当前副本重新规划。'));
        const note = el('textarea'); note.setAttribute('aria-label', '重新规划说明（可选）'); note.maxLength = 4000; note.placeholder = '可以说明这些变化的来源，以及接下来应保留什么。';
        const submit = button('确认变化并重新规划', '', null); submit.type = 'submit'; form.append(caption, note, submit); body.append(form);
        form.onsubmit = async event => {
          event.preventDefault(); if (!check.checked || submit.disabled) return; submit.disabled = true;
          if (typeof AssistantUI !== 'undefined' && !AssistantUI.allowDispatch(task.id, 'replan')) { submit.disabled = false; return; }
          try { const result = await api('tasks/' + task.id + '/replan', { source_hash: value.source_hash, confirm: true, note: note.value.trim() }); if (details.isConnected) body.append(el('p', 'hint', result.already_confirmed ? '此版本已确认，可用上方本批继续操作。没有重复启动任务。' : '已保存确认记录，正在重新规划。')); await refresh(); }
          catch (error) { if (details.isConnected) body.append(el('p', 'notice error', friendly(error.message))); }
          finally { submit.disabled = false; }
        };
      } catch (error) { if (details.isConnected) { body.textContent = friendly(error.message); details.dataset.loaded = ''; } }
    };
  }
  function execution(parent, task) {
    const run = [...(task.runs || [])].reverse().find(item => ['running', 'preparing'].includes(item.status));
    if (!run) return;
    const line = el('p', 'summary-execution', roleTitle(run.role, run.domain) + ' · ' + (run.provider_name || run.provider_id) + ' · ' + run.model + (run.effort ? ' · ' + run.effort : ''));
    const last = el('span', 'hint', '可在“执行过程”查看实际工具调用与输出。'); last.id = 'summary-last-action'; line.append(last); parent.append(line);
  }
  function notify(tasks) {
    for (const task of tasks) {
      const old = previousStates.get(task.id); previousStates.set(task.id, task.state);
      if (!WorkbenchState.shouldNotify(old, task.state)) continue;
      if (typeof Notification !== 'undefined' && Notification.permission === 'granted' && preference() === 'on') {
        const note = new Notification('小布丁 · ' + (states[task.state] || task.state), { body: task.goal.slice(0, 100), tag: 'xbd-' + task.id });
        note.onclick = () => { window.focus(); selectTask(task.id); note.close(); };
      }
    }
    if (notificationInit || !$('notification-toggle')) return; notificationInit = true;
    $('notification-toggle').onclick = async () => {
      if (typeof Notification === 'undefined') { notice('global-error', '当前浏览器不支持桌面提醒，任务状态仍会在工作台显示。'); return; }
      if (preference() === 'on') { preference('off'); $('notification-toggle').textContent = '开启提醒'; return; }
      const result = await Notification.requestPermission();
      if (result === 'granted') { preference('on'); $('notification-toggle').textContent = '提醒已开启'; }
      else notice('global-error', '桌面提醒未启用，任务状态仍会在工作台显示。');
    };
    $('notification-toggle').textContent = preference() === 'on' ? '提醒已开启' : '开启提醒';
  }
  function preference(value) { try { if (value) localStorage.setItem('xiaobuding.notifications', value); return localStorage.getItem('xiaobuding.notifications'); } catch { return 'off'; } }
  return { guidance, execution, notify };
})();
