'use strict';
const TaskActivity = (() => {
  let task = null, epoch = 0, cursor = 0, timer = null, controller = null, entries = [], limited = false;
  let following = true, filter = 'all', host, list, status, follow, last, connected = false, busy = false;
  let initialized = false, before = 0, hasEarlier = false, historical = false, earlier, latest;
  const working = ['queued','preparing','auditing','implementing','checking','reviewing','cancelling'];
  const labels = { thinking: '思考状态', message: '模型说明', command: '命令', tool: '工具', output: '输出', progress: '文件进度', diagnostic: '诊断', phase: '阶段', agent: 'Agent', state: '状态', error: '错误' };
  function make(tag, cls, text) { const node = document.createElement(tag); if (cls) node.className = cls; if (text !== undefined) node.textContent = text; return node; }
  function build() {
    host = document.getElementById('task-activity'); host.replaceChildren();
    const head = make('div', 'activity-heading'), title = make('div'); title.append(make('h2', '', '实时执行'), make('p', 'hint', '实际工具调用、输出与模型公开说明；推理事件显示为状态。'));
    status = make('span', 'badge', '正在连接…'); status.id = 'activity-status'; status.setAttribute('role', 'status'); head.append(title, status);
    const toolbar = make('div', 'activity-toolbar'), label = make('label', '', '显示');
    const select = make('select'); select.id = 'activity-filter';
    for (const [value, text] of [['all','全部过程'],['tools','工具与命令'],['model','模型说明']]) { const option = make('option', '', text); option.value = value; select.append(option); }
    select.value = filter; select.onchange = () => { filter = select.value; renderEntries(); }; label.append(select);
    const followLabel = make('label', 'activity-follow'); follow = make('input'); follow.type = 'checkbox'; follow.checked = following; follow.id = 'activity-follow'; follow.onchange = () => { following = follow.checked; if (following) bottom(); }; followLabel.append(follow, document.createTextNode('跟随最新'));
    earlier = make('button', '', '读取更早记录'); earlier.id = 'activity-earlier'; earlier.type = 'button'; earlier.onclick = olderPage;
    latest = make('button', '', '回到最新'); latest.id = 'activity-latest'; latest.type = 'button'; latest.onclick = latestPage;
    follow.onchange = () => { following = follow.checked; if (following && historical) latestPage(); else if (following) bottom(); controls(); };
    toolbar.append(label, followLabel, earlier, latest); last = make('p', 'hint'); last.id = 'activity-last';
    list = make('ol', 'activity-feed'); list.id = 'activity-feed'; list.setAttribute('aria-label', '执行过程记录'); list.setAttribute('role', 'log'); list.setAttribute('aria-live', 'off'); list.tabIndex = 0;
    list.addEventListener('wheel', event => { if (event.deltaY < 0) { following = false; follow.checked = false; controls(); } }, { passive: true });
    list.addEventListener('keydown', event => { if (['PageUp','Home','ArrowUp'].includes(event.key)) { following = false; follow.checked = false; controls(); } });
    host.append(head, toolbar, last, list); renderEntries();
  }
  function visible(event) { return filter === 'all' || (filter === 'model' ? ['message','thinking','agent'].includes(event.kind) : ['command','tool','output','progress','diagnostic'].includes(event.kind)); }
  function row(event) {
    const node = make('li', 'activity-event'); node.dataset.sequence = event.sequence; node.dataset.kind = event.kind;
    const meta = make('div', 'activity-meta'), time = make('time', '', new Date(event.time).toLocaleTimeString('zh-CN', { hour12: false })); time.dateTime = event.time;
    meta.append(time, make('span', '', labels[event.kind] || '活动'));
    if (event.provider) meta.append(make('span', '', event.provider + (event.model ? ' · ' + event.model : '')));
    if (event.role) meta.append(make('span', '', roleTitle(event.role, event.domain)));
    node.append(meta);
    if (event.detail) { const details = make('details'), summary = make('summary', '', event.title), body = make('pre', 'activity-output', event.detail); details.append(summary, body); if (event.kind === 'message') details.open = true; node.append(details); }
    else node.append(make('p', 'activity-title', event.title));
    return node;
  }
  function bottom() {
    if (!list || !following || !host.getClientRects().length) return;
    const scroll = document.getElementById('workspace-content') || list;
    scroll.scrollTop = scroll.scrollHeight;
  }
  function renderEntries() {
    if (!list) return;
    const expanded = new Set([...list.querySelectorAll('li:has(details[open])')].map(item => item.dataset.sequence));
    const scroll = list.scrollTop; list.replaceChildren();
    const shown = entries.filter(visible);
    for (const event of shown) { const node = row(event); if (expanded.has(String(event.sequence)) && node.querySelector('details')) node.querySelector('details').open = true; list.append(node); }
    if (!shown.length) list.append(make('li', 'activity-empty', entries.length ? '这个分类暂时没有记录。' : '此任务尚无实时记录。旧任务不会补造执行过程。'));
    if (limited || entries.length >= 500) list.append(make('li', 'activity-empty', limited ? '原始实时记录已达到保存上限；已有历史可分页查看，最终核验仍继续更新。' : '当前窗口最多显示 500 条，可读取更早记录或回到最新。'));
    if (following) bottom(); else list.scrollTop = scroll;
  }
  function updateStatus(active) {
    if (!status || !task) return;
    const stopping = task.state === 'cancelling';
    status.textContent = !connected ? '连接中断，正在重试' : historical ? '正在查看历史记录' : stopping ? '正在停止' : active ? '执行中' : working.includes(task.state) ? '等待执行记录' : '本次执行已结束';
    const newest = entries.at(-1), age = newest ? Math.max(0, Math.floor((Date.now() - new Date(newest.time).getTime()) / 1000)) : null;
    last.textContent = newest ? (historical ? '当前历史窗口末条：' : '最近动作：') + newest.title + ' · ' + (age < 60 ? age + ' 秒前' : Math.floor(age / 60) + ' 分钟前') : '等待真实事件；本地准备不会上传项目。';
  }
  function controls() {
    if (!earlier) return;
    earlier.disabled = busy || !hasEarlier;
    latest.disabled = !historical && following && initialized && connected;
  }
  function unique(events) { const seen = new Set(); return events.filter(event => !seen.has(event.sequence) && seen.add(event.sequence)); }
  function applyPage(result, mode) {
    if (!Array.isArray(result.events)) throw Error('记录格式无效');
    if (mode === 'older') {
      entries = unique([...result.events, ...entries]).slice(0, 500);
      before = result.before_cursor || 0; hasEarlier = !!result.has_earlier;
    } else if (mode === 'tail') {
      entries = result.events.slice(-500); cursor = result.cursor; before = result.before_cursor || 0;
      hasEarlier = !!result.has_earlier; initialized = true;
    } else {
      if (result.cursor < cursor) { initialized = false; entries = []; return; }
      const merged = unique([...entries, ...result.events]);
      if (merged.length > 500) { before = merged[merged.length - 501].sequence; hasEarlier = true; }
      entries = merged.slice(-500); cursor = result.cursor;
    }
    if (mode !== 'forward' || result.events.length) renderEntries();
    if (mode === 'older') (document.getElementById('workspace-content') || list).scrollTop = 0;
  }
  async function requestPage(generation, mode) {
    if (!task || generation !== epoch || busy) return;
    busy = true; controller = new AbortController(); controls(); let active = false;
    const query = mode === 'older' ? 'before=' + before : mode === 'tail' ? 'tail=1' : 'after=' + cursor;
    try {
      const response = await fetch('/api/v1/reliable/tasks/' + encodeURIComponent(task.id) + '/activity?' + query, { signal: controller.signal });
      if (generation !== epoch) return;
      if (!response.ok) { if (mode === 'forward' && response.status === 409) initialized = false; throw Error('记录暂不可用'); }
      const result = await response.json(); if (generation !== epoch) return;
      connected = true; active = !!result.active; limited = !!result.limited;
      applyPage(result, mode);
      updateStatus(active);
    } catch (error) { if (generation === epoch && error.name !== 'AbortError') { connected = false; updateStatus(active); } }
    finally {
      if (generation === epoch) { busy = false; controller = null; controls(); timer = setTimeout(() => poll(generation), active || working.includes(task.state) ? 1000 : 5000); }
    }
  }
  function poll(generation) {
    if (!task || generation !== epoch) return;
    if (document.hidden || historical || !host.getClientRects().length) { timer = setTimeout(() => poll(generation), 1500); return; }
    void requestPage(generation, initialized ? 'forward' : 'tail');
  }
  function olderPage() {
    if (busy || !hasEarlier) return;
    clearTimeout(timer); historical = true; following = false; follow.checked = false;
    void requestPage(epoch, 'older');
  }
  function latestPage() {
    epoch++; clearTimeout(timer); controller?.abort(); busy = false;
    initialized = false; historical = false; following = true; follow.checked = true; connected = false; cursor = 0; before = 0; hasEarlier = false;
    entries = []; renderEntries(); controls(); void poll(epoch);
  }
  function sync(current) {
    if (current?.id === task?.id) { task = current; return; }
    epoch++; clearTimeout(timer); controller?.abort(); busy = false; task = current || null; cursor = 0; entries = []; limited = false; connected = false;
    initialized = false; before = 0; hasEarlier = false; historical = false;
    if (!task) return;
    build(); controls(); void poll(epoch);
  }
  document.getElementById('workspace-content')?.addEventListener('wheel', event => {
    if (host?.getClientRects().length && event.deltaY < 0) { following = false; follow.checked = false; controls(); }
  }, { passive: true });
  return { sync };
})();
