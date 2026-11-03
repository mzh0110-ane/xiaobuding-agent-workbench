'use strict';
const TaskProof = (() => {
  const disclosures = new Map(); let previousTask = '', generation = 0;
  function disclosure(parent, key, title) {
    const details = el('details', 'proof-disclosure'); details.dataset.disclosure = key;
    details.append(el('summary', '', title)); parent.append(details); return details;
  }
  function evidenceLink(task, label, path) {
    const link = el('a', '', label); link.href = evidenceURL(task.id, path);
    link.target = '_blank'; link.rel = 'noopener'; return link;
  }
  function auditReport(parent, audit, includeSummary = true) {
    if (!audit) return;
    const details = disclosure(parent, 'audit', '调查依据与修改计划'), section = el('section', 'detail-section');
    if (includeSummary) section.append(el('h3', '', '调查结论'), el('p', '', audit.summary));
    if (audit.plan?.length) {
      section.append(el('h3', '', '调查阶段制定的计划'));
      const plan = el('ul'); for (const item of audit.plan) plan.append(el('li', '', item)); section.append(plan);
    }
    for (const [key, title] of [['facts', '有来源的事实'], ['inferences', '模型推断'], ['unknowns', '尚待确认']]) {
      if (!audit[key]?.length) continue;
      section.append(el('h3', '', title)); const list = el('ul'); for (const item of audit[key]) list.append(el('li', '', item)); section.append(list);
    }
    if (audit.read_files?.length) {
      section.append(el('h3', '', '调查引用的文件'));
      const files = el('ul'); for (const file of audit.read_files) files.append(el('li', '', file)); section.append(files);
    }
    details.append(section);
  }
  function resultStatus(task) {
    const checksPass = task.checks?.length > 0 && task.checks.every(check => check.exit_code === 0 && !check.verification_problem);
    const sameRequirements = (task.requirement_revision || 0) === (task.audited_revision || 0);
    if (sameRequirements && checksPass && task.review?.verdict === 'pass') {
      if (task.state === 'accepted') return ['本批已验收，修改已应用', 'good'];
      if (task.state === 'awaiting_acceptance') return ['本批核验已通过，等待你验收', 'good'];
      if (task.state === 'applying') return ['本批已核验，正在应用修改', ''];
    }
    const labels = {
      preparing: '正在准备项目', queued: '等待开始', auditing: '正在调查项目', implementing: '正在完成修改',
      checking: '正在运行实际检查', reviewing: '正在独立审查', blocked: '本批已暂停，需要处理',
      interrupted: '运行中断，已有记录保留', exhausted: '本批自动尝试已结束，仍有问题待处理',
      cancelled: '本批已停止，已有记录保留', cancelling: '正在停止本批任务',
    };
    return [labels[task.state] || '本批结果尚未核验完整', ['blocked', 'exhausted', 'interrupted'].includes(task.state) ? 'bad' : ''];
  }
  function resultSummary(parent, task) {
    const section = el('section', 'result-summary'), [label, tone] = resultStatus(task);
    section.append(el('h3', 'result-status', label));
    if (tone) section.classList.add(tone);
    section.append(el('p', 'result-scope-note', '这里说明本批修改的核验情况。整个项目是否达到你的预期，还需要结合审查意见和实际效果验收。'));
    if ((task.requirement_revision || 0) !== (task.audited_revision || 0)) section.append(el('p', 'hint', '需求已有更新，已有检查和审查记录不能代替对新要求的重新核验。'));
    if (task.reason && ['blocked', 'interrupted', 'exhausted', 'cancelled'].includes(task.state)) section.append(el('p', '', friendly(task.reason)));
    const meta = el('p', 'result-provenance'), source = task.snapshot || '';
    meta.append(el('span', '', task.round > 0 ? '记录范围：本批第 ' + task.round + ' 轮' : '尚未开始实施修改'));
    if (source) { const fingerprint = el('code', '', source.slice(0, 16)); fingerprint.title = source; meta.append(el('span', '', '核验源码指纹：'), fingerprint); }
    else meta.append(el('span', '', '本轮尚未保存完整的源码核验指纹'));
    section.append(meta); parent.append(section);
  }
  function checksReport(parent, task) {
    const section = el('section', 'detail-section'); section.append(el('h3', '', '实际检查记录'));
    if (!task.checks?.length) section.append(el('p', 'muted', task.auto_checks ? '项目的检查方式尚未运行并形成结果。运行后，可逐项查看原始日志。' : '还没有实际检查结果。未运行检查不能视为已通过。'));
    else {
      const list = el('div', 'checks');
      for (const check of task.checks) {
        const row = el('div', 'check'), content = el('div', 'check-content'), passed = check.exit_code === 0 && !check.verification_problem;
        content.append(el('code', '', check.command));
        if (check.verification_problem) content.append(el('p', 'hint', '本项未完成有效验证：' + check.verification_problem));
        if (task.round > 0 && /^check-\d+\.log$/.test(check.log || '')) content.append(evidenceLink(task, '打开本项日志', 'round-' + task.round + '/' + check.log));
        else content.append(el('small', 'muted', '本项没有可用的日志路径'));
        row.append(content, el('strong', passed ? 'check-status' : 'check-status fail', check.verification_problem ? '未有效验证' : passed ? '通过' : '失败（退出码 ' + check.exit_code + '）')); list.append(row);
      }
      section.append(list);
    }
    parent.append(section);
  }
  function reviewReport(parent, task) {
    const section = el('section', 'detail-section result-limitations'); section.append(el('h3', '', '独立审查意见'));
    if (task.review) {
      const verdict = ({ pass: '本轮审查结论：通过', rework: '本轮审查结论：需要返工', blocked: '本轮审查结论：证据不足或存在阻碍' })[task.review.verdict] || '本轮审查尚无有效结论';
      section.append(el('p', 'hint', verdict), el('p', 'review-summary', task.review.summary || '尚未保存可读的审查摘要。'));
      if (task.review.findings?.length) {
        section.append(el('h3', '', '验收前仍需留意'));
        const findings = el('ul', 'result-findings');
        for (const finding of task.review.findings) {
          const item = el('li', 'result-finding');
          item.append(el('strong', '', finding.severity + ' · ' + (finding.file || '未标注文件')), el('p', '', finding.reason)); findings.append(item);
        }
        section.append(findings);
      } else section.append(el('p', 'muted', '这份审查报告没有列出遗留问题。请结合本批目标检查实际效果。'));
      const run = [...(task.runs || [])].reverse().find(item => item.role === 'reviewer' && item.round === task.round && item.status === 'completed');
      if (run) section.append(el('p', 'result-provenance', '审查执行：' + (run.provider_name || run.provider_id) + ' · ' + run.model + ' · 第 ' + run.round + ' 轮'));
    } else section.append(el('p', 'muted', '还没有独立审查结论。检查完成后，由新会话核对源码、需求和原始日志。'));
    parent.append(section);
  }
  function runHistory(parent, task) {
    const section = el('section', 'detail-section'); section.append(el('h3', '', '实际交接记录'));
    const list = el('ul', 'run-list');
    for (const run of task.runs || []) {
      const row = el('li'), who = el('span', '', (run.provider_name || run.provider_id || '未记录工具') + ' · ' + (run.model || '沿用原工具设置'));
      who.append(el('small', '', run.session ? '会话：' + run.session : '当前会话尚未完成'), el('small', '', '源码指纹：' + (run.source_hash || '').slice(0, 12) + '；安排版本：' + run.plan_revision));
      if (run.effort) who.append(el('small', '', '思考：' + (PlanUI.effortNames[run.effort] || run.effort)));
      row.append(el('strong', '', roleTitle(run.role, run.domain) + ' · 第 ' + run.round + ' 轮'), who, el('span', '', ({ completed: '已完成', running: '执行中', preparing: '准备交接', interrupted: '已中断', not_started: '未派发', failed: '失败' })[run.status] || run.status)); list.append(row);
    }
    if (!task.runs?.length) section.append(el('p', 'muted', '实际派发后，会在这里保留交接记录。'));
    section.append(list); parent.append(section);
  }
  function evidenceFiles(parent, task, epoch, quick) {
    const details = el('details', 'files-disclosure'), files = el('div', 'files'); details.dataset.disclosure = 'files'; details.append(el('summary', '', '全部原始证据文件'), files); parent.append(details);
    files.append(el('p', 'hint', '正在读取证据文件列表…'));
    api('tasks/' + task.id + '/evidence').then(paths => {
      if (epoch !== generation || selected !== task.id) return;
      files.replaceChildren();
      for (const path of paths) files.append(evidenceLink(task, path, path));
      if (!paths.length) files.append(el('p', 'hint', '尚无证据文件，执行后会逐步保存。'));
      if (!quick) return;
      quick.replaceChildren();
      const prefix = 'round-' + task.round;
      for (const [label, path] of [['打开代码修改（补丁文件）', prefix + '/changes.patch'], ['打开独立审查原文（JSON）', prefix + '-review.json']]) {
        if (task.round > 0 && paths.includes(path)) quick.append(evidenceLink(task, label, path));
      }
      if (!quick.children.length) quick.append(el('p', 'hint', '本轮尚未生成可打开的代码补丁或审查文件。'));
    }).catch(err => {
      if (epoch !== generation || selected !== task.id) return;
      files.replaceChildren(el('p', 'hint', '证据列表未能读取：' + friendly(err.message)));
      if (quick) quick.replaceChildren(el('p', 'hint', '交付文件列表暂时无法读取，已有检查与审查记录仍保留。'));
    });
  }
  function render(task) {
    const root = $('task-proof');
    if (previousTask) disclosures.set(previousTask, Object.fromEntries([...root.querySelectorAll('details[data-disclosure]')].map(item => [item.dataset.disclosure, item.open])));
    previousTask = task.id; const saved = disclosures.get(task.id) || {}, epoch = ++generation;
    root.replaceChildren(); const panel = el('article', 'surface result-panel');
    if (task.read_only) {
      panel.append(el('h2', '', '项目调查报告'), el('p', 'result-scope-note', '这是基于已读文件的模型调查，没有执行代码修改、实际测试或独立验收。'));
      if (task.audit) {
        panel.append(el('p', 'review-summary', task.audit.summary)); auditReport(panel, task.audit, false);
      } else panel.append(el('p', 'muted', ['cancelled', 'interrupted', 'blocked'].includes(task.state) ? '调查尚未形成完整报告。已有执行记录和证据保留在下方。' : '调查完成后，这里会显示项目用途、当前进展、事实与待确认项。'));
      const evidence = disclosure(panel, 'evidence', '证据与交接记录'); runHistory(evidence, task); evidenceFiles(evidence, task, epoch);
    } else {
      const heading = el('div', 'result-heading'); heading.append(el('h2', '', '本批结果'), el('span', 'badge', task.round ? '第 ' + task.round + ' 轮' : '尚未开始实施')); panel.append(heading);
      resultSummary(panel, task); ResultArtifacts.mount(panel, task); reviewReport(panel, task);
      const quick = el('div', 'proof-links'); quick.append(el('p', 'hint', '正在读取本轮交付文件…')); panel.append(quick);
      checksReport(panel, task); auditReport(panel, task.audit);
      const evidence = disclosure(panel, 'evidence', '证据与交接记录'); runHistory(evidence, task); evidenceFiles(evidence, task, epoch, quick);
    }
    root.append(panel);
    for (const details of root.querySelectorAll('details[data-disclosure]')) details.open = Object.prototype.hasOwnProperty.call(saved, details.dataset.disclosure) ? !!saved[details.dataset.disclosure] : task.read_only && details.dataset.disclosure === 'audit';
  }
  return { render };
})();
