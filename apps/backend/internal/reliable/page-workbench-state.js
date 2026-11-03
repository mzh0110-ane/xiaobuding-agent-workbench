'use strict';
const WorkbenchState = (() => {
  function groups(tasks) {
    const grouped = new Map();
    for (const task of tasks) {
      const id = task.project_id || String(task.project || '').replace(/\\/g, '/').toLowerCase();
      if (!grouped.has(id)) grouped.set(id, { id, project: task.project, tasks: [] });
      grouped.get(id).tasks.push(task);
    }
    return [...grouped.values()].map(group => {
      group.tasks.sort((a, b) => String(b.created || '').localeCompare(String(a.created || '')));
      group.current = group.tasks.find(task => !task.successor_task_id) || group.tasks[0];
      return group;
    }).sort((a, b) => String(b.current.created || '').localeCompare(String(a.current.created || '')));
  }
  function reader(request) {
    const cached = new Map(), pending = new Map();
    function read(path) {
      if (pending.has(path)) return pending.get(path);
      const previous = cached.get(path), headers = previous?.etag ? { 'If-None-Match': previous.etag } : {};
      const promise = Promise.resolve().then(() => request(path, { headers })).then(async response => {
        if (response.status === 304) {
          if (!previous) throw Error('缓存记录已失效，请刷新。');
          return previous.value;
        }
        const value = await response.json();
        if (!response.ok) throw Error(value.error || '读取项目记录失败。');
        cached.set(path, { etag: response.headers.get('ETag'), value });
        return value;
      }).finally(() => pending.delete(path));
      pending.set(path, promise); return promise;
    }
    return { read };
  }
  function shouldNotify(previous, current) {
    return !!previous && previous !== current && ['blocked', 'interrupted', 'awaiting_acceptance', 'exhausted'].includes(current);
  }
  function previewURL(value) {
    try {
      const url = new URL(value);
      return url.protocol === 'http:' && ['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname) && url.port && !url.username && !url.password ? url.href : '';
    } catch { return ''; }
  }
  function navigationItem(summary, detail, selected) { return detail && summary.id === selected ? { ...detail, project_id: summary.project_id } : summary; }
  function mergeMessages(...windows) { const messages = new Map(); for (const window of windows) for (const message of window || []) messages.set(message.id, message); return [...messages.values()].sort((a, b) => Number.isInteger(a._position) && Number.isInteger(b._position) ? a._position - b._position : 0); }
  function requestGate() { let revision = 0; return { begin: () => ++revision, owns: token => token === revision }; }
  function earlierCursor(messages, before) { const seen = new Set(messages.map(message => message._position)); while (before > 0 && seen.has(before - 1)) before--; return before; }
  async function summaryPages(read, count, query = '') {
    const items = new Map(); let offset = 0, total = 0, next = -1;
    for (let page = 0; page < count; page++) {
      const value = await read('tasks?view=summary' + (offset ? '&offset=' + offset : '') + (query ? '&q=' + encodeURIComponent(query) : ''));
      const tasks = Array.isArray(value) ? value : value.tasks;
      for (const task of tasks) items.set(task.id, task);
      total = value.total ?? tasks.length; next = value.next_offset ?? -1;
      if (next < 0 || next <= offset) break; offset = next;
    }
    return { tasks: [...items.values()], total, next_offset: next };
  }
  return { groups, reader, shouldNotify, previewURL, navigationItem, mergeMessages, earlierCursor, requestGate, summaryPages };
})();
