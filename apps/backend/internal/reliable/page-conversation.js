'use strict';
const TaskConversation = (() => {
  let task = null, host, list, content, kind, send, hint, error, requirement, targetGroup, targetPath, targetBrowse, signature = ''; 
  const drafts = new Map(), pending = new Set(), nextBatches = new Set(), batchNonces = new Map();
  const historyPages = new Map(); let conversationEpoch = 0;
  const delivery = { queued: '已记录，等待安全节点', reading: '正在读取项目并回答', answered: '已回答', failed: '未完成', interrupted: '已中断，请重新发送', cancelled: '已停止', applied: '已纳入本批计划', linked: '已开启下一批' };
  function make(tag, cls, text) { const n = document.createElement(tag); if (cls) n.className = cls; if (text !== undefined) n.textContent = text; return n; }
  function draft(id) {
    if (!drafts.has(id)) {
      let saved = {}; try { saved = JSON.parse(sessionStorage.getItem('xiaobuding.followup.' + id) || '{}'); } catch {}
      drafts.set(id, { content: saved.content || '', kind: saved.kind === 'requirement' ? 'requirement' : 'question', nonce: saved.nonce || crypto.randomUUID(), project: saved.project || '', attachment_ids: saved.attachment_ids || [] });
    }
    return drafts.get(id);
  }
  function saveDraft() {
    if (!task) return;
    const current = draft(task.id); const attachmentIDs = AttachmentUI.ids('task:' + task.id); if (current.content !== content.value || current.kind !== kind.value || current.project !== targetPath.value || JSON.stringify(current.attachment_ids || []) !== JSON.stringify(attachmentIDs)) current.nonce = crypto.randomUUID(); current.content = content.value; current.kind = kind.value; current.project = targetPath.value; current.attachment_ids = attachmentIDs;
    try { sessionStorage.setItem('xiaobuding.followup.' + task.id, JSON.stringify(current)); } catch {}
    updateControls();
  }
  function build() {
    host = document.getElementById('task-conversation'); host.replaceChildren();
    host.append(make('h2', '', '项目对话'));
    const original = make('details', 'task-requirement');
    requirement = make('p', 'requirement-text', ModelState.goalText(task));
    original.append(make('summary', '', '查看本批完整需求'), requirement); host.append(original);
    list = make('ol', 'conversation-list'); list.id = 'conversation-list'; list.setAttribute('aria-label', '项目对话记录'); host.append(list);
    const form = make('form', 'followup-form'); form.id = 'followup-form';
    const context = make('div', 'composer-context'), label = make('label', '', '发送方式'); label.htmlFor = 'followup-kind';
    kind = make('select'); kind.id = 'followup-kind';
    for (const [value, title] of [['question', '提问（只读）'], ['requirement', '安排修改／下一批']]) { const option = make('option', '', title); option.value = value; kind.append(option); }
    context.append(label, kind);
    targetGroup = make('div', 'field followup-target'); targetGroup.hidden = true;
    const targetLabel = make('label', '', '要修改的项目文件夹'); targetLabel.htmlFor = 'followup-project';
    targetPath = make('input'); targetPath.id = 'followup-project'; targetPath.placeholder = '选择具体项目文件夹，留空使用当前目录'; targetPath.setAttribute('aria-describedby', 'followup-project-hint');
    targetBrowse = make('button', '', '选择文件夹'); targetBrowse.type = 'button';
    const targetHint = make('p', 'hint', '如果刚才查看的是总目录，请选择其中要修改的具体项目；会保留本次调查和对话。'); targetHint.id = 'followup-project-hint';
    targetGroup.append(targetLabel, targetPath, targetBrowse, targetHint);
    targetBrowse.onclick = async () => { targetBrowse.disabled = true; try { const result = await api('pick-project', {}); if (result.path) { targetPath.value = result.path; saveDraft(); } } catch (err) { error.textContent = friendly(err.message); error.hidden = false; } finally { targetBrowse.disabled = false; } };
    targetPath.oninput = saveDraft;
    const inputLabel = make('label', 'sr-only', '你的问题或要求'); inputLabel.htmlFor = 'followup-content';
    content = make('textarea'); content.id = 'followup-content'; content.maxLength = 12000; content.rows = 2; content.placeholder = '继续提问，或切换为修改要求…';
    hint = make('p', 'hint'); hint.id = 'followup-hint'; content.setAttribute('aria-describedby', hint.id);
    error = make('p', 'notice error'); error.id = 'followup-error'; error.hidden = true; error.setAttribute('role', 'alert');
    const footer = make('div', 'followup-footer'), help = make('details', 'composer-help');
    help.append(make('summary', '', '发送说明'), hint); footer.append(help, make('span', 'composer-shortcut', 'Ctrl + Enter 发送'));
    send = make('button', 'primary', '发送'); send.id = 'followup-send'; send.type = 'submit'; footer.append(send);
    const attachments = make('div'); attachments.id = 'followup-attachments';
    const assistants = make('div', 'workspace-assistant-selection'); assistants.id = 'workspace-assistant-selection';
    form.append(context, targetGroup, inputLabel, content, assistants, attachments, error, footer);
    const dock = document.getElementById('workspace-composer'); if (dock) dock.replaceChildren(form); else host.append(form);
    const current = draft(task.id); content.value = current.content; kind.value = current.kind; targetPath.value = current.project || '';
    const id = task.id; AttachmentUI.mount('task:' + id, attachments, { id: 'followup-attachments', pasteTarget: content, dropTarget: form, onChange: () => { if (task?.id === id) saveDraft(); } });
    content.oninput = saveDraft; kind.onchange = saveDraft;
    content.onkeydown = event => { if ((event.ctrlKey || event.metaKey) && event.key === 'Enter' && !event.isComposing && event.keyCode !== 229) { event.preventDefault(); void submit(); } };
    form.onsubmit = event => { event.preventDefault(); void submit(); };
  }
  function updateControls() {
    if (!task || !send) return;
    const busy = pending.has(task.id), readonly = kind.value === 'question', renewal = retainedRenewalEligible(task);
    targetGroup.hidden = !(task.read_only && task.state === 'inspected' && !readonly); targetPath.disabled = targetBrowse.disabled = busy || !!task.archived_at;
    const unavailable = !readonly && (!!task.successor_task_id || !!task.seed_pending || !!task.recovery_pending || !!task.local_checkpoint && task.state !== 'accepted' || !renewal && (['cancelled','cancelling','applying','exhausted'].includes(task.state) || (!task.read_only && task.state !== 'accepted' && task.round >= 3) || (task.read_only && task.state !== 'inspected')));
    const unsavedAgent = typeof AssistantUI !== 'undefined' && AssistantUI.hasUnsaved(task.id);
    send.disabled = busy || !!task.archived_at || unavailable || unsavedAgent || AttachmentUI.blocked('task:' + task.id) || !content.value.trim() && !AttachmentUI.ids('task:' + task.id).length && !(renewal && !readonly);
    AttachmentUI.disabled('task:' + task.id, busy || !!task.archived_at);
    const begin = document.getElementById('begin-next-batch'); if (begin) begin.disabled = actionBusy || busy || unsavedAgent || !readonly && AttachmentUI.blocked('task:' + task.id);
    kind.disabled = busy || !!task.archived_at; content.disabled = busy || !!task.archived_at;
    for (const retry of host.querySelectorAll('[data-message-retry]')) retry.disabled = busy || !!task.archived_at;
    send.textContent = busy ? '正在发送…' : readonly ? '发送问题' : renewal ? '保留修改，开始下一批' : task.read_only || task.state === 'accepted' ? '开始下一批' : '发送补充要求';
    const recoveryHint = ['interrupted','blocked','cancelled','cancelling'].includes(task.state) ? '恢复检查尚未完成，请先通过任务操作重新检查保留的副本。当前草稿会保留。' : '正在检查保留的副本。检查通过后才可补充修改要求，当前草稿会保留。';
    hint.textContent = task.archived_at ? '先恢复这条任务，再继续对话。' : readonly ? '只读回答，使用“调查与规划”的工具和模型。运行中会排队，不消耗本批的自动返工轮次。' : task.successor_task_id ? '本批已经开启下一批。请点击“打开下一批”继续提出修改，避免重复开启；当前草稿保留。' : task.seed_pending ? '正在继承上一批修改并检查证据。检查通过后才可补充要求，当前草稿会保留。' : task.recovery_pending ? recoveryHint : renewal ? '已有修改和未解决问题会一起带入独立的新批次，每批最多自动尝试三轮。可补充要求，也可留空继续原目标；仍需实际检查、独立审查和你验收。' : unavailable ? '本批暂不能接收修改要求。已有副本和记录保留，请查看上方原因与可用操作；也可继续提问。' : task.read_only || task.state === 'accepted' ? '会开启下一批修改，继承已有背景；切换文件夹后会重新调查和检查。'  : '新要求从下一次调查与规划生效。本批最多自动尝试三轮；旧结果未覆盖新要求时不能验收。';
    content.placeholder = renewal && !readonly ? '可留空继续原目标和未解决问题；也可以补充下一批要做的修改。' : '例如：刚才说的待完成项有哪些？或：继续增加导出功能。';
    if (unsavedAgent) hint.textContent = 'Agent 改选尚未保存。请先保存切换，再发送这条消息；文字和附件都保留。';
    host.setAttribute('aria-busy', String(busy));
  }
  function renderMessages() {
    const history = historyPages.get(task.id);
    const key = JSON.stringify([task.messages || [], task.parent_task_id, task.successor_task_id, task.messages_before, history]); if (key === signature) return; signature = key;
    const expanded = new Set([...list.querySelectorAll('details[open]')].map(node => node.dataset.message));
    list.replaceChildren();
    const before = history ? history.before : task.messages_before || 0;
    if (before > 0) {
      const item = make('li', 'conversation-history'), load = make('button', '', history?.busy ? '正在读取…' : '读取更早对话（剩余 ' + before + ' 条）'); load.type = 'button'; load.disabled = !!history?.busy;
      load.onclick = () => loadHistory(before); item.append(load); list.append(item);
    }
    if (history?.updated && history.updated !== task.updated) { const item = make('li', 'hint'), reload = make('button', '', '重新读取较早对话'); reload.type = 'button'; reload.onclick = () => { historyPages.delete(task.id); signature = ''; renderMessages(); }; item.append(document.createTextNode('较早记录可能有更新。'), reload); list.append(item); }
    if (task.parent_task_id) { const item = make('li', 'conversation-context'), link = make('a', '', '查看上一批的项目记录'); link.href = '#' + encodeURIComponent(task.parent_task_id); item.append(link); list.append(item); }
    if (task.successor_task_id) { const item = make('li', 'conversation-context'), link = make('a', '', '打开下一批'); link.href = '#' + encodeURIComponent(task.successor_task_id); item.append(link); list.append(item); }
    const messages = new Map(); for (const message of [...(history?.messages || []), ...(task.messages || [])]) messages.set(message.id, message);
    for (const message of messages.values()) {
      const row = make('li', 'conversation-message'); row.dataset.messageId = message.id;
      const meta = make('div', 'conversation-meta'); meta.append(make('strong', '', '你 · ' + (message.kind === 'question' ? '提问' : '修改要求')), make('span', '', delivery[message.status] || message.status));
      if (message.created) { const time = make('time', '', new Date(message.created).toLocaleString('zh-CN')); time.dateTime = message.created; meta.append(time); }
      row.append(meta, make('p', 'conversation-user', message.content)); AttachmentUI.historical(row, message.attachments);
      if (message.applied_round) row.append(make('p', 'hint', '纳入本批第 ' + message.applied_round + ' 轮计划，最终结果以检查和独立审查为准。'));
      else if (message.kind === 'requirement' && message.status === 'queued') row.append(make('p', 'delivery-receipt', '要求已接收' + (message.revision ? ' · 需求版本 ' + message.revision : '') + '，将在下一次调查与规划中核对。'));
      if (message.reply) {
        const reply = make('div', 'conversation-reply'); reply.append(make('strong', '', (message.provider === 'deepseek' ? 'DeepSeek Harness' : 'Codex') + (message.model ? ' · ' + message.model : '')), make('p', '', message.reply.summary));
        const details = make('details'); details.dataset.message = message.id; details.open = expanded.has(message.id); details.append(make('summary', '', '查看依据、待确认项与建议'));
        for (const [field, title] of [['facts','事实'],['inferences','推断'],['unknowns','待确认'],['plan','后续建议'],['read_files','读取的文件']]) {
          if (!message.reply[field]?.length) continue; details.append(make('h3', '', title)); const items = make('ul'); for (const text of message.reply[field]) items.append(make('li', '', text)); details.append(items);
        }
        reply.append(details, make('p', 'hint', '基于当前文件的只读回答，未运行测试，也不代表修改已经通过验收。')); row.append(reply);
      }
      if (message.error) {
        row.append(make('p', 'conversation-error', friendly(message.error)));
        if (['failed','interrupted','cancelled'].includes(message.status) && !task.archived_at) {
          const retry = make('button', '', '重新提问'), owner = task.id; retry.type = 'button'; retry.dataset.messageRetry = '';
          retry.onclick = () => { if (task?.id !== owner || pending.has(owner) || task.archived_at) return; content.value = message.content; kind.value = 'question'; AttachmentUI.set('task:' + owner, message.attachments); draft(owner).nonce = crypto.randomUUID(); saveDraft(); content.focus(); }; row.append(retry);
        }
      }
      if (message.linked_task) { const link = make('a', '', '查看这一批的协作进度'); link.href = '#' + encodeURIComponent(message.linked_task); row.append(link); }
      list.append(row);
    }
    if (!task.messages?.length) list.append(make('li', 'conversation-empty', '在下方继续提问或补充修改要求。执行过程、调查报告和成果可从上方标签查看。'));
    if (task.messages?.some(message => message.status === 'reading')) { const stop = make('button', '', '停止回答'); stop.type = 'button'; stop.onclick = () => perform(task.id, 'cancel'); list.append(stop); }
  }
  async function loadHistory(before) {
    const owner = task.id, generation = conversationEpoch, saved = historyPages.get(owner) || { messages: [], latest: task.messages || [], latestTotal: task.messages_total || task.messages?.length || 0, before, updated: task.updated };
    if (saved.busy) return; saved.busy = true; historyPages.set(owner, saved); signature = ''; renderMessages();
    try {
      const page = await api('tasks/' + owner + '/messages?before=' + before + '&limit=25');
      if (generation !== conversationEpoch || task?.id !== owner) return;
      const indexed = page.messages.map((message, index) => ({ ...message, _position: page.before + index }));
      saved.messages = WorkbenchState.mergeMessages(indexed, saved.messages, saved.latest, task.messages); saved.before = WorkbenchState.earlierCursor(saved.messages, page.before); saved.updated = page.updated;
    } catch (problem) { if (task?.id === owner && generation === conversationEpoch) { error.textContent = friendly(problem.message); error.hidden = false; } }
    finally { saved.busy = false; if (task?.id === owner && generation === conversationEpoch) { signature = ''; renderMessages(); } }
  }
  async function submit() {
    if (!task || send.disabled || pending.has(task.id)) return;
    saveDraft(); const current = { ...draft(task.id) }; WorkspaceUI.show('conversation');
    await deliver(current, current.kind === 'requirement' && retainedRenewalEligible(task), true);
  }
  function defaultBatchNonce(id) {
    if (!batchNonces.has(id)) {
      let saved; try { saved = sessionStorage.getItem('xiaobuding.nextbatch.' + id); } catch {}
      const nonce = saved || crypto.randomUUID(); batchNonces.set(id, nonce);
      try { sessionStorage.setItem('xiaobuding.nextbatch.' + id, nonce); } catch {}
    }
    return batchNonces.get(id);
  }
  async function startNextBatch(id) {
    if (task?.id !== id || !retainedRenewalEligible(task) || pending.has(id)) return;
    const current = draft(id), useDraft = current.kind === 'requirement';
    if (typeof AssistantUI !== 'undefined' && AssistantUI.hasUnsaved(id) || useDraft && AttachmentUI.blocked('task:' + id)) return;
    await deliver(useDraft ? { ...current } : { content: '', kind: 'requirement', nonce: defaultBatchNonce(id), attachment_ids: [] }, true, useDraft);
  }
  async function deliver(current, renewal, clearDraft) {
    const id = task.id; pending.add(id); if (renewal) nextBatches.add(id); updateControls(); error.hidden = true;
    detailSignature = ''; render();
    try {
      const message = await api('tasks/' + id + (renewal ? '/exhausted-next-batch' : '/messages'), { id: current.nonce, kind: current.kind, content: current.content, attachment_ids: current.attachment_ids || [], ...(task.read_only && current.kind === 'requirement' && current.project?.trim() ? { project: current.project.trim() } : {}) });
      const sameDraft = clearDraft && drafts.get(id)?.nonce === current.nonce;
      if (sameDraft) {
        drafts.set(id, { content: '', kind: current.kind, nonce: crypto.randomUUID() });
        try { sessionStorage.removeItem('xiaobuding.followup.' + id); } catch {}
        AttachmentUI.clear('task:' + id, false);
      }
      if (task?.id === id && selected === id) { if (sameDraft) content.value = ''; if (message.linked_task) selectTask(message.linked_task); }
      await refresh();
    } catch (err) { if (task?.id === id && selected === id) { error.textContent = friendly(err.message); error.hidden = false; } }
    finally { pending.delete(id); nextBatches.delete(id); detailSignature = ''; render(); }
  }
  function sync(current) {
    if (current) current = { ...current, messages: (current.messages || []).map((message, index) => ({ ...message, _position: (current.messages_before || 0) + index })) };
    const history = current && historyPages.get(current.id);
    if (history) { if ((current.messages_before || 0) > (history.latestTotal || 0)) history.before = Math.max(history.before, current.messages_before); history.messages = WorkbenchState.mergeMessages(history.messages, history.latest, current.messages); history.before = WorkbenchState.earlierCursor(history.messages, history.before); history.latest = current.messages || []; history.latestTotal = current.messages_total || current.messages.length; }
    if (current?.id !== task?.id) { conversationEpoch++; task = current || null; signature = ''; if (!task) return; build(); }
    else task = current || null;
    if (!task) return; requirement.textContent = ModelState.goalText(task); renderMessages(); updateControls();
  }
  return { sync, startNextBatch, isPending: id => pending.has(id), isNextBatchPending: id => nextBatches.has(id), isNextBatchBlocked: id => (typeof AssistantUI !== 'undefined' && AssistantUI.hasUnsaved(id)) || draft(id).kind === 'requirement' && AttachmentUI.blocked('task:' + id) };
})();
