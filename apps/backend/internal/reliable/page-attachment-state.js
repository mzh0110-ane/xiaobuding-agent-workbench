'use strict';
const AttachmentState = (() => {
  const maxFiles = 10, maxBytes = 100 * 1024 * 1024;
  function selectionError(current, files) {
    if (current.length + files.length > maxFiles) return '每次最多添加 10 个附件，请先移除不需要的文件。';
    if (files.some(file => !Number.isSafeInteger(file.size) || file.size < 0 || file.size > maxBytes)) return '单个附件不能超过 100 MB。';
    if ([...current, ...files].reduce((sum, file) => sum + file.size, 0) > maxBytes) return '附件总大小不能超过 100 MB。';
    return '';
  }
  function blocksSend(items) { return items.some(item => item.status !== 'ready'); }
  function ids(items) { return items.filter(item => item.status === 'ready' && item.attachment?.id).map(item => item.attachment.id); }
  function settle(items, key, generation, patch) { return items.map(item => item.key === key && item.generation === generation ? { ...item, ...patch } : item); }
  function bytes(value) { return value < 1024 ? value + ' B' : value < 1024 * 1024 ? (value / 1024).toFixed(1) + ' KB' : (value / (1024 * 1024)).toFixed(1) + ' MB'; }
  return { selectionError, blocksSend, ids, settle, bytes };
})();
if (typeof module !== 'undefined') module.exports = AttachmentState;
