'use strict';
const AttachmentUI = (() => {
  const scopes = new Map();
  function node(tag, cls, text) { const value = document.createElement(tag); if (cls) value.className = cls; if (text !== undefined) value.textContent = text; return value; }
  function metadata(value) { return value && typeof value.id === 'string' && value.id.length > 0 && typeof value.name === 'string' && Number.isSafeInteger(value.size) && value.size >= 0; }
  function state(scope) {
    if (!scopes.has(scope)) {
      let saved = []; try { saved = JSON.parse(sessionStorage.getItem('xiaobuding.attachments.' + scope) || '[]'); } catch {}
      const entries = Array.isArray(saved) ? saved.filter(value => metadata(value)).slice(0, 10).map(attachment => ({ key: crypto.randomUUID(), generation: 0, name: attachment.name, size: attachment.size, status: 'ready', attachment })) : [];
      scopes.set(scope, { scope, entries, uploads: new Map(), disabled: false });
    }
    return scopes.get(scope);
  }
  function remember(current) { try { sessionStorage.setItem('xiaobuding.attachments.' + current.scope, JSON.stringify(current.entries.filter(entry => entry.status === 'ready').map(entry => entry.attachment))); } catch {} }
  function publish(current) { remember(current); render(current); current.onChange?.(); }
  function fileLink(attachment) {
    const link = node('a', 'attachment-download'); link.href = '/api/v1/reliable/attachments/' + encodeURIComponent(attachment.id); link.download = attachment.name; link.title = '下载 ' + attachment.name;
    if (attachment.image) { const image = node('img', 'attachment-thumbnail'); image.src = link.href + '/preview'; image.alt = ''; image.loading = 'lazy'; link.append(image); }
    else link.append(DesktopUI.icon('file'));
    const label = node('span', 'attachment-label'); label.append(node('strong', '', attachment.name), node('small', '', AttachmentState.bytes(attachment.size))); link.append(label); return link;
  }
  function historical(host, attachments) {
    if (!attachments?.length) return;
    const list = node('ul', 'attachment-history'); list.setAttribute('aria-label', '附件记录');
    for (const attachment of attachments.filter(metadata)) { const row = node('li'); row.dataset.attachmentId = attachment.id; row.append(fileLink(attachment)); list.append(row); }
    host.append(list);
  }
  function row(current, entry) {
    const item = node('li', 'attachment-item'); item.dataset.attachmentId = entry.attachment?.id || entry.key; item.dataset.uploadStatus = entry.status;
    if (entry.preview || entry.attachment?.image) { const image = node('img', 'attachment-thumbnail'); image.src = entry.preview || '/api/v1/reliable/attachments/' + encodeURIComponent(entry.attachment.id) + '/preview'; image.alt = ''; item.append(image); }
    else item.append(DesktopUI.icon('file'));
    const label = node('span', 'attachment-label'); label.append(node('strong', '', entry.name), node('small', '', AttachmentState.bytes(entry.size) + ' · ' + (entry.status === 'ready' ? '已上传' : entry.status === 'failed' ? entry.error : '正在上传')));
    if (entry.status === 'uploading') { const progress = node('progress'); progress.max = 100; if (entry.progress !== undefined) progress.value = entry.progress; progress.setAttribute('aria-label', '上传进度 ' + entry.name); label.append(progress); }
    item.append(label);
    if (entry.status === 'failed' && entry.file) { const retry = node('button', '', '重试'); retry.type = 'button'; retry.setAttribute('aria-label', '重试上传 ' + entry.name); retry.disabled = current.disabled; retry.onclick = () => upload(current, entry.key); item.append(retry); }
    const remove = node('button', 'attachment-remove', '×'); remove.type = 'button'; remove.setAttribute('aria-label', '移除附件 ' + entry.name); remove.disabled = current.disabled; remove.onclick = () => removeEntry(current, entry.key); item.append(remove); return item;
  }
  function render(current) {
    if (!current.host?.isConnected) return;
    current.list.replaceChildren(...current.entries.map(entry => row(current, entry)));
    current.picker.disabled = current.disabled; current.input.disabled = current.disabled;
    current.status.textContent = current.error || (current.entries.some(entry => entry.status === 'uploading') ? '附件正在保存到本机，完成后即可发送。' : '可选择任意格式、拖入文件或粘贴图片；最多 10 个，总计 100 MB。');
    current.status.dataset.error = String(!!current.error);
  }
  function removeEntry(current, key) {
    const entry = current.entries.find(item => item.key === key); if (!entry || current.disabled) return;
    current.entries = current.entries.filter(item => item.key !== key); current.uploads.get(key)?.abort(); current.uploads.delete(key); if (entry.preview) URL.revokeObjectURL(entry.preview); current.error = ''; publish(current);
  }
  function finish(current, key, generation, patch) {
    if (!current.entries.some(entry => entry.key === key && entry.generation === generation)) return;
    current.entries = AttachmentState.settle(current.entries, key, generation, patch); current.uploads.delete(key); publish(current);
  }
  function upload(current, key) {
    const entry = current.entries.find(item => item.key === key); if (!entry?.file || current.disabled) return;
    const generation = ++entry.generation; entry.status = 'uploading'; entry.error = ''; entry.progress = undefined; publish(current);
    const xhr = new XMLHttpRequest(); current.uploads.set(key, xhr); xhr.open('POST', '/api/v1/reliable/attachments'); xhr.setRequestHeader('X-XBD-Token', token); xhr.timeout = 180000;
    xhr.upload.onprogress = event => { current.entries = AttachmentState.settle(current.entries, key, generation, { progress: event.lengthComputable ? Math.round(100 * event.loaded / event.total) : undefined }); render(current); };
    const failed = text => finish(current, key, generation, { status: 'failed', error: friendly(text || 'attachment_upload_failed: upload failed') });
    xhr.onerror = xhr.ontimeout = () => failed('attachment_upload_failed: connection failed'); xhr.onabort = () => failed('attachment_upload_failed: upload cancelled');
    xhr.onload = () => { let result; try { result = JSON.parse(xhr.responseText); } catch { failed('attachment_upload_failed: unreadable response'); return; } if (xhr.status !== 201 || !metadata(result)) { failed(result.error); return; } finish(current, key, generation, { status: 'ready', attachment: result }); };
    const body = new FormData(); body.append('file', entry.file, entry.name); xhr.send(body);
  }
  function add(current, files) {
    if (current.disabled || !files.length) return;
    current.error = AttachmentState.selectionError(current.entries, files); if (current.error) { render(current); return; }
    const added = files.map(file => ({ key: crypto.randomUUID(), generation: 0, name: file.name, size: file.size, status: 'queued', file, preview: ['image/png','image/jpeg','image/webp','image/gif'].includes(file.type) ? URL.createObjectURL(file) : '' }));
    current.entries.push(...added); publish(current); for (const entry of added) upload(current, entry.key);
  }
  function mount(scope, host, options) {
    const current = state(scope); current.host = host; current.onChange = options.onChange; current.disabled = !!options.disabled; host.classList.add('attachments'); host.replaceChildren();
    current.list = node('ul', 'attachment-list'); current.list.setAttribute('aria-label', '待发送附件');
    current.picker = node('button', 'attachment-picker', '添加附件'); current.picker.type = 'button'; current.picker.prepend(DesktopUI.icon('file'));
    current.input = node('input'); current.input.type = 'file'; current.input.multiple = true; current.input.hidden = true; current.input.id = options.id + '-file'; current.picker.setAttribute('aria-controls', current.input.id);
    current.status = node('p', 'attachment-help'); current.status.setAttribute('role', 'status'); current.status.setAttribute('aria-live', 'polite');
    host.append(current.list, current.picker, current.input, current.status); current.picker.onclick = () => current.input.click(); current.input.onchange = () => { const files = [...current.input.files]; current.input.value = ''; add(current, files); };
    options.pasteTarget.addEventListener('paste', event => { const files = [...(event.clipboardData?.files || [])]; if (!files.length || current.disabled) return; event.preventDefault(); add(current, files); });
    const target = options.dropTarget; target.addEventListener('dragover', event => { if (!event.dataTransfer?.types.includes('Files') || current.disabled) return; event.preventDefault(); target.classList.add('attachment-dragover'); });
    target.addEventListener('dragleave', event => { if (!target.contains(event.relatedTarget)) target.classList.remove('attachment-dragover'); });
    target.addEventListener('drop', event => { target.classList.remove('attachment-dragover'); if (!event.dataTransfer?.files.length || current.disabled) return; event.preventDefault(); add(current, [...event.dataTransfer.files]); });
    render(current);
  }
  function clear(scope, notify = true) { const current = state(scope); for (const entry of [...current.entries]) { if (entry.preview) URL.revokeObjectURL(entry.preview); current.uploads.get(entry.key)?.abort(); } current.entries = []; current.uploads.clear(); current.error = ''; if (notify) publish(current); else { remember(current); render(current); } }
  function set(scope, attachments) { clear(scope, false); const current = state(scope); current.entries = (attachments || []).filter(metadata).map(attachment => ({key:crypto.randomUUID(),generation:0,name:attachment.name,size:attachment.size,status:'ready',attachment})); publish(current); }
  function disabled(scope, value) { const current = state(scope); if (current.disabled !== !!value) { current.disabled = !!value; render(current); } }
  for (const type of ['dragover','drop']) document.addEventListener(type, event => { if (event.dataTransfer?.types.includes('Files')) event.preventDefault(); });
  return { mount, historical, clear, set, disabled, ids: scope => AttachmentState.ids(state(scope).entries), blocked: scope => AttachmentState.blocksSend(state(scope).entries) };
})();
