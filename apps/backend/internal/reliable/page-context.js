'use strict';
const ContextUI = (() => {
  let owner = '', host = null, body = null, generation = 0;
  function summary(value) {
    return { saved: `${value.saved_batches || 0} 批 · ${value.saved_messages || 0} 条项目消息已保存`, archive: value.prepared && value.complete ? `已有交接快照 · 包含 ${value.archived_messages || 0} 条消息` : '尚未准备交接快照，下次派发前自动准备', detail: value.summary || '共享记录可用不代表模型已阅读或理解。' };
  }
  async function read(task) {
    if (!body || owner !== task.id) return;
    const expected = ++generation;
    body.replaceChildren(el('p', 'hint', '正在核对已保存的项目记录…'));
    try {
      const value = await api('tasks/' + task.id + '/context/status');
      if (generation !== expected || owner !== task.id || !host?.isConnected) return;
      const text = summary(value); body.replaceChildren(el('p', '', text.saved), el('p', '', text.archive), el('p', 'hint', text.detail));
      body.append(button('刷新记录状态', '', () => read(task)));
    } catch (error) {
      if (generation === expected && owner === task.id && host?.isConnected) body.replaceChildren(el('p', 'notice error', friendly(error.message)));
    }
  }
  function sync(task) {
    const anchor = $('task-conversation')?.querySelector('.task-requirement');
    if (!task || !anchor) { owner = ''; host = null; generation++; return; }
    if (owner !== task.id || !host?.isConnected) {
      generation++; owner = task.id; host = el('details', 'shared-context-status'); host.id = 'shared-context-status';
      host.append(el('summary', '', '共享项目上下文')); body = el('div'); host.append(body); anchor.after(host);
      host.ontoggle = () => { if (host.open) void read(task); };
    }
  }
  return { sync, summary };
})();
if (typeof module !== 'undefined') module.exports = ContextUI;
