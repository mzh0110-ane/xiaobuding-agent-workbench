'use strict';
const AppearanceUI = (() => {
  const key = 'xiaobuding.background';
  let state, customURL = '', customStored = false, imageVersion = 0, modeVersion = 0;
  function validate(file) {
    if (!(file instanceof Blob) || !['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) throw new Error('请选择 PNG、JPG 或 WebP 图片。');
    if (file.size > 20 * 1024 * 1024) throw new Error('图片超过 20 MB，请选择小一点的图片。');
  }
  async function imageURL(file) {
    validate(file);
    const url = URL.createObjectURL(file), image = new Image(); image.src = url;
    try { await image.decode(); return url; }
    catch { URL.revokeObjectURL(url); throw new Error('这张图片无法读取，请换一张有效图片。'); }
  }
  function storedImage(write, value) {
    return new Promise((resolve, reject) => {
      let request;
      try { request = indexedDB.open('xiaobuding-appearance', 1); } catch (error) { reject(error); return; }
      request.onupgradeneeded = () => request.result.createObjectStore('images');
      request.onerror = request.onblocked = () => reject(new Error('图片存储暂时不可用。'));
      request.onsuccess = () => {
        const db = request.result;
        try {
          const transaction = db.transaction('images', write ? 'readwrite' : 'readonly'), store = transaction.objectStore('images');
          const operation = write ? store.put(value, 'custom') : store.get('custom');
          transaction.oncomplete = () => { db.close(); resolve(operation.result); };
          transaction.onerror = transaction.onabort = () => { db.close(); reject(new Error('图片保存失败。')); };
        } catch (error) { db.close(); reject(error); }
      };
    });
  }
  function notice(message, error = false) {
    $('background-status').textContent = message;
    $('background-status').dataset.error = String(error);
  }
  function normalize(value) {
    return {
      image: ['none', 'custom'].includes(value?.image) ? value.image : 'portrait',
      transparency: typeof value?.transparency === 'number' && Number.isFinite(value.transparency) ? Math.round(Math.min(100, Math.max(0, value.transparency))) : 80,
    };
  }
  function read() { try { return normalize(JSON.parse(localStorage.getItem(key))); } catch { return normalize(null); } }
  function apply(value) {
    state = value;
    document.documentElement.dataset.background = value.image;
    document.documentElement.style.setProperty('--background-opacity', String((100 - value.transparency) / 100));
    if (value.image === 'custom' && customURL) document.documentElement.style.setProperty('--custom-background-image', 'url("' + customURL + '")');
    else document.documentElement.style.removeProperty('--custom-background-image');
    $('background-image').value = value.image;
    $('background-image').querySelector('option[value=custom]').disabled = !customURL;
    $('background-transparency').value = value.transparency;
    $('background-transparency').disabled = value.image === 'none';
    $('background-transparency-value').value = value.transparency + '%';
  }
  function remember() {
    const saved = { ...state, image: state.image === 'custom' && !customStored ? 'portrait' : state.image };
    try { localStorage.setItem(key, JSON.stringify(saved)); return true; } catch { return false; }
  }
  async function restore() {
    try {
      const file = await storedImage(false);
      if (!file) throw new Error('missing image');
      const url = await imageURL(file);
      if (imageVersion) { URL.revokeObjectURL(url); return; }
      customURL = url; customStored = true; apply(state);
    } catch {
      if (!imageVersion && state.image === 'custom') {
        apply({ ...state, image: 'portrait' }); remember();
        notice('自选图片暂时无法恢复，请重新选择图片。', true);
      }
    }
  }
  async function choose() {
    const file = $('background-file').files[0]; $('background-file').value = '';
    if (!file) return;
    let version = imageVersion;
    try {
      validate(file); version = ++imageVersion; const selection = ++modeVersion;
      notice('正在保存图片…');
      const url = await imageURL(file);
      if (version !== imageVersion) { URL.revokeObjectURL(url); return; }
      let persisted = false;
      try { await storedImage(true, file); persisted = true; } catch {}
      if (version !== imageVersion) { URL.revokeObjectURL(url); return; }
      const oldURL = customURL; customURL = url; customStored = persisted;
      apply(selection === modeVersion ? { ...state, image: 'custom' } : state);
      const saved = remember(); if (oldURL) URL.revokeObjectURL(oldURL);
      notice(persisted && saved ? '图片已保存，可随时切换使用。' : '已临时使用；浏览器未能保存，下次需重新选择。');
    } catch (error) { if (version === imageVersion) notice(error.message, true); }
  }
  function init() {
    apply(read());
    const update = () => {
      const value = normalize({ image: $('background-image').value, transparency: Number($('background-transparency').value) });
      if (value.image === 'custom' && !customURL) value.image = 'portrait';
      apply(value); remember();
    };
    $('background-image').onchange = () => { modeVersion++; update(); };
    $('background-transparency').oninput = update;
    $('background-pick').onclick = () => $('background-file').click();
    $('background-file').onchange = choose;
    window.addEventListener('beforeunload', () => { if (customURL) URL.revokeObjectURL(customURL); });
    void restore();
  }
  return { init };
})();
