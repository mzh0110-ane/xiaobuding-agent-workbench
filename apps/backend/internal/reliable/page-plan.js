'use strict';
const PlanUI = (() => {
  const effortNames = { off: '关闭思考', low: '低', medium: '中', high: '高', xhigh: '更高', max: '最大', ultra: '极高' };
  const titles = { frontend: '前端界面', backend: '后端功能', auditor: '调查与规划', reviewer: '独立审查', implementer: '实施修改' };
  const modeNames = { frontend: '只做前端', backend: '只做后端', fullstack: '前后端一起' };
  function keys(plan) {
    if (!plan.mode) return ['auditor', 'implementer', 'reviewer'];
    return [...(plan.mode !== 'backend' ? ['frontend'] : []), ...(plan.mode !== 'frontend' ? ['backend'] : []), 'auditor', 'reviewer'];
  }
  function control(id, label, disabled) {
    const field = el('div', 'assignment-control'), text = el('label', '', label), select = el('select');
    select.id = id; select.disabled = disabled; text.htmlFor = id; field.append(text, select); return { field, select };
  }
  function fillOptions(select, choices, current) {
    select.replaceChildren();
    for (const choice of choices) { const option = el('option', '', choice.name); option.value = choice.id; option.disabled = !!choice.disabled; select.append(option); }
    if (current && !choices.some(c => c.id === current)) { const missing = el('option', '', current + '（目录未确认，请重新选择）'); missing.value = current; missing.disabled = true; select.append(missing); }
    select.value = current || '';
  }
  function render(parent, prefix, plan, disabled, onChange) {
    parent.replaceChildren(); parent._plan = { ...plan };
    function changed() {
      const migratedReview = parent._plan.reviewer !== 'codex';
      parent._plan = AssistantState.custom(parent._plan); onChange();
      if (migratedReview) { render(parent, prefix, parent._plan, disabled, onChange); AgentSettings.sync(tasks.find(item => item.id === selected)); }
    }
    for (const key of keys(plan)) {
      const row = el('section', 'assignment-row'), heading = el('div', 'assignment-title');
      heading.append(el('h3', '', titles[key]), el('small', '', key === 'reviewer' ? (plan.mode ? 'Codex 新会话核对源码和实际检查' : '独立新会话核对源码和实际检查') : key === 'auditor' ? '只读调查，确定分工范围与接口' : key === 'implementer' ? '历史任务的单实施者流程' : '在同一副本依次接手，范围由调查确定'));
      const tool = control(prefix + '-' + key, titles[key] + ' · 执行工具', disabled || (!!(plan.mode || plan.assistant_mode) && key === 'reviewer'));
      const model = control(prefix + '-' + key + '-model', titles[key] + ' · 模型', disabled);
      const effort = control(prefix + '-' + key + '-effort', titles[key] + ' · 思考强度', disabled);
      const note = el('small', 'catalog-note');
      const providers = (config.providers || []).filter(p => !(key === 'reviewer' && p.tool_kind === 'zcode')).filter(p => !(plan.mode || plan.assistant_mode) || key !== 'reviewer' || p.id === 'codex');
      fillOptions(tool.select, providers.map(p => ({ id: p.id, name: p.name + (p.available ? '' : '（未连接）'), disabled: !p.available })), parent._plan[key]);
      function updateChoices() {
        const p = (config.providers || []).find(p => p.id === parent._plan[key]);
        const options = p?.models?.length ? p.models : p ? [{ id: p.model, name: p.model, efforts: [] }] : [];
        const selectedModel = parent._plan[key + '_model'] || p?.model || '';
        parent._plan[key + '_model'] = selectedModel;
        fillOptions(model.select, options.map(m => ({ id: m.id, name: m.name && m.name !== m.id ? m.name + '（' + m.id + '）' : m.id })), selectedModel);
        const chosen = options.find(m => m.id === selectedModel);
        const autoName = chosen?.default_effort ? '自动（推荐：' + (effortNames[chosen.default_effort] || chosen.default_effort) + '）' : '自动（沿用原工具设置）';
        fillOptions(effort.select, [{ id: '', name: autoName }, ...(chosen?.efforts || []).map(id => ({ id, name: (effortNames[id] || id) + ' · ' + id }))], parent._plan[key + '_effort'] || '');
        note.textContent = p?.catalog_status || '使用已有原生连接；实际可用性以执行结果为准。';
        note.className = 'catalog-note' + (p?.catalog_status?.includes('未刷新') ? ' catalog-warning' : '');
      }
      tool.select.onchange = () => { parent._plan[key] = tool.select.value; const p = config.providers.find(p => p.id === tool.select.value); parent._plan[key + '_model'] = p?.model || ''; parent._plan[key + '_effort'] = ''; updateChoices(); changed(); };
      model.select.onchange = () => { parent._plan[key + '_model'] = model.select.value; parent._plan[key + '_effort'] = ''; updateChoices(); changed(); };
      effort.select.onchange = () => { parent._plan[key + '_effort'] = effort.select.value; changed(); };
      updateChoices(); row.append(heading, tool.field, model.field, effort.field, note); parent.append(row);
    }
    if (plan.mode && prefix === 'new-plan') renderScope(parent, onChange);
  }
  function renderScope(parent, onChange) {
    const details = el('details', 'domain-scope'), summary = el('summary', '', '限定前端与后端的修改目录（可选）'); details.append(summary);
    for (const key of keys(parent._plan).filter(k => k === 'frontend' || k === 'backend')) {
      const field = el('div', 'field'), label = el('label', '', titles[key] + '允许修改的目录或文件'), input = el('input');
      input.id = 'new-plan-' + key + '-paths'; label.htmlFor = input.id; input.placeholder = '留空由调查确定；多个相对路径用英文逗号分隔'; input.value = parent._plan[key + '_paths'] || ''; input.spellcheck = false;
      input.oninput = () => { parent._plan[key + '_paths'] = input.value; onChange(); }; field.append(label, input); details.append(field);
    }
    parent.append(details);
  }
  function modePicker(parent, plan, onChange) {
    parent.replaceChildren(); const fieldset = el('fieldset', 'work-mode'), legend = el('legend', '', '这次要改哪里？'); fieldset.append(legend);
    for (const [id, title] of Object.entries(modeNames)) {
      const label = el('label', 'mode-choice'), input = el('input'); input.type = 'radio'; input.name = 'work-mode'; input.id = 'work-mode-' + id; input.value = id; input.checked = plan.mode === id;
      input.onchange = () => { if (input.checked) onChange(id); }; label.append(input, el('span', '', title)); fieldset.append(label);
    }
    parent.append(fieldset);
  }
  return { render, modePicker, keys, titles, modeNames, effortNames };
})();
