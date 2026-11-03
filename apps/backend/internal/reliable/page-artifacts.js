'use strict';
const ResultArtifacts = (() => {
  const reasons = {
    preview_no_static_entry: '没有识别到可直接打开的 HTML 成果，可先查看修改文件或打开工作副本。',
    preview_dependencies_missing: '此副本缺少已有 Vite 依赖。可在副本准备依赖后再试，小布丁不会自动安装。',
    preview_vite_basic: '基础 Vite 源码预览，不加载项目配置或插件。',
    preview_static_ready: '可预览此副本中的静态页面。',
    preview_build_unverified: '这是已有构建文件，尚未核实它是否包含最新源码修改。',
    preview_process_exited: '预览进程已退出，可查看日志后重新启动。',
    preview_cleanup_unknown: '上次预览的停止尚未确认，请处理进程状态后再启动。',
    preview_source_changed: '副本已变化，当前预览与启动时的版本不同，请停止后重新启动。',
    preview_stopped: '预览已停止。',
  };
  function reason(raw) { return reasons[raw] || (/preview_busy|task.*running/.test(raw || '') ? '任务仍在执行，结束后可查看本批文件与预览。' : friendly(raw)); }
  function mount(parent, task) {
    if (!task.worktree) return;
    const panel = el('details', 'proof-disclosure result-artifacts'); panel.dataset.disclosure = 'artifacts';
    panel.append(el('summary', '', '打开本批成果与修改文件')); const body = el('div'); panel.append(body); parent.append(panel);
    let loaded = false;
    panel.ontoggle = () => { if (panel.open && !loaded && panel.isConnected) { loaded = true; void load(body, task); } };
  }
  async function load(parent, task) {
    const preview = el('section', 'artifact-preview'), files = el('section', 'artifact-files');
    parent.replaceChildren(preview, files); preview.append(el('p', 'hint', '正在核对副本与预览方式…'));
    await Promise.allSettled([
      api('tasks/' + task.id + '/preview').then(value => { if (parent.isConnected) renderPreview(preview, task, value); }).catch(error => { if (parent.isConnected) preview.replaceChildren(el('p', 'hint', reason(error.message))); }),
      loadFiles(files, task),
    ]);
  }
  function renderPreview(parent, task, value) {
    if (!parent.previewGate) parent.previewGate = WorkbenchState.requestGate();
    parent.replaceChildren(el('h3', '', '修改后的运行效果'));
    parent.append(el('p', 'hint', '来源：本批工作副本。预览不等于检查或验收通过。'), el('p', 'path', task.worktree));
    const message = value.build_unverified ? reasons.preview_build_unverified : value.stale ? reasons.preview_source_changed : reason(value.reason);
    if (message) parent.append(el('p', 'hint', message));
    if (value.source_hash) parent.append(el('p', 'hint', '启动版本：' + value.source_hash.slice(0, 16) + (value.current_source_hash ? ' · 当前版本：' + value.current_source_hash.slice(0, 16) : '')));
    const controls = el('div', 'artifact-controls');
    const select = el('select'); select.setAttribute('aria-label', '选择本批预览入口');
    const candidates = value.candidates?.length ? value.candidates : value.entry ? [value.entry] : [];
    for (const candidate of candidates) { const option = el('option', '', candidate === 'vite' ? '基础 Vite 源码预览' : candidate); option.value = candidate; select.append(option); }
    select.value = value.entry || candidates[0] || ''; if (candidates.length) controls.append(select);
    const execute = async action => {
      const request = parent.previewGate.begin();
      for (const control of controls.querySelectorAll('button,select')) control.disabled = true;
      try { const result = await api('tasks/' + task.id + '/preview', { action, ...(action === 'start' ? { entry: select.value } : {}) }); if (parent.isConnected && parent.previewGate.owns(request)) renderPreview(parent, task, result); }
      catch (error) { if (parent.isConnected && parent.previewGate.owns(request)) parent.append(el('p', 'notice error', reason(error.message))); }
      finally { for (const control of controls.querySelectorAll('button,select')) control.disabled = false; }
    };
    if (value.supported && !value.running) controls.append(button('启动预览', '', () => execute('start')));
    const url = WorkbenchState.previewURL(value.url);
    if (url && value.running) { const open = el('a', 'batch-link', '打开修改后的页面'); open.href = url; open.target = '_blank'; open.rel = 'noopener'; controls.append(open); }
    if (value.url || value.running) controls.append(button('停止预览', '', () => execute('stop')));
    controls.append(button('刷新预览状态', '', () => { const request = parent.previewGate.begin(); void api('tasks/' + task.id + '/preview').then(result => { if (parent.isConnected && parent.previewGate.owns(request)) renderPreview(parent, task, result); }).catch(error => { if (parent.isConnected && parent.previewGate.owns(request)) parent.append(el('p', 'hint', reason(error.message))); }); }));
    parent.append(controls);
    if (value.log_tail) { const log = el('details'); log.append(el('summary', '', '预览进程日志（最近 16 KiB）'), el('pre', 'result-file-viewer', value.log_tail)); parent.append(log); }
  }
  async function loadFiles(parent, task) {
    parent.replaceChildren(el('h3', '', '本批修改文件'));
    const list = el('div'), viewer = el('pre', 'result-file-viewer'); viewer.hidden = true; viewer.tabIndex = 0;
    const status = el('p', 'hint'), more = button('读取更多文件', '', () => page());
    parent.append(list, status, more, viewer); let offset = 0, reading = false, fileGeneration = 0;
    async function readFile(file) {
      const generation = ++fileGeneration; viewer.hidden = false; viewer.textContent = '正在读取 ' + file.path + '…';
      try { const value = await api('tasks/' + task.id + '/files?path=' + encodeURIComponent(file.path)); if (viewer.isConnected && generation === fileGeneration) viewer.textContent = value.path + '\nSHA256 ' + value.sha256 + '\n\n' + value.content; }
      catch (error) { if (viewer.isConnected && generation === fileGeneration) viewer.textContent = friendly(error.message); }
    }
    async function page() {
      if (reading) return; reading = true; more.disabled = true; status.textContent = '正在读取修改文件…';
      try {
        const value = await api('tasks/' + task.id + '/files?offset=' + offset + '&limit=50'); if (!parent.isConnected) return;
        for (const file of value.files) {
          const row = el('div', 'result-artifact'); row.append(el('span', '', ({ new: '新增', modified: '修改', deleted: '删除' })[file.status] + ' · ' + file.path));
          if (file.status !== 'deleted') row.append(button('查看文件', '', () => readFile(file))); list.append(row);
        }
        status.textContent = value.total ? '共 ' + value.total + ' 个可列出的修改文件；文本查看上限 512 KiB。' : '尚无可列出的源码修改。';
        offset = value.next_offset; more.hidden = offset < 0;
      } catch (error) { if (parent.isConnected) status.textContent = friendly(error.message); }
      finally { reading = false; more.disabled = false; }
    }
    await page();
  }
  return { mount };
})();
