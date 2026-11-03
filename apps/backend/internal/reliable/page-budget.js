'use strict';
const StorageBudgetUI = (() => {
  const unit = 1024 ** 3;
  let root, info, form, feedback, toggle, refresh, reset, save, fields = {}, record = null, draft = null, shown = false, busy = false;
  let message = '', failed = false;
  const gate = requestGate();
  function bytes(value) { return (Number(value || 0) / unit).toFixed(2) + ' GB'; }
  function amount(text, label) {
    const value = String(text ?? '').trim();
    if (!/^\d+(?:\.\d{1,6})?$/.test(value)) throw Error(label + '额度请填写普通数字。');
    const result = Math.round(Number(value) * unit);
    if (!Number.isSafeInteger(result) || result < 64 * 1024 ** 2 || result > 1024 ** 4) throw Error(label + '额度必须在 0.0625～1024 GB 之间。');
    return result;
  }
  function payload(value) {
    if (!Number.isInteger(value?.revision) || value.revision < 0) throw Error('请先读取当前额度版本，再保存。');
    return { expected_revision: value.revision, cache_limit_bytes: amount(value.cache, '缓存'), temp_limit_bytes: amount(value.temp, '临时文件'), minimum_free_bytes: amount(value.free, '最低剩余空间') };
  }
  function reconcile(previous, current) {
    if (previous?.dirty) return previous;
    return { revision: current.revision, cache: String(current.cache_limit_bytes / unit), temp: String(current.temp_limit_bytes / unit), free: String(current.minimum_free_bytes / unit), dirty: false };
  }
  function requestGate() {
    let sequence = 0;
    const begin = () => ++sequence, owns = token => token === sequence;
    async function run(work, success, error, finish) {
      const token = begin();
      try { const result = await work(); if (owns(token)) success(result); }
      catch (failure) { if (owns(token)) error(failure); }
      finally { if (owns(token)) finish(); }
    }
    return { begin, owns, run };
  }
  function showStatus(text, error = false) { message = text; failed = error; if (feedback) { feedback.textContent = text; feedback.dataset.error = String(error); } }
  function paint() {
    if (!root) return;
    toggle.disabled = busy;
    form.hidden = !shown || !record; info.hidden = !shown; refresh.hidden = !shown; toggle.textContent = shown ? '收起空间预算' : '查看空间预算';
    toggle.setAttribute('aria-expanded', String(shown));
    refresh.disabled = busy; save.disabled = busy || !record || draft?.revision !== record.revision; reset.disabled = busy || !record;
    for (const [key, input] of Object.entries(fields)) { input.disabled = busy; if (draft && document.activeElement !== input) input.value = draft[key]; }
    info.replaceChildren();
    if (record && shown) {
      info.append(el('p', '', '缓存 ' + bytes(record.cache_bytes) + ' / ' + bytes(record.cache_limit_bytes) + '；临时文件 ' + bytes(record.temp_bytes) + ' / ' + bytes(record.temp_limit_bytes)));
      info.append(el('p', 'hint', record.free_known ? '数据磁盘剩余 ' + bytes(record.free_bytes) + '；最低余量 ' + bytes(record.minimum_free_bytes) : '数据磁盘剩余空间暂时无法确认。'));
      for (const warning of record.warnings || []) info.append(el('p', 'hint storage-budget-warning', warning));
      if (draft?.dirty && draft.revision !== record.revision) info.append(el('p', 'hint storage-budget-warning', '其他窗口已修改额度。你的输入已保留，请载入最新值后再保存。'));
    }
    showStatus(message, failed);
  }
  async function perform(work, success) {
    if (busy) return;
    busy = true; showStatus('正在读取或保存空间设置…'); paint();
    await gate.run(work, success, error => showStatus(friendly(error.message) + '。输入已保留。', true), () => { busy = false; paint(); });
  }
  async function load(resetDraft = false) {
    await perform(() => api('storage/budget'), result => { record = result; draft = reconcile(resetDraft ? null : draft, result); showStatus('空间统计已更新。'); });
  }
  function control(parent, key, label) {
    const box = el('div', 'field'), caption = el('label', '', label), input = el('input');
    input.id = 'storage-budget-' + key; input.type = 'number'; input.min = '0.0625'; input.max = '1024'; input.step = 'any'; caption.htmlFor = input.id;
    input.oninput = () => { if (draft) { draft[key] = input.value; draft.dirty = true; } }; fields[key] = input; box.append(caption, input); parent.append(box);
  }
  function create() {
    root = el('section', 'storage-budget'); root.setAttribute('aria-label', '空间预算与提醒');
    const actions = el('div', 'project-controls'); toggle = button('查看空间预算', '', () => { shown = !shown; paint(); if (shown && !record) void load(); });
    toggle.setAttribute('aria-controls', 'storage-budget-info'); refresh = button('刷新统计', '', () => load()); actions.append(toggle, refresh);
    info = el('div', 'storage-budget-info'); info.id = 'storage-budget-info';
    form = el('form', 'storage-budget-form'); const controls = el('div', 'storage-budget-fields');
    control(controls, 'cache', '缓存提醒额度（GB）'); control(controls, 'temp', '临时文件提醒额度（GB）'); control(controls, 'free', '最低剩余空间（GB）');
    const note = el('p', 'hint', '1 GB 按 1024 MB 换算。缓存与临时额度用于提醒；低于最低余量时暂停新的副本与预览分配。不会自动删除文件。');
    const footer = el('div', 'project-controls'); save = button('保存空间设置', '', null); save.type = 'submit'; reset = button('载入最新值', '', () => load(true)); footer.append(save, reset);
    form.onsubmit = event => { event.preventDefault(); let body; try { body = payload(draft); } catch (error) { showStatus(error.message, true); return; } void perform(() => api('storage/budget', body), result => { record = result; draft = reconcile(null, result); showStatus('空间设置已保存。'); }); };
    form.append(controls, note, footer); feedback = el('p', 'project-feedback storage-budget-feedback'); feedback.setAttribute('role', 'status'); feedback.setAttribute('aria-live', 'polite');
    root.append(el('h3', '', '空间预算与提醒'), actions, info, form, feedback);
  }
  function mount(parent) { if (!root) create(); parent.append(root); paint(); }
  return { mount, payload, reconcile, requestGate };
})();
