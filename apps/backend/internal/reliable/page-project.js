'use strict';
const ProjectUI = (() => {
  const records = new Map(), drafts = new Map();
  const labels = { planned: '待开始', in_progress: '进行中', verified: '本批已核验', accepted: '已验收', blocked: '待处理', deferred: '已延期', cancelled: '已取消' };
  const memoryLabels = { decision: '设计决定', constraint: '项目约束', open_issue: '待解决' };
  let task = null, epoch = 0, signature = '', busy = false, loadRequest = 0;
  const patchLimit = 512 * 1024;
  function counts(record) {
    const items = record.items || [];
    return { total: items.length, verified: items.filter(item => ['verified', 'accepted'].includes(item.status)).length, accepted: items.filter(item => item.status === 'accepted').length };
  }
  function suggested(item) { return item.source === 'plan' || String(item.source || '').startsWith('plan:'); }
  function itemTitle(item) {
    const text = String(item.short_title || '').trim() || String(item.title || '').trim() || '未命名待办', chars = Array.from(text);
    return chars.length > 52 ? chars.slice(0, 52).join('') + '…' : text;
  }
  function savedItems(record) { return (record.items || []).filter(item => item.source !== 'task' && !String(item.id || '').startsWith('batch-')); }
  function itemEditable(item, current) {
    return !!item.id && !current.archived_at && !item.task_id && item.source !== 'task' && !String(item.id).startsWith('batch-') && ['planned', 'deferred', 'cancelled'].includes(item.status);
  }
  function movedItemOrder(record, id, direction) {
    const order = savedItems(record).map(item => item.id), index = order.indexOf(id), target = index + direction;
    if (index < 0 || target < 0 || target >= order.length || ![-1, 1].includes(direction)) return null;
    [order[index], order[target]] = [order[target], order[index]]; return order;
  }
  function reconcileItemDraft(existing, item, revision) {
    if (existing?.dirty) return existing;
    return { id: item.id, title: item.title || '', short_title: item.short_title || '', acceptance: item.acceptance || '', milestone: item.milestone || '', depends_on: [...(item.depends_on || [])], disposition: item.status, reason: item.status_reason || '', base_revision: revision, dirty: false };
  }
  function itemEditPayload(value) {
    return { expected_revision: value.base_revision, item_edit: { id: value.id, title: value.title.trim(), short_title: value.short_title.trim(), acceptance: value.acceptance.trim(), milestone: value.milestone.trim(), depends_on: [...value.depends_on], disposition: value.disposition, reason: value.reason.trim() } };
  }
  function memorySource(memory) { return memory.source === 'user' ? '你补充的记录' : String(memory.source || '').startsWith('question:') ? '原生 Agent 只读调查' : '来源：' + (memory.source || '未记录'); }
  function safeEvidencePath(path) {
    return typeof path === 'string' && !!path && !/[\\:\x00-\x1f]/.test(path) && !path.startsWith('/') && path.split('/').every(part => !!part && part !== '.' && part !== '..');
  }
  function evidence(parent, entries, fallback) {
    let linked = 0;
    for (const entry of entries || []) {
      const path = typeof entry === 'string' ? entry : entry.path, id = entry.task_id || fallback;
      if (!id || !safeEvidencePath(path)) continue;
      const link = el('a', 'project-evidence', entry.label || path); link.href = evidenceURL(id, path); link.target = '_blank'; link.rel = 'noopener'; parent.append(link); linked++;
    }
    if (!linked) parent.append(el('span', 'hint', '尚无逐项验收证据，不能据此认定已完成。'));
  }
  function taskLink(id, title) { const link = el('a', '', title); link.href = '#' + encodeURIComponent(id); return link; }
  function status(message, error = false) {
    const node = $('project-status'); if (!node) return;
    node.textContent = message || ''; node.dataset.error = String(error);
  }
  function draft() {
    if (!drafts.has(task.id)) drafts.set(task.id, { title: '', acceptance: '', kind: 'decision', text: '', itemEdits: {} });
    return drafts.get(task.id);
  }
  function field(form, name, label, value, multiline = false) {
    const control = el(multiline ? 'textarea' : 'input'); control.id = 'project-' + name; control.name = name; control.value = value; control.maxLength = name === 'title' ? 1000 : 4000;
    const caption = el('label', '', label); caption.htmlFor = control.id; control.oninput = () => { draft()[name] = control.value; }; form.append(caption, control); return control;
  }
  async function mutate(body, success, clear = [], itemID = '') {
    if (busy || !task) return; const id = task.id, record = records.get(id), attempt = epoch;
    if (typeof AssistantUI !== 'undefined' && !AssistantUI.allowDispatch(id, 'project-save', body)) return;
    busy = true; loadRequest++; paint(record); status('正在保存…');
    try {
      const result = await api('tasks/' + encodeURIComponent(id) + '/project', { expected_revision: record.revision, ...body });
      records.set(id, result); for (const key of clear) drafts.get(id)[key] = '';
      if (itemID) delete drafts.get(id).itemEdits[itemID];
      if (attempt === epoch) { paint(result); status(success); }
    } catch (err) { if (attempt === epoch) status(friendly(err.message) + '。输入已保留，请刷新项目记录后重试。', true); }
    finally { finish(attempt); }
  }
  function enable() {
    const current = $('project-status'), message = current?.textContent, error = current?.dataset.error === 'true';
    paint(records.get(task?.id)); status(message, error);
  }
  function finish(attempt) {
    busy = false;
    if (attempt === epoch) enable(); else if (task && WorkspaceUI.current() === 'project') load(true);
  }
  async function dispatch(action, body = {}) {
    if (task && typeof AssistantUI !== 'undefined' && !AssistantUI.allowDispatch(task.id, action, body)) return;
    if (busy || !task) return; const id = task.id, attempt = epoch; busy = true; loadRequest++; paint(records.get(id)); status(action === 'plan' ? '正在提交只读规划请求…' : '正在安排下一项…');
    try {
      const message = await api('tasks/' + encodeURIComponent(id) + '/project/' + action, body);
      if (attempt !== epoch) return;
      if (message.linked_task) { location.hash = message.linked_task; await refresh(); }
      else { status('规划请求已加入项目对话。完成后请刷新项目记录，建议会标注为待确认。'); WorkspaceUI.show('conversation'); await refresh(); }
    } catch (err) { if (attempt === epoch) status(friendly(err.message), true); }
    finally { finish(attempt); }
  }
  function itemList(parent, record) {
    const list = el('ol', 'project-items');
    for (const item of record.items || []) {
      const row = el('li', 'project-item'), heading = el('div', 'project-item-heading');
      heading.append(el('strong', '', itemTitle(item)), el('span', 'badge', labels[item.status] || '尚未核验')); row.append(heading);
      const details = el('details', 'project-item-requirement'); details.append(el('summary', '', '完整要求与验收条件'), el('p', '', item.title));
      if (item.acceptance) details.append(el('p', '', '验收标准：' + item.acceptance)); row.append(details);
      if (item.milestone) row.append(el('p', 'hint', '里程碑：' + item.milestone));
      if (item.status_reason) row.append(el('p', 'hint', '原因：' + item.status_reason));
      if (item.depends_on?.length) row.append(el('p', 'hint', '前置待办：' + item.depends_on.map(id => itemTitle((record.items || []).find(other => other.id === id) || { title: id })).join('、')));
      if (suggested(item) && !item.confirmed && !item.task_id) row.append(el('p', 'hint', '模型规划建议，需你确认后才安排实施。'));
      if (item.task_id || item.evidence?.length) {
        const links = el('div', 'project-item-links'); evidence(links, item.evidence, item.task_id);
        if (item.task_id) links.append(taskLink(item.task_id, '查看对应批次')); row.append(links);
      }
      if (['accepted', 'inspected'].includes(task.state) && !task.successor_task_id && !task.archived_at && item.status === 'planned' && !item.task_id) {
        const pending = (item.depends_on || []).some(id => (record.items || []).find(other => other.id === id)?.status !== 'accepted');
        const start = button(pending ? '等待前置待办验收' : suggested(item) && !item.confirmed ? '确认并开始这项' : '开始这项', '', () => dispatch('start', { item_id: item.id })); start.disabled = busy || pending; row.append(start);
      }
      if (itemEditable(item, task)) itemEditor(row, record, item);
      list.append(row);
    }
    if (!list.children.length) parent.append(el('p', 'muted', '还没有项目待办。可以添加下一步目标，或让原生规划 Agent 只读整理建议。'));
    parent.append(list);
  }
  function itemEditor(parent, record, item) {
    const details = el('details', 'project-item-edit'), owner = task.id;
    details.dataset.projectKey = 'edit-' + item.id; const caption = el('summary', '', '编辑这项待办'); caption.setAttribute('aria-label', '编辑待办：' + itemTitle(item)); details.append(caption);
    details.ontoggle = () => {
      if (!details.open || details.dataset.loaded || !details.isConnected || task?.id !== owner) return;
      details.dataset.loaded = 'true'; buildItemEditor(details, record, item);
    };
    parent.append(details);
    const order = el('div', 'project-item-order');
    for (const [direction, title] of [[-1, '上移'], [1, '下移']]) {
      const next = movedItemOrder(record, item.id, direction), move = button(title, '', () => mutate({ item_order: next }, '待办顺序已更新。'));
      move.setAttribute('aria-label', title + '待办：' + itemTitle(item));
      move.disabled = !next || busy; order.append(move);
    }
    parent.append(order);
  }
  function buildItemEditor(details, record, item) {
    const cache = draft().itemEdits, value = reconcileItemDraft(cache[item.id], item, record.revision); cache[item.id] = value;
    const form = el('form', 'project-form');
    const input = (name, title, multiline = false) => {
      const control = el(multiline ? 'textarea' : 'input'), label = el('label', '', title); control.id = 'project-edit-' + item.id + '-' + name; label.htmlFor = control.id; control.value = value[name]; control.maxLength = name === 'title' ? 1000 : name === 'acceptance' ? 1300 : name === 'reason' ? 320 : 80;
      control.oninput = () => { value[name] = control.value; value.dirty = true; }; form.append(label, control); return control;
    };
    input('short_title', '短标题（可选，完整要求仍保留）'); input('title', '完整要求', true).required = true;
    input('acceptance', '验收条件', true); input('milestone', '所属里程碑（可选）');
    itemDependencies(form, record, item, value); itemDisposition(form, item, value, input);
    const save = button('保存待办', '', null); save.type = 'submit'; form.append(save);
    form.onsubmit = event => { event.preventDefault(); if (!form.reportValidity()) return; mutate(itemEditPayload(value), '待办已保存。完整要求和验收历史均保留。', [], item.id); };
    if (value.base_revision !== record.revision && value.dirty) {
      form.append(el('p', 'hint', '项目记录已更新，你的输入已保留。核对当前记录后再按最新版本保存。'));
      form.append(button('核对后按最新版本保存', '', () => { value.base_revision = record.revision; mutate(itemEditPayload(value), '已按你核对的最新记录保存待办。', [], item.id); }));
    }
    form.append(button('放弃此项输入，载入当前记录', '', () => { delete cache[item.id]; paint(record); }));
    details.append(form);
    if (busy) for (const control of form.querySelectorAll('button,input,select,textarea')) control.disabled = true;
  }
  function itemDependencies(parent, record, item, value) {
    const details = el('details', 'project-item-dependencies'); details.append(el('summary', '', '前置待办（' + value.depends_on.length + ' 项）'));
    details.append(el('p', 'hint', '勾选需先验收的待办。程序会阻止循环依赖；检查通过仍需你验收后才能继续。'));
    for (const other of savedItems(record).filter(other => other.id !== item.id)) {
      const label = el('label', 'project-dependency-choice'), check = el('input'); check.type = 'checkbox'; check.checked = value.depends_on.includes(other.id);
      check.onchange = () => { value.depends_on = check.checked ? [...value.depends_on, other.id] : value.depends_on.filter(id => id !== other.id); value.dirty = true; details.querySelector('summary').textContent = '前置待办（' + value.depends_on.length + ' 项）'; };
      label.append(check, el('span', '', itemTitle(other) + ' · ' + (labels[other.status] || '尚未核验'))); details.append(label);
    }
    parent.append(details);
  }
  function itemDisposition(parent, item, value, input) {
    const label = el('label', '', '待办安排'), select = el('select'); select.id = 'project-disposition-' + item.id; label.htmlFor = select.id;
    for (const state of ['planned', 'deferred', 'cancelled']) { const option = el('option', '', labels[state]); option.value = state; select.append(option); }
    select.value = value.disposition; parent.append(label, select);
    const reason = input('reason', '延期或取消原因'); reason.required = value.disposition !== 'planned'; reason.disabled = value.disposition === 'planned';
    select.onchange = () => { value.disposition = select.value; value.dirty = true; reason.required = value.disposition !== 'planned'; reason.disabled = value.disposition === 'planned'; if (!reason.required) { value.reason = ''; reason.value = ''; } };
  }
  function requirementForm(parent) {
    const details = el('details', 'project-disclosure'), form = el('form', 'project-form'); details.append(el('summary', '', '添加项目待办'));
    const title = field(form, 'title', '下一步要完成什么', draft().title); title.required = true;
    field(form, 'acceptance', '怎样才算完成（可选）', draft().acceptance, true);
    const submit = button('保存待办', '', null); submit.type = 'submit'; form.append(submit);
    form.onsubmit = event => { event.preventDefault(); if (!title.value.trim()) return; mutate({ items: [{ title: title.value.trim(), acceptance: draft().acceptance.trim() }] }, '待办已保存。它会经过实际检查和独立审查，再由你验收。', ['title', 'acceptance']); };
    details.append(form); parent.append(details);
  }
  function memories(parent, record) {
    const details = el('details', 'project-disclosure'); details.append(el('summary', '', '项目记忆与待解决事项（' + (record.memories || []).length + '）'));
    details.append(el('p', 'hint', '保存需求、约束与决定，关联原始批次。人工记录和模型推断会保留来源，不能代替实际核验。'));
    const list = el('ul', 'project-memories');
    for (const memory of record.memories || []) {
      const item = el('li'); item.append(el('strong', '', memory.kind === 'observation' ? '只读调查记录' : memoryLabels[memory.kind] || memory.kind || '项目记录'), el('p', '', memory.text));
      const meta = el('div', 'project-item-links'); meta.append(el('small', 'hint', memorySource(memory) + (memory.verified ? ' · 有核验记录' : ' · 尚未独立核验')));
      if (memory.source_task) meta.append(taskLink(memory.source_task, '查看来源批次'));
      if (memory.commit) meta.append(el('code', '', 'Git ' + memory.commit.slice(0, 12)));
      if (memory.evidence?.length) evidence(meta, memory.evidence, memory.source_task); item.append(meta); list.append(item);
    }
    if (!list.children.length) details.append(el('p', 'muted', '尚未记录长期决定或约束。'));
    details.append(list); const form = el('form', 'project-form'), label = el('label', '', '记录类型'), kind = el('select'); kind.id = 'project-memory-kind'; label.htmlFor = kind.id;
    for (const [value, text] of Object.entries(memoryLabels)) { const option = el('option', '', text); option.value = value; kind.append(option); }
    kind.value = draft().kind; kind.onchange = () => { draft().kind = kind.value; }; form.append(label, kind);
    const text = field(form, 'text', '补充项目记忆', draft().text, true); text.required = true;
    const save = button('保存这条记忆', '', null); save.type = 'submit'; form.append(save);
    form.onsubmit = event => { event.preventDefault(); if (text.value.trim()) mutate({ memory: { kind: kind.value, text: text.value.trim() } }, '记忆已保存，后续 Agent 可读取这条记录及其来源。', ['text']); };
    details.append(form); parent.append(details);
  }
  function batches(parent, record) {
    const details = el('details', 'project-disclosure'); details.append(el('summary', '', '同一项目的批次记录（' + (record.batches || []).length + '）'));
    const list = el('ol', 'project-batches');
    for (const batch of record.batches || []) {
      const item = el('li'), link = taskLink(batch.id, itemTitle({ title: batch.goal || batch.id })); link.title = batch.goal || batch.id;
      item.append(link, el('small', 'hint', (states[batch.state] || batch.state) + ' · ' + (batch.round || 0) + '/3')); list.append(item);
    }
    details.append(list); parent.append(details);
  }
  function handoffCard(parent, record) {
    const handoff = record.handoff; if (!handoff) return;
    const box = el('section', 'project-handoff'), head = el('div', 'project-item-heading');
    head.append(el('h3', '', '当前批次交接卡'), el('span', 'badge', handoff.verified_batch ? '本批有核验记录' : '本批仍需核验')); box.append(head);
    box.append(el('p', '', handoff.goal_excerpt || '当前尚未记录目标。'));
    const meta = el('p', 'hint', '项目记录版本 ' + handoff.project_revision + ' · 需求版本 ' + handoff.requirement_revision + ' · 已调查版本 ' + handoff.audited_revision);
    box.append(meta);
    if (handoff.source_snapshot) { const version = el('code', '', '源码指纹 ' + handoff.source_snapshot.slice(0, 12)); version.title = handoff.source_snapshot; box.append(version); }
    const warnings = el('ul', 'project-handoff-warnings'); for (const warning of handoff.warnings || []) warnings.append(el('li', '', warning)); box.append(warnings);
    for (const [key, title] of [['open_issues', '未解决与待确认'], ['facts', '调查报告引用的事实'], ['inferences', '模型推断']]) {
      const notes = handoff[key] || []; if (!notes.length) continue;
      const details = el('details', 'project-handoff-notes'), list = el('ul'); details.dataset.projectKey = 'handoff-' + key; details.append(el('summary', '', title + '（' + notes.length + '）'));
      for (const note of notes) {
        const item = el('li'), source = el('div', 'project-item-links'); item.append(el('p', '', note.text));
        const sourceName = ({ audit: '原生 Agent 调查', review: '独立审查报告', controller: '实际执行记录' })[note.source] || note.source || '历史记录';
        source.append(el('small', 'hint', note.source === 'user' ? '你记录的内容' : '来源：' + sourceName));
        source.append(el('small', 'hint', note.verified ? '有关联版本的核验记录' : '尚未作为独立核验事实确认'));
        if (note.source_task) source.append(taskLink(note.source_task, '查看来源批次'));
        if (note.commit) source.append(el('code', '', note.commit.slice(0, 12))); item.append(source); list.append(item);
      }
      details.append(list); box.append(details);
    }
    if (handoff.evidence?.length) { const links = el('div', 'project-item-links'); evidence(links, handoff.evidence, handoff.task_id); box.append(links); }
    if (handoff.truncated) box.append(el('p', 'hint', '交接卡为节选，完整记录仍保留在来源批次。'));
    parent.append(box);
  }
  function recovery(parent) {
    if (!task.stage_recovery) return;
    const saved = task.stage_recovery, panel = el('div', 'project-recovery');
    panel.append(el('strong', '', '阶段恢复记录'), el('p', '', '当前阶段：' + (saved.domain ? roleTitle(saved.domain) : '等待确认') + '；累计执行 ' + (saved.attempts || 0) + ' 次。每次继续最多自动尝试 3 次，故障恢复与代码返工轮次分别记录。'));
    if (saved.manual_resumes) panel.append(el('p', '', '已由你继续恢复 ' + saved.manual_resumes + ' 次，之前完成的阶段会先核对源码与交接证据。'));
    if (saved.last_error) panel.append(el('p', 'hint', '最近中断原因：' + friendly(saved.last_error)));
    parent.append(panel);
  }
  async function readPatch(response) {
    if (!response.ok) throw Error('本批尚无可读取的代码补丁。');
    if (Number(response.headers.get('Content-Length')) > patchLimit) { await response.body?.cancel(); throw Error('补丁较大，请打开原始补丁文件查看。'); }
    const reader = response.body.getReader(), chunks = []; let size = 0;
    try {
      while (true) { const part = await reader.read(); if (part.done) break; size += part.value.length; if (size > patchLimit) { await reader.cancel(); throw Error('补丁较大，请打开原始补丁文件查看。'); } chunks.push(part.value); }
    } finally { reader.releaseLock(); }
    const joined = new Uint8Array(size); let offset = 0; for (const chunk of chunks) { joined.set(chunk, offset); offset += chunk.length; }
    return new TextDecoder().decode(joined);
  }
  function patchLines(text) { const lines = text.split('\n'); return { lines: lines.slice(0, 3000), limited: lines.length > 3000 }; }
  function changes(parent) {
    const details = el('details', 'project-disclosure project-diff'), body = el('div'); details.append(el('summary', '', '查看本批代码变化'), body);
    body.append(el('p', 'hint', '按需读取本批补丁，最多显示 512 KB。绿色为新增，红色为删除。'));
    if (task.worktree) body.append(button('打开修改后的工作副本', '', () => perform(task.id, 'open')));
    const current = task;
    if (!current.round) { body.append(el('p', 'muted', '本批还没有代码修改记录。')); parent.append(details); return; }
    const url = evidenceURL(current.id, 'round-' + current.round + '/changes.patch'), raw = el('a', 'project-evidence', '打开原始补丁文件'); raw.href = url; raw.target = '_blank'; raw.rel = 'noopener'; body.append(raw);
    details.ontoggle = async () => {
      if (!details.open || details.dataset.loaded) return; details.dataset.loaded = 'true';
      const result = el('pre', 'project-patch', '正在读取本批补丁…'); body.append(result);
      try {
        const patch = await readPatch(await fetch(url)), displayed = patchLines(patch); result.replaceChildren();
        for (const line of displayed.lines) result.append(el('span', line.startsWith('+') && !line.startsWith('+++') ? 'patch-added' : line.startsWith('-') && !line.startsWith('---') ? 'patch-removed' : '', line + '\n'));
        if (displayed.limited) body.append(el('p', 'hint', '页面最多展示 3000 行，请打开原始补丁文件查看完整内容。'));
        if (!patch) result.textContent = '本批补丁为空。';
      } catch (err) { result.textContent = err.message; }
    };
    parent.append(details);
  }
  function storage(parent) {
    const details = el('details', 'project-disclosure'); details.append(el('summary', '', '空间与临时文件')); details.append(el('p', 'hint', '项目记忆保存在现有数据库。这里只检查允许清理的临时文件，不删除任务副本、验收成果或审查证据。'));
    const info = el('div', 'project-storage-info');
    const inspect = button('查看空间与可清理项', '', async () => {
      inspect.disabled = true; info.replaceChildren(el('p', 'hint', '正在检查允许清理的临时文件…'));
      try { const report = await api('storage'); renderStorage(info, report); }
      catch (err) { info.replaceChildren(el('p', 'hint', friendly(err.message))); }
      finally { inspect.disabled = false; }
    });
    details.append(inspect, info); StorageBudgetUI.mount(details); parent.append(details);
  }
  function bytes(value) { const amount = Number(value || 0); return amount >= 1024 ** 3 ? (amount / 1024 ** 3).toFixed(2) + ' GB' : amount >= 1024 ** 2 ? (amount / 1024 ** 2).toFixed(1) + ' MB' : Math.ceil(amount / 1024) + ' KB'; }
  function renderStorage(parent, report) {
    parent.replaceChildren();
    if (report.root) parent.append(el('p', 'path', '数据位置：' + report.root));
    if (report.temp_root) parent.append(el('p', 'path', '临时文件位置：' + report.temp_root));
    parent.append(el('p', '', '协作数据分类占用（已扫描）：' + bytes(report.total_bytes)), el('p', 'hint', '统计范围：审查证据、任务工作副本、任务附件、临时文件和缓存。'));
    const names = { evidence: '审查证据', worktrees: '任务工作副本', attachments: '任务附件', temp: '临时文件', cache: '缓存' };
    const list = el('ul'); for (const entry of report.entries || []) list.append(el('li', '', (names[entry.name] || entry.name) + '：' + bytes(entry.bytes) + ' · ' + (entry.files || 0) + ' 个文件')); parent.append(list);
    if (report.partial) parent.append(el('p', 'hint', '部分目录未完成统计，当前数字不代表完整磁盘占用。'));
    const clean = button('检查可清理的临时文件', '', async () => {
      clean.disabled = true;
      try { const preview = await api('storage/cleanup', { dry_run: true }); renderCleanup(parent, preview); }
      catch (err) { parent.append(el('p', 'hint', friendly(err.message))); clean.disabled = false; }
    }); parent.append(clean);
  }
  function renderCleanup(parent, preview) {
    const box = el('div', 'project-cleanup'); box.append(el('p', '', '检查到 ' + (preview.files || 0) + ' 个超过 ' + (preview.minimum_age_days || 7) + ' 天的允许清理临时文件，预计释放 ' + bytes(preview.bytes) + '。'));
    if (preview.skipped_active) box.append(el('p', 'hint', '正在使用的临时目录已跳过，不会打断任务。'));
    if (preview.partial) box.append(el('p', 'hint', '此次为有界检查，部分条目已跳过。'));
    if (!preview.files) { parent.append(box); return; }
    const confirm = button('确认清理这些临时文件', '', async () => {
      confirm.disabled = true;
      try { const result = await api('storage/cleanup', { dry_run: false }); box.replaceChildren(el('p', '', '临时文件清理完成，删除 ' + (result.removed_files || 0) + ' 个临时文件，释放 ' + bytes(result.removed_bytes) + '。项目副本和证据保留。')); }
      catch (err) { box.append(el('p', 'hint', friendly(err.message))); confirm.disabled = false; }
    }); box.append(confirm); parent.append(box);
  }
  function paint(record) {
    if (!task || !record) return; const root = $('task-project'), open = [...root.querySelectorAll('details[open]')].map(item => item.dataset.projectKey || item.querySelector('summary')?.textContent);
    root.replaceChildren(); const head = el('div', 'project-heading'), stat = counts(record);
    head.append(el('h2', '', '项目总览'), el('span', 'hint', '核验 ' + stat.verified + '/' + stat.total + ' 项 · 已验收 ' + stat.accepted + ' 项'));
    const refreshButton = button('刷新项目记录', '', () => load(true)); refreshButton.disabled = busy; head.append(refreshButton); root.append(head, el('p', 'project-scope-note', '显示同一项目的长期待办、来源与批次记录。批次通过不代表每个项目要求都已完成。'));
    const feedback = el('p', 'project-feedback'); feedback.id = 'project-status'; feedback.setAttribute('role', 'status'); feedback.setAttribute('aria-live', 'polite'); root.append(feedback);
    recovery(root); const controls = el('div', 'project-controls'), plan = button('整理项目计划', '', () => dispatch('plan'));
    plan.disabled = busy || !!task.archived_at; controls.append(plan);
    const label = el('label', 'project-autocontinue'), check = el('input'); check.type = 'checkbox'; check.checked = !!record.auto_continue; check.disabled = busy || !!task.archived_at;
    check.onchange = () => { if (typeof AssistantUI !== 'undefined' && !AssistantUI.allowDispatch(task.id, 'project-save', { auto_continue: check.checked })) { check.checked = !!record.auto_continue; return; } void mutate({ auto_continue: check.checked }, check.checked ? '已开启：你验收后自动开始下一项已确认待办。' : '已关闭自动开始下一项。'); }; label.append(check, el('span', '', '验收后自动开始下一项待办')); controls.append(label); root.append(controls, el('p', 'hint', '整理计划使用当前原生规划 Agent，只读调查，不消耗三轮修改。模型建议需要你确认。'));
    if (task.state === 'accepted') root.append(el('p', 'hint', '本批已验收，开启自动衔接后即可开始下一项已确认待办。'));
    handoffCard(root, record); itemList(root, record); requirementForm(root); memories(root, record); batches(root, record); changes(root); storage(root);
    if (task.archived_at) { root.append(el('p', 'hint', '项目任务已归档，恢复到任务列表后可编辑项目待办和记忆。')); for (const control of root.querySelectorAll('.project-form button,.project-form input,.project-form textarea,.project-form select')) control.disabled = true; }
    for (const details of root.querySelectorAll('details')) details.open = open.includes(details.dataset.projectKey || details.querySelector('summary')?.textContent);
    if (busy) for (const control of root.querySelectorAll('button,input,select,textarea')) control.disabled = true;
  }
  async function load(force = false) {
    if (!task || busy) return; const id = task.id, attempt = epoch, request = ++loadRequest;
    if (!force && records.has(id)) { paint(records.get(id)); return; }
    const root = $('task-project'); root.setAttribute('aria-busy', 'true'); if (!records.has(id)) root.replaceChildren(el('p', 'hint', '正在读取项目记录…'));
    try { const record = await api('tasks/' + encodeURIComponent(id) + '/project'); if (attempt === epoch && request === loadRequest) { records.set(id, record); paint(record); } }
    catch (err) { if (attempt === epoch && request === loadRequest) { root.replaceChildren(el('p', 'hint', '项目记录暂未读取：' + friendly(err.message)), button('重试读取项目记录', '', () => load(true))); } }
    finally { if (attempt === epoch && request === loadRequest) root.setAttribute('aria-busy', 'false'); }
  }
  function sync(current) {
    const next = current ? current.id + ':' + current.updated : '';
    if (next === signature) return; signature = next;
    if (task?.id !== current?.id) epoch++;
    task = current;
    const editing = $('task-project')?.contains(document.activeElement) && document.activeElement.matches('input,textarea,select');
    if (task && WorkspaceUI.current() === 'project' && !busy && !editing) load(true);
  }
  return { sync, show: () => load(true), counts, suggested, safeEvidencePath, readPatch, patchLines, itemTitle, itemEditable, movedItemOrder, reconcileItemDraft, itemEditPayload };
})();
