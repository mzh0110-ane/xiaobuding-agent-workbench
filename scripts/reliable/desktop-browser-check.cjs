const fs = require('node:fs');
const http = require('node:http');
const assert = require('node:assert/strict');
const { createRequire } = require('node:module');
const path = require('node:path');
const root = process.env.XIAOBUDING_SOURCE_ROOT || path.resolve(__dirname, '../..');
const { chromium } = createRequire(path.join(root, 'apps/web/package.json'))('@playwright/test');

const source = root + '/apps/backend/internal/reliable/';
const out = process.env.XIAOBUDING_VALIDATION_DIR || path.join(root, '.tmp/reliable-desktop-browser'); fs.mkdirSync(out, { recursive: true });
const read = name => fs.existsSync(source + name) ? fs.readFileSync(source + name, 'utf8') : '';
const contentPolicy = read('http.go').match(/Set\("Content-Security-Policy", "([^"]+)"\)/)[1];
const embedSource = read('http.go');
const embeddedVariables = Object.fromEntries([...embedSource.matchAll(/\/\/go:embed (page[^\r\n]+)\r?\nvar (\w+) string/g)].map(match => [match[2], match[1]]));
const embeddedScripts = [...embedSource.match(/"__SCRIPT__", (.*)\)\.Replace/)[1].matchAll(/\b(page\w*JS)\b/g)].map(match => embeddedVariables[match[1]]);
const html = read('page.html').replace('__TOKEN__', 'desktop-fixture').replace('__STYLE__', read('page.css')).replace('__SCRIPT__', embeddedScripts.map(read).join('\n'));
const plan = { mode: 'fullstack', frontend: 'deepseek', backend: 'codex', auditor: 'codex', reviewer: 'codex', implementer: 'codex' };
const providers = ['codex', 'deepseek'].map(id => ({ id, name: id === 'codex' ? 'Codex（ChatGPT 登录）' : 'DeepSeek Harness', available: true, model: id + '-default', models: [{ id: id + '-default', name: id === 'codex' ? 'GPT-6.1-Sol' : 'DeepSeek Flash', efforts: ['low', 'high'], default_effort: 'low' }] }));
let rows, calls, actionFailure, pickerResult, activity, messageGate, uploaded, uploadFailure, uploadGate, projects, projectReads, storageReads;
function reset() { projects = new Map(); projectReads = 0; storageReads = 0; uploaded = new Map(); uploadFailure = false; uploadGate = null; calls = []; actionFailure = false; pickerResult = { path: '' }; activity = []; messageGate = null; rows = [
  { id: 'a', goal: '给记账页面增加月份筛选', project: 'E:\\记账项目', worktree: 'E:\\副本\\a', state: 'implementing', round: 1, model_plan: { ...plan }, plan_revision: 1, updated: new Date().toISOString(), checks: [], runs: [{ id: 'running', role: 'implementer', domain: 'frontend', status: 'running', provider_name: 'DeepSeek Harness', model: 'deepseek-default', source_hash: 'abc123', plan_revision: 1 }] },
  { id: 'b', goal: '整理导出功能', project: 'E:\\数据项目', worktree: 'E:\\副本\\b', state: 'awaiting_acceptance', round: 2, model_plan: { ...plan }, plan_revision: 1, updated: new Date().toISOString(), checks: [{ command: 'npm run test', exit_code: 0, log: 'check-1.log' }], review: { verdict: 'pass', summary: '核对了实际源码和测试', findings: [] } },
  { id: 'c', goal: '已完成的示例', project: 'E:\\样例', state: 'accepted', round: 1, model_plan: { ...plan }, plan_revision: 1, archived_at: '2026-10-19T00:00:00Z', updated: new Date().toISOString() },
]; } reset();
function projectFor(id) {
  if (!projects.has(id)) projects.set(id, { id: 'project-' + id, project: rows.find(row => row.id === id)?.project, title: '长期记账项目', revision: 1, auto_continue: false, items: [
    { id: 'planned', title: '添加年度汇总', status: 'planned', source: 'user', acceptance: '切换年份后统计与列表一致' },
    { id: 'verified', title: '月份筛选', status: 'verified', task_id: id, evidence: [{ task_id: id, path: 'round-1/check-1.log', label: '打开实际检查' }] },
    { id: 'suggested', title: '增加导出预览', status: 'planned', source: 'plan:project-plan-fixture', confirmed: false },
  ], memories: [{ id: 'memory', kind: 'constraint', text: '保留已有导出格式', source: 'user', source_task: id, verified: false }], batches: [{ id, state: 'implementing', round: 1, goal: '当前批次' }] });
  return projects.get(id);
}
const server = http.createServer(async (req, res) => {
  const path = req.url.split('?')[0];
  if (path === '/reliable/icon.png') { res.setHeader('Content-Type', 'image/png'); res.end(fs.readFileSync(source + 'assets/pudding-avatar.png')); return; }
  if (path === '/reliable/background.png') { res.setHeader('Content-Type', 'image/png'); res.end(fs.readFileSync(source + 'assets/teahouse-background.png')); return; }
  if (path === '/reliable') { res.setHeader('Content-Type', 'text/html; charset=utf-8'); res.setHeader('Content-Security-Policy', contentPolicy); res.end(html); return; }
  if (path === '/api/v1/reliable/attachments' && req.method === 'POST') {
    const chunks = []; for await (const chunk of req) chunks.push(chunk);
    const form = await new Response(Buffer.concat(chunks), { headers: { 'Content-Type': req.headers['content-type'] } }).formData();
    const file = form.get('file'); calls.push({ action: 'upload', name: file.name, token: req.headers['x-xbd-token'] });
    if (uploadGate) await uploadGate;
    res.setHeader('Content-Type', 'application/json');
    if (uploadFailure) { res.statusCode = 409; res.end(JSON.stringify({ error: 'attachment_upload_failed: fixture upload failed' })); return; }
    const metadata = { id: 'attachment-' + (uploaded.size + 1), name: file.name, size: file.size, media_type: file.type || 'application/octet-stream', sha256: 'fixture-sha', created: new Date().toISOString(), image: ['image/png','image/jpeg','image/webp','image/gif'].includes(file.type) };
    uploaded.set(metadata.id, { metadata, bytes: Buffer.from(await file.arrayBuffer()) }); res.statusCode = 201; res.end(JSON.stringify(metadata)); return;
  }
  const attachmentPath = path.match(/^\/api\/v1\/reliable\/attachments\/([^/]+)(\/preview)?$/);
  if (attachmentPath) { const file = uploaded.get(decodeURIComponent(attachmentPath[1])); if (!file) { res.statusCode = 404; res.end(); return; } res.setHeader('Content-Type', file.metadata.media_type); res.end(file.bytes); return; }
  let raw = ''; for await (const chunk of req) raw += chunk;
  const body = raw ? JSON.parse(raw) : {};
  const reply = data => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(data)); };
  if (path.endsWith('/config')) return reply({ version: '1.4.0', providers, default_plan: plan, demo_project: 'fixture' });
  if (path.endsWith('/storage')) { storageReads++; return reply({ root: 'E:\\小布丁\\data\\reliable', temp_root: 'E:\\小布丁\\data\\temp', total_bytes: 4096, entries: [{ name: '项目记忆与任务库', bytes: 4096, files: 1 }] }); }
  if (path.endsWith('/storage/cleanup')) { calls.push({ action: 'storage-cleanup', body }); return reply({ dry_run: body.dry_run, minimum_age_days: 7, files: 2, bytes: 2048, removed_files: body.dry_run ? 0 : 2, removed_bytes: body.dry_run ? 0 : 2048, skipped_active: 1 }); }
  const projectPath = path.match(/^\/api\/v1\/reliable\/tasks\/([^/]+)\/project(?:\/(plan|start))?$/);
  if (projectPath) {
    const id = projectPath[1], action = projectPath[2], record = projectFor(id);
    if (req.method === 'GET') { projectReads++; return reply(record); }
    calls.push({ id, action: action ? 'project-' + action : 'project-save', body });
    if (actionFailure) { res.statusCode = 409; return reply({ error: 'project_revision_conflict: fixture conflict' }); }
    if (action === 'plan') return reply({ id: 'project-planning', kind: 'question', status: 'queued' });
    if (action === 'start') {
      const parent = rows.find(row => row.id === id); rows.unshift({ ...parent, id: 'planned-next', parent_task_id: id, goal: '下一项项目待办', state: 'queued', round: 0, messages: [] });
      return reply({ id: 'project-start', kind: 'requirement', status: 'linked', linked_task: 'planned-next' });
    }
    for (const item of body.items || []) record.items.push({ ...item, id: 'manual-' + record.items.length, status: 'planned', source: 'user' });
    if (body.memory) record.memories.push({ ...body.memory, id: 'manual-memory', source: 'user', verified: false });
    if (body.auto_continue !== undefined) record.auto_continue = body.auto_continue;
    record.revision++; return reply(record);
  }
  if (path.endsWith('/tasks')) {
    if (req.method === 'POST') { calls.push({ action: 'create', body }); const row = { id:'new',attachments:(body.attachment_ids || []).map(id => uploaded.get(id).metadata),project:body.project,goal:body.goal,state:'auditing',round:0,read_only:body.read_only,model_plan:body.model_plan,updated:new Date().toISOString() }; rows.unshift(row); return reply(row); }
    return reply(rows);
  }
  if (path.endsWith('/activity')) { const after = Number(new URL(req.url, 'http://fixture').searchParams.get('after') || 0); const events = activity.filter(event => event.sequence > after); return reply({ events, cursor: events.at(-1)?.sequence || after, active: true, limited: false }); }
  if (path.endsWith('/pick-project')) {
    calls.push({ action: 'pick-project', body });
    if (pickerResult.error) res.statusCode = 409;
    return reply(pickerResult);
  }
  if (path.endsWith('/evidence')) {
    if (new URL(req.url, 'http://fixture').searchParams.get('path')) { res.setHeader('Content-Type', 'text/plain; charset=utf-8'); res.end('diff --git a/example.js b/example.js\n--- a/example.js\n+++ b/example.js\n@@ -1 +1 @@\n-old\n+<script>plain text</script>'); return; }
    const task = rows.find(item => item.id === path.split('/').at(-2)), round = task?.round || 1;
    return reply(['contract.json', 'round-' + round + '-review.json', 'round-' + round + '/changes.patch', ...(task?.checks || []).filter(check => check.log).map(check => 'round-' + round + '/' + check.log)]);
  }
  const detailPath = path.match(/^\/api\/v1\/reliable\/tasks\/([^/]+)$/);
  if (detailPath && req.method === 'GET') {
    const task = rows.find(row => row.id === detailPath[1]);
    if (!task) { res.statusCode = 404; return reply({ error: 'task not found' }); }
    const messages = task.messages || [];
    return reply({ ...task, messages: messages.slice(-25), messages_before: Math.max(0, messages.length - 25), messages_total: messages.length });
  }
  const messagePath = path.match(/^\/api\/v1\/reliable\/tasks\/([^/]+)\/messages$/);
  if (messagePath && req.method === 'GET') {
    const messages = rows.find(row => row.id === messagePath[1])?.messages || [];
    const before = Number(new URL(req.url, 'http://fixture').searchParams.get('before') || messages.length);
    return reply({ messages: messages.slice(Math.max(0, before - 25), before), before: Math.max(0, before - 25), total: messages.length });
  }
  const parts = path.split('/'); const action = parts.at(-1), id = parts.at(-2);
  const row = rows.find(item => item.id === id);
  if (row && req.method === 'POST') {
    calls.push({ id, action, body });
    if (actionFailure) { res.statusCode = 409; return reply({ error: typeof actionFailure === 'string' ? actionFailure : 'task_running: a new execution has started' }); }
    if (action === 'resume' && row.local_checkpoint) {
      if (messageGate) await messageGate;
      row.local_checkpoint = { ...row.local_checkpoint, complete: true, commit: row.local_checkpoint.commit || '1234567890abcdef1234567890abcdef12345678' };
      row.state = 'accepted'; delete row.reason;
      return reply(row);
    }
    if (action === 'resume' && row.stage_recovery) { row.state = 'implementing'; row.stage_recovery.status = 'running'; return reply(row); }
    if (action === 'recover-cancelled') { row.state = 'queued'; row.recovery_pending = { attempt: 'fixture-recovery', round: row.round, phase: 'validating' }; res.statusCode = 202; return reply(row); }
    if (action === 'resume' && row.recovery_pending) { row.state = 'queued'; res.statusCode = 202; return reply(row); }
    if (action === 'future-models') { row.future_plan = { plan: body.plan, revision: (row.future_plan?.revision || 0) + 1 }; return reply(row); }
    if (action === 'models') { row.model_plan = body.plan; row.plan_revision++; return reply(row); }
    if (action === 'exhausted-next-batch') {
      if (messageGate) await messageGate;
      const existing = (row.messages || []).find(message => message.id === body.id);
      if (existing) { res.statusCode = 202; return reply(existing); }
      const message = { ...body, status: 'linked', linked_task: 'retained-next', created: new Date().toISOString() };
      row.messages ||= []; row.messages.push(message); row.successor_task_id = message.linked_task;
      rows.unshift({ ...row, id: message.linked_task, parent_task_id: row.id, successor_task_id: undefined, goal: body.content || row.goal, state: 'queued', round: 0, seed_pending: { parent_task_id: row.id }, worktree: 'E:\\副本\\retained-next', messages: [], runs: [], checks: [], review: undefined });
      res.statusCode = 202; return reply(message);
    }
    if (action === 'messages') {
      row.messages ||= [];
      const message = { ...body, attachments: (body.attachment_ids || []).map(id => uploaded.get(id).metadata), status: 'queued', created: new Date().toISOString() };
      if (body.kind === 'requirement' && (row.read_only && row.state === 'inspected' || row.state === 'accepted')) {
        message.status = 'linked'; message.linked_task = 'next-batch';
        rows.unshift({...row, id:'next-batch', parent_task_id:row.id, read_only:false, goal:body.content, state:'auditing', round:0, local_checkpoint:undefined, messages:[]});
      } else if (body.kind === 'requirement') row.state = 'auditing';
      row.messages.push(message); if (messageGate) await messageGate;
      return reply(message);
    }
    if (action === 'archive') row.archived_at = new Date().toISOString();
    if (action === 'unarchive') delete row.archived_at;
    return reply(row);
  }
  res.statusCode = 404; reply({ error: 'fixture route missing' });
});
(async () => {
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const base = 'http://127.0.0.1:' + server.address().port;
  const browser = await chromium.launch({ channel: 'msedge', headless: true }); const results = [];
  async function scenario(name, run, hash = '', mobile = false) {
    if (process.env.XIAOBUDING_SCENARIO_FILTER && !new RegExp(process.env.XIAOBUDING_SCENARIO_FILTER).test(name)) return;
    reset(); const context = await browser.newContext({ viewport: mobile ? { width: 390, height: 844 } : { width: 1440, height: 1050 } });
    try {
      const page = await context.newPage(); page.setDefaultTimeout(3500); const errors = [];
      page.on('pageerror', err => errors.push(err.message)); await page.goto(base + '/reliable' + hash);
      await page.locator('[data-task-row="a"]').waitFor({ state: 'attached' }); await run(page); assert.deepEqual(errors, []); results.push({ name, passed: true });
    } catch (err) { results.push({ name, passed: false, error: err.message }); }
    finally { await context.close(); }
  }
  async function settings(page, role) {
    if (!await page.locator('#agent-settings-drawer').isVisible()) await page.locator('#agent-settings-toggle').click();
    if (role) await page.locator('[data-agent-role="'+role+'"]').click();
  }
  async function tab(page, name) {
    await page.locator('#tab-' + name).click();
    await page.locator('#panel-' + name).waitFor({ state: 'visible' });
  }
  async function summaryRules(page) {
    const details = page.locator('#task-summary .summary-stages');
    if (!await details.evaluate(node => node.open)) await details.locator('summary').click();
    await details.locator('.summary-status-details').waitFor({ state: 'visible' });
    return details.locator('.summary-status-details').innerText();
  }
  function separated(a, b) { return a.x + a.width <= b.x + 1 || b.x + b.width <= a.x + 1 || a.y + a.height <= b.y + 1 || b.y + b.height <= a.y + 1; }
  async function composerInViewport(page) {
    const geometry = await page.locator('#workspace-composer').evaluate(node => {
      const bounds = node.getBoundingClientRect(), input = node.querySelector('#followup-content').getBoundingClientRect(), send = node.querySelector('#followup-send').getBoundingClientRect();
      return { top: bounds.top, bottom: bounds.bottom, left: bounds.left, right: bounds.right, inputTop: input.top, inputBottom: input.bottom, sendBottom: send.bottom, width: innerWidth, height: innerHeight };
    });
    assert.ok(geometry.left >= 0 && geometry.right <= geometry.width + 1, 'composer extends beyond the viewport');
    assert.ok(geometry.top >= 0 && geometry.bottom <= geometry.height + 1, 'composer requires page scrolling');
    assert.ok(geometry.inputTop >= geometry.top && geometry.inputBottom <= geometry.height && geometry.sendBottom <= geometry.height, 'input or send button is outside the visible composer');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  }
  try {
    await scenario('workspace tabs expose one panel and preserve a usable composer', async page => {
      for (const name of ['conversation', 'activity', 'results', 'project']) {
        await tab(page, name);
        assert.equal(await page.locator('#workspace-tabs [role="tab"][aria-selected="true"]').count(), 1);
        assert.equal(await page.locator('#workspace-tabs [role="tab"][aria-selected="true"]').getAttribute('id'), 'tab-' + name);
        assert.equal(await page.locator('#workspace [role="tabpanel"]:visible').count(), 1);
        assert.equal(await page.locator('#workspace-composer #followup-form').isVisible(), true);
        assert.equal(await page.locator('#followup-content').isEditable(), true);
      }
      assert.equal(calls.length, 0, 'changing workspace views must not start work or save model choices');
    }, '#a');
    await scenario('project overview reads lazily and distinguishes verified items from unverified memory', async page => {
      assert.equal(projectReads, 0); assert.equal(storageReads, 0);
      await tab(page, 'project'); await page.locator('.project-items .project-item').first().waitFor();
      assert.match(await page.locator('.project-heading').innerText(), /核验 1\/3 项 · 已验收 0 项/);
      assert.equal(await page.locator('.project-item-links a[href*="round-1%2Fcheck-1.log"]').count(), 1);
      await page.locator('.project-disclosure > summary').filter({ hasText: '项目记忆' }).click();
      assert.match(await page.locator('.project-memories').innerText(), /保留已有导出格式[\s\S]*你补充的记录 · 尚未独立核验/);
      assert.equal(await page.locator('.project-items input[type=checkbox]').count(), 0, 'manual checks cannot claim automatic verification');
      assert.equal(storageReads, 0); assert.equal(calls.length, 0);
      await page.screenshot({ path: out + '/project-overview-desktop.png', animations: 'disabled' });
    }, '#a');
    await scenario('project manual requirements and memories retain intent without declaring verification', async page => {
      await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      await page.locator('.project-disclosure > summary').filter({ hasText: '添加项目待办' }).click();
      await page.locator('#project-title').fill('支持季度汇总'); await page.locator('#project-acceptance').fill('图表和列表合计一致');
      await page.getByRole('button', { name: '保存待办', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('project-status').textContent.includes('待办已保存'));
      const saved = calls.find(call => call.action === 'project-save'); assert.equal(saved.body.expected_revision, 1);
      assert.deepEqual(saved.body.items, [{ title: '支持季度汇总', acceptance: '图表和列表合计一致' }]);
      assert.equal(saved.body.items[0].status, undefined); assert.equal(rows[0].round, 1);
      await page.locator('.project-disclosure > summary').filter({ hasText: '项目记忆' }).click();
      await page.locator('#project-memory-kind').selectOption('constraint'); await page.locator('#project-text').fill('保留现有导出按钮');
      await page.getByRole('button', { name: '保存这条记忆', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('project-status').textContent.includes('记忆已保存'));
      const memory = calls.filter(call => call.action === 'project-save').at(-1); assert.deepEqual(memory.body.memory, { kind: 'constraint', text: '保留现有导出按钮' });
      assert.equal(memory.body.memory.verified, undefined); assert.match(await page.locator('#task-project').innerText(), /核验 1\/4 项/);
    }, '#a');
    await scenario('project revision conflicts preserve drafts and refresh does not erase their text', async page => {
      await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      await page.locator('.project-disclosure > summary').filter({ hasText: '添加项目待办' }).click();
      await page.locator('#project-title').fill('我的长期项目要求'); await page.locator('#project-acceptance').fill('保留这一条验收标准');
      actionFailure = true; await page.getByRole('button', { name: '保存待办', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('project-status').dataset.error === 'true');
      assert.equal(await page.locator('#project-title').inputValue(), '我的长期项目要求');
      actionFailure = false; await page.getByRole('button', { name: '刷新项目记录', exact: true }).click();
      assert.equal(await page.locator('#project-title').inputValue(), '我的长期项目要求'); assert.equal(await page.locator('#project-acceptance').inputValue(), '保留这一条验收标准');
      await page.getByRole('button', { name: '保存待办', exact: true }).click(); await page.waitForFunction(() => document.getElementById('project-status').textContent.includes('待办已保存'));
    }, '#a');
    await scenario('project native planning starts only after an explicit request and does not consume rounds', async page => {
      await tab(page, 'project'); await page.locator('.project-heading').waitFor(); assert.equal(calls.length, 0);
      await page.getByRole('button', { name: '整理项目计划', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('tab-conversation').getAttribute('aria-selected') === 'true');
      assert.deepEqual(calls.map(call => call.action), ['project-plan']); assert.equal(rows[0].round, 1);
      await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      await page.getByLabel('验收后自动开始下一项待办', { exact: true }).check();
      await page.waitForFunction(() => document.getElementById('project-status').textContent.includes('已开启'));
      assert.equal(calls.at(-1).body.auto_continue, true); assert.equal(rows[0].state, 'implementing');
    }, '#a');
    await scenario('project confirmed planned items can start the next batch after acceptance', async page => {
      rows[1].state = 'accepted'; await page.reload(); await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      await page.getByRole('button', { name: '开始这项', exact: true }).click();
      await page.waitForURL('**#planned-next'); assert.equal(calls.find(call => call.action === 'project-start').body.item_id, 'planned');
      assert.equal(rows.find(row => row.id === 'planned-next').round, 0);
    }, '#b');
    await scenario('project diff is fetched on demand and renders code as text', async page => {
      await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      assert.equal(await page.locator('.project-patch').count(), 0);
      await page.locator('.project-diff > summary').click();
      await page.locator('.project-patch .patch-added').waitFor();
      assert.match(await page.locator('.project-patch').innerText(), /<script>plain text<\/script>/);
      assert.equal(await page.locator('.project-patch script').count(), 0); assert.equal(await page.locator('.project-patch .patch-removed').count(), 1);
      await page.locator('#theme').selectOption('dark'); await page.locator('.project-patch').scrollIntoViewIfNeeded();
      assert.equal(await page.locator('.patch-removed').evaluate(node => { const expected = getComputedStyle(document.documentElement).getPropertyValue('--danger').trim(); const probe = document.createElement('span'); probe.style.color = expected; document.body.append(probe); const matches = getComputedStyle(probe).color === getComputedStyle(node).color; probe.remove(); return matches; }), true, 'removed diff lines should use the red danger token');
      await page.screenshot({ path: out + '/project-diff-desktop-dark.png', animations: 'disabled' });
      await page.setViewportSize({ width: 390, height: 844 }); await page.locator('.project-patch').scrollIntoViewIfNeeded(); assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/project-diff-mobile-dark.png', animations: 'disabled' }); await page.setViewportSize({ width: 1440, height: 1050 });
      await page.getByRole('button', { name: '打开修改后的工作副本', exact: true }).click(); assert.equal(calls.at(-1).action, 'open');
    }, '#a');
    await scenario('storage checks stay explicit and cleanup requires dry-run then a separate confirmation', async page => {
      await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      await page.locator('.project-disclosure > summary').filter({ hasText: '空间与临时文件' }).click();
      assert.equal(storageReads, 0); assert.equal(calls.length, 0);
      await page.getByRole('button', { name: '查看空间与可清理项', exact: true }).click();
      await page.locator('.project-storage-info').getByText('数据位置：', { exact: false }).waitFor(); assert.equal(storageReads, 1);
      assert.match(await page.locator('.project-storage-info').innerText(), /E:\\小布丁\\data/);
      assert.match(await page.locator('.project-storage-info').innerText(), /协作数据分类占用（已扫描）[\s\S]*统计范围/);
      await page.getByRole('button', { name: '检查可清理的临时文件', exact: true }).click();
      await page.getByRole('button', { name: '确认清理这些临时文件', exact: true }).waitFor();
      assert.deepEqual(calls.filter(call => call.action === 'storage-cleanup').map(call => call.body.dry_run), [true]);
      await page.getByRole('button', { name: '确认清理这些临时文件', exact: true }).click();
      await page.waitForFunction(() => document.querySelector('.project-cleanup').textContent.includes('清理完成'));
      assert.deepEqual(calls.filter(call => call.action === 'storage-cleanup').map(call => call.body.dry_run), [true, false]); assert.equal(rows[0].state, 'implementing');
    }, '#a');
    await scenario('project tab remains usable on a phone without horizontal overflow', async page => {
      rows[0].stage_recovery = { round: 1, domain: 'backend', attempts: 2, manual_resumes: 1, status: 'retryable', last_error: 'upstream timeout' };
      await page.reload(); await tab(page, 'project'); await page.locator('.project-heading').waitFor();
      assert.match(await page.locator('.project-recovery').innerText(), /后端功能[\s\S]*累计执行 2 次[\s\S]*恢复与代码返工轮次分别记录/);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false); await composerInViewport(page);
      const tabs = await page.locator('#workspace-tabs').boundingBox(); assert.ok(tabs.x >= 0 && tabs.x + tabs.width <= 390);
      await page.screenshot({ path: out + '/project-mobile.png', animations: 'disabled' });
    }, '#a', true);
    await scenario('unfinished writer can resume at three rounds while next-batch renewal remains explicit', async page => {
      Object.assign(rows[0], { state: 'blocked', round: 3, base: 'base', baseline: { hash: 'baseline' }, audit: { summary: 'A frozen audit' }, requirement_revision: 1, audited_revision: 1, runs: [{ role: 'implementer', domain: 'backend', status: 'failed', round: 3 }], stage_recovery: { round: 3, domain: 'backend', attempts: 3, manual_resumes: 0, status: 'retryable' } });
      await page.reload();
      const resume = page.getByRole('button', { name: '继续未完成阶段', exact: true });
      await resume.waitFor({ state: 'visible' }); await page.locator('#begin-next-batch').waitFor({ state: 'visible' });
      assert.equal(await resume.isVisible(), true, 'the unfinished writer resume action must remain visible at round three'); assert.equal(await page.locator('#begin-next-batch').isVisible(), true, 'the explicit renewal action must remain available');
      await resume.click(); await page.waitForFunction(() => document.querySelector('#task-summary .badge').textContent === '实施修改中');
      assert.equal(calls.at(-1).action, 'resume'); assert.equal(calls.at(-1).id, 'a'); assert.equal(rows[0].round, 3); assert.equal(rows.length, 3);
    }, '#a');
    for (const mobile of [false, true]) {
      await scenario('workspace keeps long requirements readable and the composer visible on ' + (mobile ? 'phone' : 'desktop'), async page => {
        const fullGoal = '继续完善题目网站的交互分析功能，保留原来的图片上传、计算与导出。'.repeat(60) + '\n\nNEXT BATCH REQUIREMENT:\n修复上一批独立审查问题。';
        rows[0].goal = fullGoal;
        rows[0].messages = Array.from({ length: 25 }, (_, i) => ({ id: 'history-' + i, kind: 'question', status: 'answered', content: '第 ' + i + ' 次讨论：' + '请核对项目依据。'.repeat(5), reply: { summary: '已依据本地文件记录当前进展。'.repeat(6), unknowns: ['尚未验证网络服务。'], read_files: ['README.md'] } }));
        await page.reload();
        const title = await page.locator('#task-summary h1').innerText();
        assert.ok(title.length <= 100, 'the entire requirement is still used as the workspace title');
        assert.doesNotMatch(title, /NEXT BATCH REQUIREMENT/);
        const requirement = page.locator('details.task-requirement');
        assert.equal(await requirement.evaluate(node => node.open), false, 'full requirement should start collapsed');
        await requirement.locator('summary').click(); assert.ok((await requirement.innerText()).includes(fullGoal.replace(/NEXT BATCH REQUIREMENT:\s*/g, '下一批要求：\n')), 'disclosure lost part of the requirement while translating the internal batch marker');
        await requirement.locator('summary').click();
        for (const name of ['conversation', 'activity', 'results']) {
          await tab(page, name); await composerInViewport(page);
          assert.equal(await page.locator('#followup-content').isEditable(), true);
        }
        await page.screenshot({ path: out + '/workspace-' + (mobile ? 'mobile' : 'desktop') + '.png', animations: 'disabled' });
        assert.equal(calls.length, 0);
      }, '#a', mobile);
    }
    await scenario('workspace tabs use keyboard navigation without stealing composer focus during refresh', async page => {
      const selectedTab = async () => page.locator('#workspace-tabs [aria-selected="true"]').getAttribute('id');
      await page.locator('#tab-conversation').focus(); await page.keyboard.press('ArrowRight');
      assert.equal(await selectedTab(), 'tab-activity');
      assert.equal(await page.locator('#tab-activity').evaluate(node => node === document.activeElement), true);
      await page.keyboard.press('End'); assert.equal(await selectedTab(), 'tab-project');
      await page.keyboard.press('ArrowRight'); assert.equal(await selectedTab(), 'tab-conversation');
      await page.keyboard.press('ArrowLeft'); assert.equal(await selectedTab(), 'tab-project');
      await page.keyboard.press('Home'); assert.equal(await selectedTab(), 'tab-conversation');
      assert.equal(await page.locator('#workspace-tabs [role="tab"][tabindex="0"]').count(), 1);
      for (const name of ['conversation', 'activity', 'results', 'project']) {
        assert.equal(await page.locator('#tab-' + name).getAttribute('aria-controls'), 'panel-' + name);
        assert.equal(await page.locator('#panel-' + name).getAttribute('aria-labelledby'), 'tab-' + name);
      }
      await page.locator('#followup-content').fill('刷新和切换时保留我的草稿');
      await page.locator('#followup-content').focus(); rows[0].updated = new Date().toISOString();
      const refreshed = page.waitForResponse(response => response.url().endsWith('/tasks') && response.request().method() === 'GET');
      await page.evaluate(() => document.getElementById('refresh').click());
      await (await refreshed).finished(); await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      await page.waitForFunction(() => document.activeElement?.id === 'followup-content');
      assert.equal(await page.locator('#followup-content').inputValue(), '刷新和切换时保留我的草稿');
      assert.equal(calls.length, 0);
    }, '#a');
    await scenario('workspace tab choices and uploaded drafts stay with each selected project', async page => {
      await page.locator('#followup-content').fill('项目 A 的草稿');
      await page.locator('#followup-attachments-file').setInputFiles({ name: 'project-a.txt', mimeType: 'text/plain', buffer: Buffer.from('project A attachment') });
      await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
      await tab(page, 'activity'); await tab(page, 'results');
      assert.equal(await page.locator('#followup-content').inputValue(), '项目 A 的草稿');
      assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(), 1);
      await page.locator('[data-task-select="b"]').click();
      await page.locator('#followup-content').fill('项目 B 的草稿'); await tab(page, 'activity');
      assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(), 0);
      await page.locator('[data-task-select="a"]').click();
      assert.equal(await page.locator('#tab-results').getAttribute('aria-selected'), 'true');
      assert.equal(await page.locator('#followup-content').inputValue(), '项目 A 的草稿');
      assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(), 1);
      await page.reload();
      assert.equal(await page.locator('#tab-results').getAttribute('aria-selected'), 'true');
      assert.equal(await page.locator('#followup-content').inputValue(), '项目 A 的草稿');
      assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(), 1);
      await page.locator('[data-task-select="b"]').click();
      assert.equal(await page.locator('#tab-activity').getAttribute('aria-selected'), 'true');
      assert.equal(await page.locator('#followup-content').inputValue(), '项目 B 的草稿');
      assert.deepEqual(calls.map(call => call.action), ['upload']);
    }, '#a');
    await scenario('workspace width preference preserves the portrait choice and survives reload', async page => {
      const theme = await page.locator('#theme').inputValue(), background = await page.locator('#background-image').inputValue();
      const before = await page.locator('body').getAttribute('data-workspace-width');
      await page.locator('#workspace-width').click();
      const after = await page.locator('body').getAttribute('data-workspace-width');
      assert.ok(['wide', 'portrait'].includes(after)); assert.notEqual(after, before);
      assert.equal(await page.locator('#theme').inputValue(), theme); assert.equal(await page.locator('#background-image').inputValue(), background);
      await page.reload(); await page.waitForFunction(expected => document.body.dataset.workspaceWidth === expected, after);
      assert.equal(await page.locator('body').getAttribute('data-workspace-width'), after);
      await page.locator('#workspace-width').click(); assert.equal(await page.locator('body').getAttribute('data-workspace-width'), before);
      assert.equal(calls.length, 0);
    }, '#a');
    await scenario('workspace restores scroll positions separately for each tab and project', async page => {
      rows[0].messages = Array.from({ length: 35 }, (_, i) => ({ id: 'scroll-' + i, kind: 'question', status: 'answered', content: '历史问题 ' + i, reply: { summary: '按项目证据说明当前进展。'.repeat(12) } }));
      await page.reload();
      const scroll = page.locator('#workspace-content');
      const conversationScroll = await scroll.evaluate(node => { node.scrollTop = 400; node.dispatchEvent(new Event('scroll')); return node.scrollTop; });
      assert.ok(conversationScroll >= 300, 'conversation needs an independently scrollable history');
      await tab(page, 'results'); assert.equal(await scroll.evaluate(node => node.scrollTop), 0);
      await tab(page, 'conversation');
      await page.waitForFunction(expected => Math.abs(document.getElementById('workspace-content').scrollTop - expected) < 2, conversationScroll).catch(async () => { throw new Error('tab switch lost conversation scroll: expected ' + conversationScroll + ', actual ' + await scroll.evaluate(node => node.scrollTop)); });
      await page.locator('[data-task-select="b"]').click();
      await page.locator('[data-task-select="a"]').click();
      await page.waitForFunction(expected => Math.abs(document.getElementById('workspace-content').scrollTop - expected) < 2, conversationScroll).catch(async () => { throw new Error('project switch lost conversation scroll: expected ' + conversationScroll + ', actual ' + await scroll.evaluate(node => node.scrollTop)); });
      await composerInViewport(page); assert.equal(calls.length, 0);
    }, '#a');
    await scenario('workspace results expose every actual log and visible review limitations without implying full completion', async page => {
      Object.assign(rows[1], { requirement_revision: 1, audited_revision: 1, checks: [
        { command: 'npm test', exit_code: 0, log: 'check-1.log' },
        { command: 'npm run browser-check', exit_code: 0, verification_problem: 'no tests executed', log: 'check-2.log' },
        { command: 'npm run lint', exit_code: 2, log: 'check-3.log' },
      ], review: { verdict: 'pass', summary: '已核对本批导出修改；题图自动理解仍未实现。', findings: [] }, audit: { summary: '已读取导出模块', unknowns: ['历史调查时尚未运行浏览器检查'], plan: ['修改导出模块'], read_files: ['src/export.js'] } });
      await page.reload(); await tab(page, 'results');
      await page.locator('#task-proof .result-limitations').waitFor();
      assert.match(await page.locator('#task-proof .result-limitations').innerText(), /题图自动理解仍未实现/);
      assert.match(await page.locator('#task-proof .result-status').innerText(), /未|失败|完整|检查/);
      for (let index = 1; index <= 3; index++) {
        const log = page.locator('#task-proof .checks a[href*="' + encodeURIComponent('round-2/check-' + index + '.log') + '"]');
        assert.equal(await log.count(), 1, 'every actual check should have its own log'); assert.equal(await log.isVisible(), true);
      }
      assert.match(await page.locator('#task-proof .checks').innerText(), /未有效验证/);
      assert.match(await page.locator('#task-proof .checks').innerText(), /失败/);
      await page.locator('#task-proof .proof-links a[href*="' + encodeURIComponent('round-2/changes.patch') + '"]').waitFor();
      delete rows[1].review; await page.locator('#refresh').click();
      await page.locator('#task-proof .proof-links a[href*="' + encodeURIComponent('round-2/changes.patch') + '"]').waitFor();
      assert.equal(calls.length, 0, 'opening or refreshing results must not apply a candidate');
    }, '#b');
    await scenario('workspace verified results describe this batch and invalidate pass after requirements change', async page => {
      Object.assign(rows[1], { requirement_revision: 1, audited_revision: 1, snapshot: 'fixture-current-source-fingerprint', review: { verdict: 'pass', summary: '已核对本批导出修改与原始测试日志。', findings: [] } });
      await page.reload(); await tab(page, 'results');
      assert.match(await page.locator('#task-proof .result-status').innerText(), /本批核验已通过/);
      assert.match(await page.locator('#task-proof .result-scope-note').innerText(), /整个项目.*实际效果验收/);
      assert.match(await page.locator('#task-proof .result-provenance').innerText(), /第 2 轮.*fixture-current-/s);
      rows[1].requirement_revision = 2;
      const refreshed = page.waitForResponse(response => response.url().endsWith('/tasks') && response.request().method() === 'GET');
      await page.locator('#refresh').click(); await (await refreshed).finished();
      await page.waitForFunction(() => document.querySelector('#task-proof .result-summary').textContent.includes('需求已有更新'));
      assert.doesNotMatch(await page.locator('#task-proof .result-status').innerText(), /已通过/);
      assert.equal(calls.length, 0);
    }, '#b');
    await scenario('pending followup disables old failed-message retry and preserves the active attachment identity', async page => {
      rows[0].messages = [{ id: 'failed-question', kind: 'question', content: '旧的问题，不应覆盖正在发送的草稿', status: 'failed', error: 'fixture old answer failed' }];
      let release; messageGate = new Promise(resolve => { release = resolve; });
      try {
        await page.reload();
        await page.locator('#followup-attachments-file').setInputFiles({ name: 'current-question.txt', mimeType: 'text/plain', buffer: Buffer.from('current question attachment') });
        await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
        const attachmentID = await page.locator('#followup-attachments [data-attachment-id]').getAttribute('data-attachment-id');
        await page.locator('#followup-content').fill('当前正在发送的问题'); await page.locator('#followup-send').click();
        await page.waitForFunction(() => document.querySelector('#followup-send').disabled && document.querySelector('#followup-content').disabled);
        const retry = page.locator('[data-message-id="failed-question"]').getByRole('button', { name: '重新提问', exact: true });
        assert.equal(await retry.isDisabled(), true, 'retrying an old message must not replace the pending draft or its uploaded attachments');
        await retry.evaluate(node => node.click());
        assert.equal(await page.locator('#followup-content').inputValue(), '当前正在发送的问题');
        assert.equal(await page.locator('#followup-attachments [data-attachment-id]').getAttribute('data-attachment-id'), attachmentID);
        const completed = page.waitForResponse(response => response.url().endsWith('/tasks/a/messages') && response.request().method() === 'POST');
        release(); await (await completed).finished();
        await page.waitForFunction(() => !document.querySelector('#followup-content').disabled && document.querySelector('#followup-content').value === '');
        assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(), 0);
        assert.equal(calls.filter(call => call.action === 'messages').length, 1);
        assert.deepEqual(calls.find(call => call.action === 'messages').body.attachment_ids, [attachmentID]);
        assert.equal(await retry.isEnabled(), true, 'retry remains available after the earlier send has completed');
      } finally { release(); }
    }, '#a');
    // @covers AC-RELIABLE-ATTACHMENTS-001.1/.2/.4
    await scenario('attachments accept images and arbitrary files and are submitted with a new task', async page => {
      const picker = page.locator('#new-attachments-file'); await picker.waitFor({ state: 'attached' });
      assert.equal(await picker.getAttribute('accept'), null);
      const choosing = page.waitForEvent('filechooser'); await page.locator('#new-attachments').getByRole('button',{name:'添加附件',exact:true}).click();
      await (await choosing).setFiles([{ name:'layout.png',mimeType:'image/png',buffer:fs.readFileSync(source+'assets/pudding-avatar.png') },{ name:'project.custom-format',mimeType:'application/octet-stream',buffer:Buffer.from('custom source') }]);
      await page.waitForFunction(() => document.querySelectorAll('#new-attachments [data-upload-status="ready"]').length === 2);
      assert.equal(await page.locator('#new-attachments img').count(), 1);
      assert.equal(calls.filter(call => call.action === 'upload').length, 2); assert.ok(calls.every(call => call.token === 'desktop-fixture'));
      await page.locator('#project').fill('E:\\示例项目'); await page.locator('#goal').fill('参考附件完成页面'); await page.locator('#submit').click();
      await page.waitForFunction(() => location.hash === '#new');
      assert.equal(calls.find(call => call.action === 'create').body.attachment_ids.length, 2);
      await page.locator('#task-summary .summary-attachments > summary').click();
      await page.locator('#task-summary [data-attachment-id]').first().waitFor();
      assert.equal(await page.locator('#task-summary a[download]').count(), 2);
      const downloading = page.waitForEvent('download'); await page.locator('#task-summary a[download="project.custom-format"]').click(); assert.equal((await downloading).suggestedFilename(),'project.custom-format');
      await page.screenshot({ path:out+'/attachments-desktop.png', fullPage:true });
    });
    // @covers AC-RELIABLE-ATTACHMENTS-001.1/.2/.4
    await scenario('attachment upload failure keeps files for retry and removed uploads never return', async page => {
      uploadFailure = true; await page.locator('#new-attachments-file').setInputFiles({name:'notes.txt',mimeType:'text/plain',buffer:Buffer.from('reference')});
      await page.locator('#new-attachments [data-upload-status="failed"]').waitFor();
      assert.equal(await page.locator('#submit').isDisabled(), true); uploadFailure = false;
      await page.locator('#new-attachments').getByRole('button',{name:'重试上传 notes.txt',exact:true}).click();
      await page.locator('#new-attachments [data-upload-status="ready"]').waitFor();
      let release; uploadGate = new Promise(resolve => { release = resolve; });
      await page.locator('#new-attachments-file').setInputFiles({name:'remove.bin',mimeType:'application/octet-stream',buffer:Buffer.from('remove')});
      await page.locator('#new-attachments [data-upload-status="uploading"]').waitFor();
      await page.locator('#new-attachments').getByRole('button',{name:'移除附件 remove.bin',exact:true}).click(); release();
      await page.locator('#new-attachments [data-upload-status="ready"]').waitFor();
      assert.equal(await page.locator('#new-attachments [data-attachment-id]').count(), 1);
      assert.doesNotMatch(await page.locator('#new-attachments').innerText(), /remove.bin/);
    });
    // @covers AC-RELIABLE-ATTACHMENTS-001.1/.2/.4
    await scenario('followup attachment drafts stay with their project and failed sends retain uploaded files', async page => {
      await page.locator('#followup-attachments-file').setInputFiles({name:'followup.md',mimeType:'text/markdown',buffer:Buffer.from('project requirement')});
      await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
      await page.locator('#followup-content').fill('查看附件'); actionFailure = true; await page.locator('#followup-send').click();
      await page.locator('#followup-error').waitFor({state:'visible'}); assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(), 1);
      await page.locator('#sidebar-toggle').click(); await page.locator('[data-task-select="b"]').click(); assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(),0);
      await page.locator('#sidebar-toggle').click(); await page.locator('[data-task-select="a"]').click(); assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(),1);
      actionFailure = false; await page.locator('#followup-send').click(); await page.locator('#conversation-list [data-attachment-id]').waitFor();
      assert.equal(calls.find(call => call.action === 'messages' && call.body.attachment_ids?.length).body.attachment_ids.length,1);
      assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(),0);
      await page.locator('#sidebar-toggle').click(); await page.locator('[data-task-select="b"]').click(); await page.locator('#sidebar-toggle').click(); await page.locator('[data-task-select="a"]').click();
      assert.equal(await page.locator('#followup-content').inputValue(), '', 'a successful attachment send must not restore the old text draft');
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),false);
      await page.locator('#conversation-list [data-attachment-id]').scrollIntoViewIfNeeded(); await page.evaluate(() => Promise.all(document.getAnimations().filter(animation => Number.isFinite(animation.effect.getComputedTiming().iterations)).map(animation => animation.finished.catch(() => undefined))));
      await page.screenshot({path:out+'/attachments-mobile.png',fullPage:true});
    }, '#a', true);
    // @covers AC-RELIABLE-ATTACHMENTS-001.1/.2/.4
    await scenario('attachment-only requests are allowed and uploaded references survive a reload', async page => {
      await page.locator('#new-attachments-file').setInputFiles({name:'requirements.txt',mimeType:'text/plain',buffer:Buffer.from('create a small website')});
      await page.locator('#new-attachments [data-upload-status="ready"]').waitFor(); await page.reload();
      await page.locator('#new-attachments [data-upload-status="ready"]').waitFor();
      assert.equal(await page.locator('#goal').getAttribute('required'), null);
      await page.locator('#project').fill('E:\\附件项目'); await page.locator('#submit').click(); await page.waitForFunction(() => location.hash === '#new');
      const request = calls.find(call => call.action === 'create').body; assert.equal(request.goal, ''); assert.equal(request.attachment_ids.length, 1);
      assert.equal(calls.filter(call => call.action === 'upload').length, 1, 'reload must not upload the same file again');
    });
    // @covers AC-RELIABLE-ATTACHMENTS-001.1/.2/.4
    await scenario('attachment paste and drop preserve normal text paste and show arbitrary filenames as text', async page => {
      const goal = page.locator('#goal'); await goal.fill('保留文字');
      const prevented = await goal.evaluate(target => { const data = new DataTransfer(); data.setData('text/plain', 'normal paste'); const event = new ClipboardEvent('paste', {clipboardData:data,bubbles:true,cancelable:true}); target.dispatchEvent(event); return event.defaultPrevented; });
      assert.equal(prevented,false);
      await goal.evaluate(target => { const bytes = Uint8Array.from(atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg=='), c=>c.charCodeAt(0)); const data = new DataTransfer(); data.items.add(new File([bytes], 'pasted.png', {type:'image/png'})); target.dispatchEvent(new ClipboardEvent('paste',{clipboardData:data,bubbles:true,cancelable:true})); });
      await page.locator('#new-attachments [data-upload-status="ready"]').waitFor();
      await page.locator('.composer').evaluate(target => { const data = new DataTransfer(); data.items.add(new File(['unknown contents'], '<img onerror=alert(1)>.custom', {type:'application/octet-stream'})); target.dispatchEvent(new DragEvent('dragover',{dataTransfer:data,bubbles:true,cancelable:true})); target.dispatchEvent(new DragEvent('drop',{dataTransfer:data,bubbles:true,cancelable:true})); });
      await page.waitForFunction(() => document.querySelectorAll('#new-attachments [data-upload-status="ready"]').length === 2);
      assert.equal(await goal.inputValue(), '保留文字'); assert.equal(await page.locator('#new-attachments img').count(),1);
      assert.match(await page.locator('#new-attachments').innerText(), /<img onerror=alert\(1\)>.custom/);
      assert.equal(await page.locator('#new-attachments [onerror]').count(),0);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),false);
    });
    // @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
    await scenario('an in-flight attachment stays scoped when switching projects and enables attachment-only followup after completion', async page => {
      let release; uploadGate = new Promise(resolve => {release=resolve;});
      await page.locator('#followup-attachments-file').setInputFiles({name:'diagram.dat',mimeType:'application/octet-stream',buffer:Buffer.from('diagram')});
      await page.locator('#followup-attachments [data-upload-status="uploading"]').waitFor();
      assert.equal(await page.locator('#followup-send').isDisabled(),true);
      await page.locator('[data-task-select="b"]').click(); assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(),0);
      const completed = page.waitForResponse(response => response.url().endsWith('/attachments') && response.status() === 201); release(); await completed;
      assert.equal(new URL(page.url()).hash, '#b'); assert.equal(await page.locator('#followup-attachments [data-attachment-id]').count(),0);
      await page.locator('[data-task-select="a"]').click(); await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
      assert.equal(await page.locator('#followup-content').inputValue(),''); assert.equal(await page.locator('#followup-send').isEnabled(),true);
      await page.locator('#followup-send').click(); await page.locator('#conversation-list [data-attachment-id]').waitFor();
      const request = calls.find(call => call.action === 'messages').body; assert.equal(request.content,''); assert.equal(request.attachment_ids.length,1);
    }, '#a');
    // @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
    await scenario('exhausted renewal carries explicit requirement attachments', async page => {
      exhaustFixture(); await page.reload();
      await page.locator('#followup-attachments-file').setInputFiles({name:'review.txt',mimeType:'text/plain',buffer:Buffer.from('unresolved issue')});
      await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
      await page.locator('#followup-kind').selectOption('requirement');
      await page.locator('#task-summary').getByRole('button',{name:'保留修改，开始下一批',exact:true}).click(); await page.waitForFunction(()=>location.hash==='#retained-next');
      assert.equal(calls.find(call=>call.action==='exhausted-next-batch').body.attachment_ids.length,1);
    }, '#a');
    // @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
    await scenario('default exhausted renewal leaves a question draft and its uploaded attachments untouched', async page => {
      exhaustFixture(); await page.reload(); await page.locator('#followup-content').fill('这是尚未发送的问题');
      await page.locator('#followup-attachments-file').setInputFiles({name:'question.txt',mimeType:'text/plain',buffer:Buffer.from('question reference')});
      await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor();
      await page.locator('#task-summary').getByRole('button',{name:'保留修改，开始下一批',exact:true}).click(); await page.waitForFunction(()=>location.hash==='#retained-next');
      assert.deepEqual(calls.find(call=>call.action==='exhausted-next-batch').body.attachment_ids,[]);
      await page.locator('[data-task-select="a"]').click();
      assert.equal(await page.locator('#followup-content').inputValue(),'这是尚未发送的问题');
      assert.equal(await page.locator('#followup-attachments [data-upload-status="ready"]').count(),1);
    }, '#a');
    // @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
    await scenario('exhausted header cannot bypass an unfinished requirement attachment upload', async page => {
      exhaustFixture(); await page.reload(); await page.locator('#followup-kind').selectOption('requirement');
      let release; uploadGate = new Promise(resolve => {release=resolve;});
      await page.locator('#followup-attachments-file').setInputFiles({name:'pending.txt',mimeType:'text/plain',buffer:Buffer.from('required reference')});
      await page.locator('#followup-attachments [data-upload-status="uploading"]').waitFor();
      const begin = page.locator('#task-summary').getByRole('button',{name:'保留修改，开始下一批',exact:true});
      assert.equal(await begin.isDisabled(),true);
      await page.locator('#followup-kind').selectOption('question'); assert.equal(await begin.isEnabled(),true, 'default continuation does not consume question drafts');
      await page.locator('#followup-kind').selectOption('requirement'); assert.equal(await begin.isDisabled(),true);
      release(); await page.locator('#followup-attachments [data-upload-status="ready"]').waitFor(); assert.equal(await begin.isEnabled(),true);
      await begin.click(); await page.waitForFunction(()=>location.hash==='#retained-next'); assert.equal(calls.find(call=>call.action==='exhausted-next-batch').body.attachment_ids.length,1);
    }, '#a');
    function checkpointFixture(state = 'blocked', complete = false) {
      return Object.assign(rows[0], {
        state, round: 3, base: 'a'.repeat(40), baseline: { hash: 'frozen-source' }, runs: [],
        checks: [{ command: 'npm run test', exit_code: 0, log: 'check-1.log' }],
        review: { verdict: 'pass', summary: '核对实际修改与检查日志', findings: [] },
        reason: 'local_checkpoint_staged: manually staged changes were preserved',
        local_checkpoint: { base: 'a'.repeat(40), complete, commit: complete ? '1234567890abcdef1234567890abcdef12345678' : '' },
      });
    }
    for (const mobile of [false, true]) {
      await scenario('accepted checkpoint at three rounds blocks new work and retains drafts on ' + (mobile ? 'phone' : 'desktop'), async page => {
        checkpointFixture(); await page.reload();
        const finish = page.getByRole('button', { name: '完成本地存档', exact: true });
        assert.equal(await finish.isEnabled(), true);
        assert.equal(await page.getByRole('button', { name: '保留修改，开始下一批', exact: true }).count(), 0);
        assert.equal(await page.getByRole('button', { name: '停止此任务', exact: true }).count(), 0);
        await page.locator('#followup-kind').selectOption('requirement');
        await page.locator('#followup-content').fill('存档完成后继续优化首页');
        assert.equal(await page.locator('#followup-send').isDisabled(), true);
        assert.match(await page.locator('#task-summary').innerText(), /已验收.*本地存档/s);
        assert.match(await summaryRules(page), /不.*消耗轮次/);
        assert.doesNotMatch(await page.locator('#task-summary').innerText(), /local_checkpoint_staged|manually staged/);
        if (mobile) await page.locator('#sidebar-toggle').click();
        await page.locator('#refresh').click(); await page.reload();
        assert.equal(await page.locator('#followup-content').inputValue(), '存档完成后继续优化首页');
        assert.equal(await page.locator('#followup-send').isDisabled(), true);
        assert.equal(calls.length, 0, 'viewing pending acceptance must not resume or create another batch');
        const bounds = await finish.boundingBox(); assert.ok(bounds.height >= 44 && bounds.x >= 0 && bounds.x + bounds.width <= (mobile ? 390 : 1440));
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
        await page.screenshot({ path: out + '/checkpoint-pending-' + (mobile ? 'mobile' : 'desktop') + '.png', fullPage: true });
      }, '#a', mobile);
    }
    await scenario('accepted checkpoint resumes once without consuming rounds and preserves the next requirement', async page => {
      const row = checkpointFixture(); let release; messageGate = new Promise(resolve => { release = resolve; }); await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('下一批改善导出效果');
      const resumed = page.waitForRequest(request => request.url().endsWith('/resume'));
      await page.getByRole('button', { name: '完成本地存档', exact: true }).click();
      await resumed;
      assert.equal(await page.getByRole('button', { name: '完成本地存档', exact: true }).isDisabled(), true);
      await page.locator('#refresh').click();
      assert.equal(await page.locator('#followup-send').isDisabled(), true);
      assert.equal(calls.filter(call => call.action === 'resume').length, 1);
      release(); await page.waitForFunction(() => document.getElementById('task-summary').textContent.includes('12345678'));
      assert.equal(row.state, 'accepted'); assert.equal(row.round, 3);
      assert.equal(await page.locator('#followup-content').inputValue(), '下一批改善导出效果');
      assert.equal(await page.locator('#followup-send').isEnabled(), true);
      assert.deepEqual(calls.map(call => call.action), ['resume']);
    }, '#a');
    await scenario('accepted checkpoint with a completed commit still finishes an interrupted acceptance explicitly', async page => {
      checkpointFixture('interrupted', true); await page.reload();
      assert.equal(await page.getByRole('button', { name: '完成本地存档', exact: true }).isEnabled(), true);
      assert.equal(await page.getByRole('button', { name: '保留修改，开始下一批', exact: true }).count(), 0);
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('先完成上一次验收');
      assert.equal(await page.locator('#followup-send').isDisabled(), true); assert.equal(calls.length, 0);
      await page.getByRole('button', { name: '完成本地存档', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('task-summary').textContent.includes('12345678'));
      assert.deepEqual(calls.map(call => call.action), ['resume']); assert.equal(rows[0].round, 3);
    }, '#a');
    await scenario('accepted checkpoint error retains the requirement and offers a localized explicit retry', async page => {
      checkpointFixture(); await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('不要丢失这个下一批要求');
      actionFailure = 'local_checkpoint_locked: another Git operation holds the index lock';
      await page.getByRole('button', { name: '完成本地存档', exact: true }).click();
      await page.locator('#global-error').waitFor({ state: 'visible' });
      assert.match(await page.locator('#global-error').innerText(), /本地存档.*保留|文件.*保留/s);
      assert.doesNotMatch(await page.locator('#global-error').innerText(), /local_checkpoint_|index lock/);
      assert.equal(await page.locator('#followup-content').inputValue(), '不要丢失这个下一批要求');
      assert.equal(await page.locator('#followup-send').isDisabled(), true);
      actionFailure = false; await page.getByRole('button', { name: '完成本地存档', exact: true }).click();
      await page.waitForFunction(() => document.getElementById('task-summary').textContent.includes('12345678'));
      assert.deepEqual(calls.map(call => call.action), ['resume', 'resume']);
      assert.equal(await page.locator('#followup-content').inputValue(), '不要丢失这个下一批要求');
    }, '#a');
    await scenario('accepted local version stays visible while the explicit next requirement creates a zero-round child', async page => {
      checkpointFixture('accepted', true); delete rows[0].reason; await page.reload();
      assert.match(await page.locator('#task-summary').innerText(), /本地 Git 版本 12345678/);
      assert.equal(await page.getByRole('button', { name: '完成本地存档', exact: true }).count(), 0);
      await tab(page, 'results'); assert.match(await page.locator('#task-summary').innerText(), /12345678/);
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('在当前项目继续改首页');
      assert.equal(await page.locator('#followup-send').textContent(), '开始下一批'); assert.equal(await page.locator('#followup-send').isEnabled(), true);
      assert.equal(calls.length, 0); await page.locator('#followup-send').click();
      await page.waitForFunction(() => location.hash === '#next-batch');
      const child = rows.find(row => row.id === 'next-batch'); assert.equal(child.round, 0); assert.equal(child.local_checkpoint, undefined);
      assert.equal(child.project, 'E:\\记账项目'); assert.equal(child.parent_task_id, 'a');
      assert.deepEqual(calls.map(call => call.action), ['messages']); assert.equal(calls[0].body.kind, 'requirement');
    }, '#a');
    await scenario('pending accepted checkpoint saves only future model settings without changing the sealed plan', async page => {
      const row = checkpointFixture(); const original = JSON.stringify(row.model_plan); await page.reload();
      await settings(page, 'backend'); await page.locator('#task-plan-backend-effort').selectOption('high');
      assert.equal(await page.locator('#save-plan').textContent(), '保存后续设置'); await page.locator('#save-plan').click();
      await page.waitForFunction(() => document.getElementById('draft-hint').textContent.includes('后续设置已保存'));
      assert.equal(JSON.stringify(row.model_plan), original); assert.equal(row.round, 3); assert.equal(row.state, 'blocked');
      assert.equal(row.future_plan.plan.backend_effort, 'high'); assert.deepEqual(calls.map(call => call.action), ['future-models']);
    }, '#a');
    function exhaustFixture() {
      Object.assign(rows[0], { state: 'exhausted', round: 3, base: 'frozen-base', baseline: { hash: 'frozen-source' }, runs: [], review: { verdict: 'rework', summary: '图片来源仍需检查', findings: ['修复图片来源判断'] } });
      return rows[0];
    }
    // @covers AC-RELIABLE-RENEWAL-001.1/.2/.7
    await scenario('exhausted batch explicitly offers retained changes and never starts from viewing or polling', async page => {
      exhaustFixture(); await page.reload();
      const begin = page.getByRole('button', { name: '保留修改，开始下一批', exact: true });
      assert.equal(await begin.isEnabled(), true);
      assert.equal(await page.locator('#followup-send').isDisabled(), true, 'empty read-only questions must still require content');
      assert.match(await page.locator('#task-summary .current-status').innerText(), /本批.*3\/3.*下一批/s);
      await page.locator('#refresh').click(); await page.reload(); assert.equal(calls.length, 0);
      await page.locator('#followup-kind').selectOption('requirement');
      assert.equal(await page.locator('#followup-send').textContent(), '保留修改，开始下一批');
      assert.equal(await page.locator('#followup-send').isEnabled(), true, 'an explicit empty request continues the inherited goal');
      await page.locator('#followup-form .composer-help > summary').click();
      assert.match(await page.locator('#followup-hint').innerText(), /已有修改.*未解决|未解决.*已有修改/);
      await page.screenshot({ path: out + '/renewal-desktop.png', fullPage: true });
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.1/.2/.7
    await scenario('exhausted batch button sends once and selects a fresh linked batch at zero rounds', async page => {
      const parent = exhaustFixture(); let release; messageGate = new Promise(resolve => { release = resolve; }); await page.reload();
      await page.locator('#followup-content').fill('保留这个尚未发出的问题');
      const begin = page.getByRole('button', { name: '保留修改，开始下一批', exact: true }); await begin.click();
      const busy = page.getByRole('button', { name: '正在建立下一批…', exact: true }); await busy.waitFor();
      assert.equal(await busy.isDisabled(), true); await page.locator('#refresh').click();
      assert.equal(await busy.isDisabled(), true); assert.equal(calls.filter(call => call.action === 'exhausted-next-batch').length, 1);
      release(); await page.waitForFunction(() => location.hash === '#retained-next');
      await page.getByRole('link', { name: '查看上一批的项目记录', exact: true }).waitFor();
      assert.match(await page.locator('#task-summary .current-status').innerText(), /正在检查上一批修改/);
      assert.equal(parent.round, 3); assert.equal(parent.state, 'exhausted'); assert.equal(rows.find(row => row.id === 'retained-next').round, 0);
      const sent = calls.find(call => call.action === 'exhausted-next-batch'); assert.equal(sent.body.kind, 'requirement'); assert.equal(sent.body.content, ''); assert.ok(sent.body.id);
      await page.locator('[data-task-select="a"]').click();
      assert.equal(await page.locator('#followup-content').inputValue(), '保留这个尚未发出的问题');
      const linked = page.locator('#task-summary').getByRole('link', { name: '打开下一批', exact: true }); assert.equal(await linked.getAttribute('href'), '#retained-next');
      assert.equal(await begin.count(), 0); await linked.click(); await page.waitForFunction(() => location.hash === '#retained-next');
      assert.equal(calls.filter(call => call.action === 'exhausted-next-batch').length, 1);
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.1/.3/.7
    await scenario('exhausted batch requirement retries with its saved identity and retains content on conflict', async page => {
      exhaustFixture(); await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('继续修复图片来源并增加导出');
      actionFailure = 'renewal_conflict: another unfinished task owns the project'; await page.locator('#followup-send').click();
      await page.locator('#followup-error').waitFor({ state: 'visible' });
      assert.match(await page.locator('#followup-error').innerText(), /另一个未完成任务/); assert.doesNotMatch(await page.locator('#followup-error').innerText(), /renewal_|unfinished/);
      assert.equal(await page.locator('#followup-content').inputValue(), '继续修复图片来源并增加导出');
      assert.equal(await page.locator('#followup-send').isEnabled(), true); const nonce = calls.at(-1).body.id;
      actionFailure = false; await page.locator('#followup-send').click(); await page.waitForFunction(() => location.hash === '#retained-next');
      assert.equal(await page.locator('#task-summary h1').textContent(), '继续修复图片来源并增加导出');
      assert.equal(calls.at(-1).body.id, nonce); assert.equal(calls.at(-1).body.content, '继续修复图片来源并增加导出');
      assert.equal(calls.filter(call => call.action === 'messages').length, 0, 'exhaustion must use the retained candidate API');
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.1/.3/.7
    await scenario('exhausted batch eligibility and existing successor remain explicit across reload', async page => {
      const parent = exhaustFixture();
      for (const state of ['exhausted', 'cancelled', 'blocked', 'interrupted']) {
        parent.state = state; await page.reload(); assert.equal(await page.locator('#task-summary').getByRole('button', { name: '保留修改，开始下一批', exact: true }).isEnabled(), true);
      }
      for (const patch of [{ state: 'reviewing' }, { state: 'exhausted', read_only: true }, { read_only: false, archived_at: new Date().toISOString() }, { archived_at: undefined, base: '' }, { base: 'base', baseline: undefined }]) {
        Object.assign(parent, patch); await page.reload(); assert.equal(await page.locator('#task-summary').getByRole('button', { name: '保留修改，开始下一批', exact: true }).count(), 0);
      }
      Object.assign(parent, { state: 'exhausted', baseline: { hash: 'source' }, successor_task_id: 'existing-next' }); await page.reload();
      assert.equal(await page.locator('#task-summary').getByRole('link', { name: '打开下一批', exact: true }).getAttribute('href'), '#existing-next');
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('不能从旧批次重复开启');
      await page.locator('#followup-form .composer-help > summary').click();
      assert.equal(await page.locator('#followup-send').isDisabled(), true); assert.match(await page.locator('#followup-hint').innerText(), /已经.*下一批|已开启下一批/);
      assert.equal(calls.length, 0);
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.4/.7
    await scenario('next batch seed validation blocks extra requirements until the retained source is checked', async page => {
      const child = exhaustFixture(); Object.assign(child, { state: 'queued', round: 0, parent_task_id: 'previous', seed_pending: { parent_task_id: 'previous' } }); await page.reload();
      assert.match(await page.locator('#task-summary .current-status').innerText(), /正在检查上一批修改/);
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('等待继承检查之后的要求');
      await page.locator('#followup-form .composer-help > summary').click();
      assert.equal(await page.locator('#followup-send').isDisabled(), true); assert.match(await page.locator('#followup-hint').innerText(), /继承.*检查|检查.*继承/);
      assert.equal(calls.length, 0); delete child.seed_pending; child.state = 'auditing'; await page.locator('#refresh').click();
      await page.waitForFunction(() => !document.querySelector('#followup-send').disabled);
      assert.equal(await page.locator('#followup-content').inputValue(), '等待继承检查之后的要求'); assert.equal(calls.length, 0);
      await page.screenshot({ path: out + '/renewal-child.png', fullPage: true });
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.3/.7
    await scenario('exhausted batch reports actionable validation errors while preserving its draft', async page => {
      exhaustFixture(); await page.reload(); await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('保留补充目标');
      for (const [code, expected] of [['renewal_source_changed', /原项目|副本/], ['renewal_evidence_invalid', /证据/], ['renewal_parent_active', /进程|执行/], ['renewal_ineligible', /状态|条件/], ['batch_next_unavailable', /状态|条件/], ['batch_project_conflict', /另一个未完成任务/], ['batch_parent_changed', /原项目|副本/], ['batch_parent_active', /进程|执行/], ['batch_seed_invalid', /证据/], ['batch_original_changed', /原项目/], ['batch_successor_exists', /下一批/], ['renewal_active', /继承|检查/], ['task_running', /执行/]]) {
        actionFailure = code + ': internal diagnostic'; await page.locator('#followup-send').click(); await page.locator('#followup-error').waitFor({ state: 'visible' });
        const explanation = await page.locator('#followup-error').innerText(); assert.match(explanation, expected); assert.match(explanation, /保留/); assert.doesNotMatch(explanation, /renewal_|batch_|task_running|internal/);
        assert.equal(await page.locator('#followup-content').inputValue(), '保留补充目标'); assert.equal(await page.locator('#followup-send').isEnabled(), true);
      }
      assert.equal(calls.length, 13);
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.2/.7
    await scenario('delayed exhausted batch response never moves another selected project or clears its draft', async page => {
      exhaustFixture(); let release; messageGate = new Promise(resolve => { release = resolve; }); await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('同项目继续修改'); await page.locator('#followup-send').click();
      await page.locator('[data-task-select="b"]').click(); await page.locator('#followup-content').fill('任务 B 的草稿'); release();
      await page.waitForResponse(response => response.url().endsWith('/exhausted-next-batch'));
      assert.equal(new URL(page.url()).hash, '#b'); assert.equal(await page.locator('#followup-content').inputValue(), '任务 B 的草稿');
      assert.equal(await page.locator('#task-summary h1').textContent(), '整理导出功能');
    }, '#a');
    // @covers AC-RELIABLE-RENEWAL-001.7
    await scenario('phone exhausted batch actions remain reachable and the retained continuation stays within the viewport', async page => {
      exhaustFixture(); await page.reload();
      const begin = page.locator('#task-summary').getByRole('button', { name: '保留修改，开始下一批', exact: true }), open = page.getByRole('button', { name: '查看工作副本', exact: true });
      const first = await begin.boundingBox(), second = await open.boundingBox(); assert.ok(first.height >= 44 && second.height >= 44); assert.ok(separated(first, second), 'phone task actions overlap'); assert.ok(first.x >= 0 && first.x + first.width <= 390 && second.x >= 0 && second.x + second.width <= 390);
      await begin.scrollIntoViewIfNeeded(); assert.equal(await begin.evaluate(node => { const rect = node.getBoundingClientRect(); return node.contains(document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2)); }), true);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/renewal-mobile.png', fullPage: true }); await begin.click(); await page.waitForFunction(() => location.hash === '#retained-next');
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/renewal-child-mobile.png', fullPage: true });
    }, '#a', true);
    // @covers AC-RELIABLE-RENEWAL-001.4/.5/.7
    await scenario('superseded batch freezes its old model drawer and directs subsequent choices to the linked child', async page => {
      const parent = exhaustFixture(); await page.reload(); await settings(page, 'backend');
      await page.locator('#task-plan-backend-effort').selectOption('high');
      assert.equal(await page.locator('#save-plan').isEnabled(), true);
      parent.successor_task_id = 'retained-next';
      rows.unshift({ ...parent, id: 'retained-next', parent_task_id: parent.id, successor_task_id: undefined, state: 'auditing', round: 0, plan_revision: 1 });
      await page.locator('#refresh').click();
      await page.waitForFunction(() => document.querySelector('#task-plan-backend-effort').disabled);
      assert.equal(await page.locator('#save-plan').isVisible(), false);
      assert.match(await page.locator('#task-plan-editor').innerText(), /本批已开启下一批，请到下一批调整 Agent/);
      assert.match(await page.locator('#draft-hint').innerText(), /下一批/);
      await page.keyboard.press('Escape');
      await page.locator('#task-summary').getByRole('link', { name: '打开下一批', exact: true }).click();
      await page.waitForFunction(() => location.hash === '#retained-next'); await settings(page, 'backend');
      assert.equal(await page.locator('#task-plan-backend-effort').isEnabled(), true);
      assert.equal(await page.locator('#task-plan-backend').inputValue(), 'codex');
      await page.locator('[data-agent-role="frontend"]').click(); assert.equal(await page.locator('#task-plan-frontend').inputValue(), 'deepseek');
      await page.locator('[data-agent-role="reviewer"]').click(); assert.equal(await page.locator('#task-plan-reviewer').inputValue(), 'codex');
      assert.equal(calls.length, 0, 'viewing linked assignments must not save historical or child settings');
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.1/.5/.7
    await scenario('cancelled recovery is explicit and reports the retained round budget', async page => {
      rows[0].state = 'cancelled'; rows[0].runs = [];
      for (const round of [0, 1, 2]) {
        rows[0].round = round; await page.reload();
		await page.getByRole('button', { name: '检查副本并继续', exact: true }).waitFor({ state: 'visible' });
        assert.ok(await page.getByRole('button', { name: '检查副本并继续', exact: true }).isVisible());
        assert.match(await summaryRules(page), new RegExp('已用 ' + round + '/3 轮，还可修改 ' + (3 - round) + ' 轮'));
        await page.locator('#refresh').click();
        assert.equal(calls.length, 0, 'opening and refreshing a stopped task must not submit work');
        if (round === 1) await page.screenshot({ path: out + '/recovery-desktop.png', fullPage: true });
      }
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('停止期间保留的要求');
      assert.equal(await page.locator('#followup-send').isDisabled(), true);
      assert.equal(calls.length, 0);
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.1/.4/.7
    await scenario('one recovery click stays pending through polling and releases requirements only after validation', async page => {
      rows[0].state = 'cancelled'; rows[0].runs = []; await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('检查通过后再接收这个要求');
      await page.getByRole('button', { name: '检查副本并继续', exact: true }).waitFor();
      const response = page.waitForResponse(item => item.url().endsWith('/recover-cancelled'));
      await page.getByRole('button', { name: '检查副本并继续', exact: true }).click();
      assert.equal((await response).status(), 202);
      const checking = page.getByRole('button', { name: '正在检查保留的副本', exact: true }); await checking.waitFor();
      assert.equal(await checking.isDisabled(), true);
      assert.equal(await page.locator('#followup-send').isDisabled(), true);
      await page.locator('#refresh').click();
      assert.equal(await checking.isDisabled(), true);
      assert.equal(new URL(page.url()).hash, '#a');
      assert.equal(await page.locator('#followup-content').inputValue(), '检查通过后再接收这个要求');
      assert.deepEqual(calls, [{ id: 'a', action: 'recover-cancelled', body: {} }]);
      await page.screenshot({ path: out + '/recovery-pending.png', fullPage: true });
      delete rows[0].recovery_pending; rows[0].state = 'auditing'; await page.locator('#refresh').click();
      await page.waitForFunction(() => !document.querySelector('#followup-send').disabled);
      assert.equal(await checking.count(), 0); assert.equal(rows[0].round, 1);
      assert.equal(calls.length, 1, 'validation success must not replay the saved requirement');
      await page.locator('#followup-send').click();
      await page.waitForFunction(() => document.querySelectorAll('[data-message-id]').length === 1);
      assert.equal(calls.filter(call => call.action === 'messages').length, 1);
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.3/.4/.7
    await scenario('recovery server rejection preserves selection and draft and allows explicit retry', async page => {
      rows[0].state = 'cancelled'; rows[0].runs = []; await page.reload();
      await page.locator('#followup-content').fill('保留未发送的问题');
      actionFailure = 'recovery_project_conflict: another unfinished task owns this project';
      await page.getByRole('button', { name: '检查副本并继续', exact: true }).click(); await page.locator('#global-error').waitFor();
      assert.match(await page.locator('#global-error').innerText(), /另一个未完成任务|其他未完成任务/);
      assert.doesNotMatch(await page.locator('#global-error').innerText(), /recovery_[a-z_]+:|unfinished/);
      assert.equal(await page.getByRole('button', { name: '检查副本并继续', exact: true }).isEnabled(), true);
      assert.equal(await page.locator('#followup-content').inputValue(), '保留未发送的问题');
      assert.equal(new URL(page.url()).hash, '#a'); assert.equal(rows[0].state, 'cancelled'); assert.equal(rows[0].round, 1);
      actionFailure = false; await page.getByRole('button', { name: '检查副本并继续', exact: true }).click();
      await page.getByRole('button', { name: '正在检查保留的副本', exact: true }).waitFor();
      assert.equal(calls.filter(call => call.action === 'recover-cancelled').length, 2);
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.3/.6/.7
    await scenario('failed recovery validation shows specific nondestructive causes after polling', async page => {
      rows[0].state = 'queued'; rows[0].runs = []; rows[0].recovery_pending = { attempt: 'fixture-recovery', round: 1, phase: 'validating' }; await page.reload();
      assert.equal(await page.getByRole('button', { name: '正在检查保留的副本', exact: true }).isDisabled(), true);
      const failures = [
        ['recovery_original_changed: original drift', /原项目.*变化/],
        ['recovery_candidate_invalid: task ownership mismatch', /副本.*归属|副本.*身份/],
        ['recovery_resource_invalid: frozen manifest mismatch', /资源/],
        ['recovery_evidence_invalid: missing receipt', /证据/],
        ['recovery_process_unknown: cannot establish cleanup', /进程.*停止|进程.*结束/],
      ];
      for (const [reason, expected] of failures) {
        rows[0].state = 'cancelled'; rows[0].reason = reason; delete rows[0].recovery_pending; await page.locator('#refresh').click();
        await page.waitForFunction(source => new RegExp(source).test(document.querySelector('#task-summary .detail-section')?.textContent || ''), expected.source);
        const explanation = await page.locator('#task-summary .detail-section').innerText();
        assert.match(explanation, expected); assert.match(explanation, /保留/); assert.doesNotMatch(explanation, /recovery_[a-z_]+:/);
        assert.equal(await page.getByRole('button', { name: '检查副本并继续', exact: true }).isEnabled(), true);
      }
      assert.equal(calls.length, 0, 'validation failures and polling must not retry automatically');
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.4/.5/.7
    await scenario('archive restoration never starts recovery and ineligible tasks retain file access', async page => {
      rows[0].state = 'cancelled'; rows[0].runs = []; rows[0].archived_at = new Date().toISOString(); await page.reload();
      assert.equal(await page.getByRole('button', { name: '检查副本并继续', exact: true }).count(), 0);
      await page.getByRole('button', { name: '恢复到任务列表', exact: true }).click();
      await page.getByRole('button', { name: '检查副本并继续', exact: true }).waitFor();
      assert.deepEqual(calls.map(call => call.action), ['unarchive']); assert.equal(rows[0].state, 'cancelled');
      for (const patch of [{ read_only: true, round: 1, state: 'cancelled' }, { read_only: false, round: 3, state: 'cancelled' }, { read_only: false, round: 1, state: 'cancelling' }]) {
        Object.assign(rows[0], patch); await page.reload();
        await page.getByRole('button', { name: '查看工作副本', exact: true }).waitFor({ state: 'visible' });
        assert.equal(await page.getByRole('button', { name: '检查副本并继续', exact: true }).count(), 0);
        assert.equal(await page.getByRole('button', { name: '查看工作副本', exact: true }).isVisible(), true);
        if (patch.round === 3) assert.match(await summaryRules(page), /已用 3\/3 轮.*本批.*不能继续/s);
      }
      assert.deepEqual(calls.map(call => call.action), ['unarchive']);
    }, '#a');
    // @covers AC-RELIABLE-RECOVERY-001.1/.7
    await scenario('phone recovery keeps actions checking and failure states reachable within the viewport', async page => {
      rows[0].state = 'cancelled'; rows[0].runs = []; await page.reload();
      const recover = page.getByRole('button', { name: '检查副本并继续', exact: true }), open = page.getByRole('button', { name: '查看工作副本', exact: true });
      const first = await recover.boundingBox(), second = await open.boundingBox();
      assert.ok(separated(first, second), 'phone recovery controls overlap');
      assert.ok(first.height >= 44 && second.height >= 44); assert.ok(first.x >= 0 && first.x + first.width <= 390 && second.x >= 0 && second.x + second.width <= 390);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/recovery-mobile.png', fullPage: true });
      await recover.click(); const checking = page.getByRole('button', { name: '正在检查保留的副本', exact: true }); await checking.waitFor();
      assert.equal(await checking.isDisabled(), true);
      rows[0].state = 'cancelled'; rows[0].reason = 'recovery_original_changed: original drift'; delete rows[0].recovery_pending;
      await recover.waitFor({ timeout: 5000 }); assert.match(await page.locator('#task-summary .detail-section').innerText(), /原项目.*变化/);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      assert.equal(calls.filter(call => call.action === 'recover-cancelled').length, 1);
      await page.screenshot({ path: out + '/recovery-error-mobile.png', fullPage: true });
    }, '#a', true);
    // @covers AC-RELIABLE-RECOVERY-001.5/.6/.7
    await scenario('restart retains the validation guard and requires an explicit continuation', async page => {
      rows[0].state = 'interrupted'; rows[0].runs = []; rows[0].recovery_pending = { attempt: 'fixture-recovery', round: 1, phase: 'validating' }; await page.reload();
      assert.equal(await page.getByRole('button', { name: '正在检查保留的副本', exact: true }).count(), 0, 'an interrupted check must not appear active');
      const resume = page.getByRole('button', { name: '处理后继续', exact: true }); assert.equal(await resume.isEnabled(), true);
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('重启后仍需保留的要求');
      assert.equal(await page.locator('#followup-send').isDisabled(), true); assert.equal(calls.length, 0);
      await resume.click(); const checking = page.getByRole('button', { name: '正在检查保留的副本', exact: true }); await checking.waitFor();
      assert.equal(await checking.isDisabled(), true); assert.equal(await page.locator('#followup-send').isDisabled(), true);
      assert.equal(await page.locator('#followup-content').inputValue(), '重启后仍需保留的要求');
      assert.equal(rows[0].round, 1); assert.deepEqual(calls, [{ id: 'a', action: 'resume', body: {} }]);
    }, '#a');
    await scenario('old file-limit pause offers a version-aware retry without resetting rounds', async page => {
      rows[0].state='blocked';rows[0].round=0;rows[0].reason='source_snapshot_too_large: this repository tracks more than 30000 files; use read-only inspection or a smaller source repository';await page.reload();
      await page.locator('#task-summary').getByText('这条任务上次被旧版的项目文件数量限制暂停').waitFor({state:'visible'});
      assert.ok(await page.getByRole('button',{name:'处理后继续',exact:true}).isVisible());
      assert.equal(calls.filter(call=>call.action==='run').length,0);
    }, '#a');
    await scenario('resource failures explain preserved inputs without suggesting deletion', async page => {
      rows[0].state='blocked'; rows[0].reason='resource_changed: frozen input changed: static/figs/a.wmf'; await page.reload();
      await page.locator('#task-summary').getByText('输入资源与冻结的版本不一致').waitFor({state:'visible'});
      assert.ok(await page.getByRole('button',{name:'处理后继续',exact:true}).isVisible());
      assert.equal(await page.getByText('resource_changed:',{exact:false}).count(),0);
    }, '#a');
    await scenario('mobile resource-size failure preserves usable task controls', async page => {
      rows[0].state='blocked'; rows[0].reason='resource_size: requested input exceeds 64 MiB'; await page.reload();
      await page.locator('#task-summary').getByText('这次需要的输入资源超过读取上限').waitFor({state:'visible'});
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
      assert.ok(await page.getByRole('button',{name:'处理后继续',exact:true}).isVisible());
    }, '#a', true);
    await scenario('stage completion preserves edits and keeps a remote future-plan conflict until reload', async page => {
      await settings(page,'backend'); await page.locator('#task-plan-backend-effort').selectOption('high');
      rows[0].state='awaiting_acceptance'; rows[0].future_plan={revision:1,plan:{...plan,backend_effort:'low'}};
      await page.locator('#refresh').click();
      await page.waitForFunction(()=>document.querySelector('#draft-hint').textContent.includes('其他窗口'));
      assert.equal(await page.locator('#task-plan-backend-effort').inputValue(),'high');
      assert.equal(await page.locator('#save-plan').isDisabled(),true);
      await page.locator('#refresh').click();
      assert.equal(await page.locator('#save-plan').isDisabled(),true);
      await page.locator('#reset-plan').click();
      assert.equal(await page.locator('#task-plan-backend-effort').inputValue(),'low');
      assert.equal(calls.some(call=>call.action==='future-models'),false);
    }, '#a');
    await scenario('completed inspection exposes editable future model and reasoning in the side drawer', async page => {
      rows[0].state='inspected'; rows[0].read_only=true; rows[0].runs=[]; await page.reload();
      await page.locator('#agent-settings-toggle').click(); await page.locator('#agent-settings-drawer').waitFor({state:'visible'});
      await page.locator('[data-agent-role="auditor"]').click();
      const toolBox=await page.locator('#task-plan-auditor').boundingBox(), modelBox=await page.locator('#task-plan-auditor-model').boundingBox(), effortBox=await page.locator('#task-plan-auditor-effort').boundingBox();
      assert.ok(modelBox.y>=toolBox.y+toolBox.height&&effortBox.y>=modelBox.y+modelBox.height,'drawer controls should stack with readable full-width labels');
      await page.locator('#task-plan-auditor').selectOption('codex'); await page.locator('#task-plan-auditor-effort').selectOption('high');
      const saved=page.waitForResponse(response=>response.url().endsWith('/future-models')&&response.request().method()==='POST');
      await page.locator('#save-plan').click(); await saved;
      assert.equal(calls.find(call=>call.action==='future-models').body.plan.auditor_effort,'high');
      await page.reload(); await page.locator('#agent-settings-toggle').click(); await page.locator('[data-agent-role="auditor"]').click();
      assert.equal(await page.locator('#task-plan-auditor-effort').inputValue(),'high');
      assert.equal(rows[0].model_plan.auditor_effort,undefined);
      await page.screenshot({path:out+'/agent-drawer-future.png',fullPage:true});
    }, '#a');
    await scenario('desktop agent drawer toggles without main-page scrolling and preserves role choices', async page => {
      const scroll=await page.locator('#main-content').evaluate(node=>{node.scrollTop=260;return node.scrollTop});
      await page.locator('#agent-settings-toggle').click(); await page.locator('[data-agent-role="backend"]').click();
      await page.locator('#task-plan-backend-effort').selectOption('high');
      await page.locator('[data-agent-role="auditor"]').click(); await page.locator('[data-agent-role="backend"]').click();
      assert.equal(await page.locator('#task-plan-backend-effort').inputValue(),'high');
      assert.equal(await page.locator('#main-content').evaluate(node=>node.scrollTop),scroll);
      await page.locator('#agent-settings-toggle').click(); assert.equal(await page.locator('#agent-settings-drawer').isVisible(),false);
      await page.locator('#agent-settings-toggle').click(); await page.keyboard.press('Escape');
      assert.equal(await page.locator('#agent-settings-toggle').evaluate(node=>node===document.activeElement),true);
    }, '#a');
    await scenario('mobile agent drawer traps focus, closes with Escape and avoids overflow', async page => {
      await page.locator('#agent-settings-toggle').click();
      assert.equal(await page.locator('#agent-settings-drawer').getAttribute('aria-modal'),'true');
      assert.equal(await page.locator('.app-layout').evaluate(node=>node.inert),true);
      await page.keyboard.press('Shift+Tab');
      assert.equal(await page.locator('#agent-settings-drawer').evaluate(node=>node.contains(document.activeElement)),true);
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
      await page.screenshot({path:out+'/agent-drawer-mobile.png',fullPage:true});
      await page.keyboard.press('Escape');
      assert.equal(await page.locator('.app-layout').evaluate(node=>node.inert),false);
      assert.equal(await page.locator('#agent-settings-toggle').evaluate(node=>node===document.activeElement),true);
    }, '#a', true);
    await scenario('a delayed send never clears another task draft', async page => {
      let release; messageGate = new Promise(resolve => { release = resolve; });
      await page.locator('#followup-content').fill('任务 A 的问题'); await page.locator('#followup-send').click();
      await page.waitForFunction(()=>document.querySelector('#followup-send').disabled);
      await page.locator('[data-task-select="b"]').click(); await page.locator('#followup-content').fill('任务 B 未发送的草稿');
      release(); await page.waitForTimeout(200);
      assert.equal(await page.locator('#followup-content').inputValue(), '任务 B 未发送的草稿');
      assert.equal(await page.locator('#task-summary h1').textContent(), '整理导出功能');
    }, '#a');
    await scenario('completed inspection starts the next batch with a persistent previous-task link', async page => {
      rows[0].state='inspected'; rows[0].read_only=true; rows[0].runs=[]; await page.reload();
      await page.locator('#followup-kind').selectOption('requirement'); await page.locator('#followup-content').fill('接着实现导出功能');
      assert.equal(await page.locator('#followup-send').textContent(),'开始下一批');
      await page.locator('#followup-send').click();
      await page.waitForFunction(()=>location.hash==='#next-batch');
      await page.getByRole('link',{name:'查看上一批的项目记录'}).waitFor();
      assert.equal(await page.locator('#task-summary h1').textContent(),'接着实现导出功能');
      assert.equal(calls.find(call=>call.action==='messages').id,'a');
    }, '#a');
    await scenario('reply text is escaped and Chinese composition does not accidentally submit', async page => {
      rows[0].messages=[{id:'actual-q',kind:'question',content:'说明项目',status:'answered',reply:{summary:'<img src=x onerror="window.injected=true">',facts:['app.txt: 已读取'],unknowns:['尚未运行测试'],read_files:['app.txt']}}];
      await page.reload(); await page.locator('.conversation-reply').waitFor();
      assert.equal(await page.locator('.conversation-reply img').count(),0);
      await page.locator('#followup-content').fill('正在输入中文');
      await page.locator('#followup-content').evaluate(node=>node.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',ctrlKey:true,isComposing:true,bubbles:true})));
      assert.equal(calls.filter(call=>call.action==='messages').length,0);
      await page.locator('#followup-content').press('Control+Enter');
      await page.waitForFunction(()=>document.querySelectorAll('[data-message-id]').length===2);
      assert.equal(calls.filter(call=>call.action==='messages').length,1);
      assert.equal(await page.evaluate(()=>!!window.injected),false);
    }, '#a');
    await scenario('task conversation sends repeated messages and preserves drafts across task changes', async page => {
      await page.locator('#followup-content').fill('先解释当前进度');
      await page.locator('[data-task-select="b"]').click();
      await page.locator('#followup-content').fill('导出按钮请保留');
      await page.locator('[data-task-select="a"]').click();
      assert.equal(await page.locator('#followup-content').inputValue(), '先解释当前进度');
      await page.locator('#followup-send').click();
      await page.locator('[data-message-id] .conversation-user').filter({hasText:'先解释当前进度'}).waitFor();
      await page.locator('#followup-content').fill('这条结论的依据是什么？');
      await page.locator('#followup-send').click();
      await page.waitForFunction(()=>document.querySelectorAll('[data-message-id]').length===2);
      assert.equal(calls.filter(call=>call.action==='messages').length,2);
      assert.equal(calls.filter(call=>call.action==='messages')[0].body.kind,'question');
      await page.reload();
      await page.waitForFunction(()=>document.querySelectorAll('[data-message-id]').length===2);
      await page.screenshot({path:out+'/conversation-desktop.png',fullPage:true});
    }, '#a');
    await scenario('message errors retain input and mobile composer does not overflow', async page => {
      actionFailure=true;
      await page.locator('#followup-content').fill('保留这个未发出的补充要求');
      await page.locator('#followup-kind').selectOption('requirement');
      await page.locator('#followup-send').click();
      await page.locator('#followup-error').waitFor({state:'visible'});
      assert.equal(await page.locator('#followup-content').inputValue(),'保留这个未发出的补充要求');
      assert.equal(await page.locator('#followup-send').isEnabled(),true);
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
      await page.screenshot({path:out+'/conversation-mobile.png',fullPage:true});
    }, '#a', true);
    await scenario('live process appends real projected events and preserves expanded details', async page => {
      await tab(page, 'activity');
      activity.push({sequence:1,time:new Date().toISOString(),kind:'command',title:'正在读取 README',detail:'type README.md',provider:'Codex',model:'gpt-6.1-sol',role:'auditor'});
      await page.locator('#activity-feed [data-sequence="1"]').waitFor();
      await page.locator('#activity-feed details > summary').click();
      activity.push({sequence:2,time:new Date().toISOString(),kind:'message',title:'Agent 的说明',detail:'已读取入口和说明文件。'});
      await page.locator('#activity-feed [data-sequence="2"]').waitFor();
      assert.equal(await page.locator('#activity-feed [data-sequence="1"] details').evaluate(node => node.open), true);
      await page.locator('#activity-filter').selectOption('model'); assert.equal(await page.locator('#activity-feed [data-kind="command"]').count(),0);
      assert.equal(await page.locator('#activity-feed [data-kind="message"]').count(),1);
      await page.locator('#activity-follow').uncheck(); await page.locator('#activity-filter').selectOption('all');
      assert.equal(await page.locator('#activity-follow').isChecked(),false);
      await page.screenshot({path:out+'/live-process.png',fullPage:true});
      await page.reload(); await page.locator('#activity-feed [data-sequence="2"]').waitFor();
    }, '#a');
    await scenario('read-only entry sends the selected existing directory without a second worktree', async page => {
      await page.locator('#project').fill('E:\\已经准备的工作副本'); await page.locator('#goal').fill('了解项目用途和进度');
      await page.locator('#read-only').check(); assert.equal(await page.locator('#work-mode').isVisible(),false);
      await settings(page,'auditor');
      assert.equal(await page.locator('#new-plan-auditor-model').isVisible(),true);
      assert.equal(await page.locator('#new-plan-frontend-model').isVisible(),false);
      await page.locator('#new-plan-auditor').selectOption('deepseek');
      await page.locator('#submit').click(); await page.waitForFunction(() => location.hash === '#new');
      await tab(page, 'results'); await page.getByRole('heading',{name:'项目调查报告',exact:true}).waitFor();
      assert.equal(calls.find(call=>call.action==='create').body.read_only,true);
      assert.equal(calls.find(call=>call.action==='create').body.model_plan.auditor,'deepseek');
      assert.equal(calls.find(call=>call.action==='create').body.project,'E:\\已经准备的工作副本');
      assert.equal(await page.getByRole('button',{name:'验收并应用到原项目',exact:true}).count(),0);
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
    });
    await scenario('live feed escapes tool output and remains usable on mobile', async page => {
      await tab(page, 'activity');
      activity.push({sequence:1,time:new Date().toISOString(),kind:'output',title:'工具结果',detail:'<img src=x onerror="window.injected=true"> '+ '长路径/'.repeat(100)});
      await page.locator('#activity-feed [data-sequence="1"]').waitFor();await page.locator('#activity-feed summary').click();
      assert.equal(await page.locator('#activity-feed img').count(),0);assert.equal(await page.evaluate(()=>!!window.injected),false);
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
      await page.screenshot({path:out+'/live-mobile.png',fullPage:true});
    }, '#a', true);
    await scenario('project folder selection fills the path and cancellation preserves the draft', async page => {
      pickerResult = { path: 'E:\\项目 中文' };
      await page.locator('#browse').click();
      await page.waitForFunction(() => document.querySelector('#project').value === 'E:\\项目 中文');
      pickerResult = { path: '' };
      await page.locator('#browse').click();
      await page.waitForFunction(() => !document.querySelector('#browse').disabled);
      assert.equal(await page.locator('#project').inputValue(), 'E:\\项目 中文');
      assert.deepEqual(calls.map(call => call.action), ['pick-project', 'pick-project']);
    });
    await scenario('project folder failure recovers the button and provides a path fallback', async page => {
      await page.locator('#project').fill('E:\\保留项目');
      pickerResult = { error: 'folder_picker_failed: exit status 1' };
      await page.locator('#browse').click();
      await page.locator('#form-error').waitFor();
      assert.match(await page.locator('#form-error').innerText(), /粘贴项目路径/);
      assert.equal(await page.locator('#browse').isEnabled(), true);
      assert.equal(await page.locator('#browse').innerText(), '选择文件夹');
      assert.equal(await page.locator('#project').inputValue(), 'E:\\保留项目');
      pickerResult = { path: 'E:\\重试项目' };
      await page.locator('#browse').click();
      await page.waitForFunction(() => document.querySelector('#project').value === 'E:\\重试项目');
      assert.equal(await page.locator('#form-error').isVisible(), false, 'successful retry retained a stale picker error');
      assert.deepEqual(calls.map(call => call.action), ['pick-project', 'pick-project']);
    });
    await scenario('desktop starts with focused requirement and collapsible complete native assignments', async page => {
      await page.locator('#task-search').waitFor();
      assert.equal(await page.getByText('正在读取任务…', { exact: true }).count(), 0, 'loaded history retained its stale loading message');
      assert.equal(await page.locator('#goal').isVisible(), true);
      assert.equal(await page.locator('#new-plan-frontend-model').isVisible(), false);
      assert.match(await page.locator('#assignment-summary').innerText(), /Harness/);
      await settings(page,'frontend');
      await page.locator('#new-plan-frontend-model').waitFor(); assert.equal(await page.locator('#new-plan-backend').inputValue(), 'codex');
      await page.locator('#agent-settings-toggle').click();
      await page.screenshot({ path: out + '/new-task-light.png', fullPage: true });
      const box = await page.locator('#goal').boundingBox(); assert.ok(box.y + box.height < 800);
    });
    await scenario('sidebar searches source paths and separates archived records', async page => {
      await page.locator('#task-search').fill('数据项目');
      assert.equal(await page.locator('[data-task-row="b"]').count(), 1); assert.equal(await page.locator('[data-task-row="a"]').count(), 0);
      await page.locator('#task-search').fill(''); await page.locator('#filter-archived').click();
      assert.equal(await page.locator('[data-task-row="c"]').count(), 1); assert.equal(await page.locator('[data-task-row="b"]').count(), 0);
    });
    await scenario('right click on inactive row opens its folder without navigating active task', async page => {
      await page.locator('[data-task-row="b"]').click({ button: 'right' });
      const opened = page.waitForResponse(response => response.url().endsWith('/tasks/b/open-project') && response.request().method() === 'POST');
      await page.getByRole('menuitem', { name: '打开原项目文件夹', exact: true }).click();
      await opened;
      assert.equal(calls.at(-1).id, 'b'); assert.equal(calls.at(-1).action, 'open-project'); assert.equal(new URL(page.url()).hash, '#a');
      await page.getByRole('heading', { name: '给记账页面增加月份筛选', exact: true }).waitFor();
    }, '#a');
    await scenario('archive is reversible metadata and waiting acceptance remains explicit', async page => {
      await page.locator('[data-task-row="b"] [data-task-menu]').click();
      await page.getByRole('menuitem', { name: '归档任务', exact: true }).click();
      assert.equal(rows.find(row => row.id === 'b').state, 'awaiting_acceptance'); assert.equal(new URL(page.url()).hash, '#a');
      await page.waitForFunction(() => document.querySelector('#archived-count').textContent.includes('1 待验收'));
      await page.locator('#filter-archived').click(); await page.locator('[data-task-row="b"] [data-task-menu]').click();
      await page.getByRole('menuitem', { name: '恢复到任务列表', exact: true }).click();
      assert.equal(rows.find(row => row.id === 'b').archived_at, undefined);
      await page.locator('#filter-active').click(); await page.reload(); await page.locator('[data-task-row="b"]').waitFor();
    }, '#a');
    await scenario('a rejected archive keeps the record in place and shows actionable feedback', async page => {
      await page.locator('[data-task-row="b"] [data-task-menu]').click();
      await page.getByRole('menuitem', { name: '打开原项目文件夹', exact: true }).click();
      await page.locator('#global-success').waitFor();
      actionFailure = true;
      await page.locator('[data-task-row="b"] [data-task-menu]').click();
      await page.getByRole('menuitem', { name: '归档任务', exact: true }).click();
      await page.locator('#global-error').waitFor();
      assert.match(await page.locator('#global-error').innerText(), /运行|执行/);
      assert.equal(rows[1].archived_at, undefined);
      assert.equal(await page.locator('[data-task-row="b"]').count(), 1);
      assert.equal(await page.locator('#global-success').isVisible(), false);
    }, '#a');
    await scenario('keyboard menu returns focus and cannot archive a running task', async page => {
      const row = page.locator('[data-task-row="a"] [data-task-select]'); await row.focus();
      await page.keyboard.press('Shift+F10'); const item = page.getByRole('menuitem', { name: '归档任务', exact: true }); await item.waitFor(); assert.equal(await item.isDisabled(), true);
      await page.keyboard.press('Escape'); assert.equal(await page.getByRole('menu').count(), 0); assert.equal(await row.evaluate(node => node === document.activeElement), true);
    });
    await scenario('task result, disclosure state and future plan survive polling', async page => {
      await tab(page, 'results');
      await settings(page,'backend'); await page.locator('#task-plan-backend-effort').waitFor();
      assert.equal(await page.locator('#task-plan-backend-effort').isDisabled(), false);
      await page.locator('#agent-settings-toggle').click();
      await page.locator('details[data-disclosure="evidence"] > summary').click();
      await page.locator('#task-summary .summary-stages > summary').click();
      const refreshed = page.waitForResponse(response => response.url().endsWith('/tasks') && response.request().method() === 'GET');
      rows[1].updated = new Date().toISOString(); await page.locator('#refresh').click();
      await (await refreshed).finished(); await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      assert.equal(await page.locator('details[data-disclosure="evidence"]').evaluate(node => node.open), true);
      assert.equal(await page.locator('#task-summary .summary-stages').evaluate(node => node.open), true, 'polling must not close the rules while the user is reading them');
      await page.screenshot({ path: out + '/result-light.png', fullPage: true });
    }, '#b');
    await scenario('appearance setting persists and dark controls remain readable', async page => {
      await page.locator('#theme').selectOption('dark'); await page.reload(); assert.equal(await page.locator('html').getAttribute('data-theme'), 'dark');
      await page.screenshot({ path: out + '/new-task-dark.png', fullPage: true });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.locator('#theme').selectOption('system');
      await page.emulateMedia({ colorScheme: 'light' }); const light = await page.locator('html').evaluate(node => getComputedStyle(node).backgroundColor);
      await page.emulateMedia({ colorScheme: 'dark' }); const dark = await page.locator('html').evaluate(node => getComputedStyle(node).backgroundColor);
      assert.notEqual(light, dark, 'system appearance ignored OS changes');
    });
    await scenario('mobile drawer offers the same task menu and stays within viewport', async page => {
      await page.locator('#sidebar-toggle').click(); await page.locator('[data-task-row="b"] [data-task-menu]').click();
      const menu = page.getByRole('menu'); await menu.waitFor(); const box = await menu.boundingBox(); assert.ok(box.x >= 0 && box.x + box.width <= 390 && box.y + box.height <= 844);
      await page.keyboard.press('Escape'); await page.locator('[data-task-row="b"] [data-task-select]').click();
      assert.equal(await page.locator('#sidebar-toggle').evaluate(node => node === document.activeElement), true, 'closing mobile navigation lost keyboard focus');
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/mobile.png', fullPage: true, animations: 'disabled' });
    }, '', true);
    await scenario('portrait background has adjustable transparency and slightly larger avatars', async page => {
      await page.locator('#background-settings > summary').click();
      assert.equal(await page.locator('#background-image').inputValue(), 'portrait');
      assert.equal(await page.locator('#background-transparency').inputValue(), '80');
      const before = await page.locator('.brand-avatar').boundingBox(); assert.ok(before.width >= 46 && before.width <= 54);
      const welcome = await page.locator('.welcome-avatar').boundingBox(); assert.ok(welcome.width >= 80 && welcome.width <= 90);
      const slider = page.locator('#background-transparency'); await slider.focus(); await page.keyboard.press('Home');
      assert.equal(await slider.inputValue(), '0');
      assert.equal(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').opacity), '1');
      await page.keyboard.press('End'); assert.equal(await slider.inputValue(), '100');
      assert.equal(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').opacity), '0');
      await page.keyboard.press('Home'); await page.keyboard.press('ArrowRight'); assert.equal(await slider.inputValue(), '1');
      await page.reload(); assert.equal(await page.locator('#background-transparency').inputValue(), '1');
      const stored = await page.evaluate(() => JSON.parse(localStorage.getItem('xiaobuding.background'))); assert.equal(stored.transparency, 1);
      await page.screenshot({ path: out + '/portrait-strong-light.png', animations: 'disabled' });
    });
    await scenario('background can be disabled and restored without changing theme or task input', async page => {
      await page.locator('#goal').fill('保留需求和模型选择'); await page.locator('#theme').selectOption('dark');
      await page.locator('#background-settings > summary').click(); await page.locator('#background-image').selectOption('none');
      assert.equal(await page.locator('#background-transparency').isDisabled(), true);
      assert.equal(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').backgroundImage), 'none');
      assert.equal(await page.locator('#goal').inputValue(), '保留需求和模型选择');
      await page.reload(); assert.equal(await page.locator('html').getAttribute('data-background'), 'none');
      assert.equal(await page.locator('#theme').inputValue(), 'dark');
      await page.locator('#background-settings > summary').click(); await page.locator('#background-image').selectOption('portrait');
      assert.equal(await page.locator('#background-transparency').isEnabled(), true);
      const image = await page.request.get(base + '/reliable/background.png'); assert.equal(image.status(), 200); assert.match(image.headers()['content-type'], /image\/png/);
      await page.screenshot({ path: out + '/portrait-dark.png', animations: 'disabled' });
    });
    await scenario('corrupted or blocked appearance storage keeps controls usable', async page => {
      await page.evaluate(() => localStorage.setItem('xiaobuding.background', '{broken'));
      await page.reload(); assert.equal(await page.locator('#background-image').inputValue(), 'portrait'); assert.equal(await page.locator('#background-transparency').inputValue(), '80');
      await page.evaluate(() => localStorage.setItem('xiaobuding.background', JSON.stringify({ image: 'url(https://invalid.invalid)', transparency: 180 })));
      await page.reload(); assert.equal(await page.locator('#background-image').inputValue(), 'portrait'); assert.equal(await page.locator('#background-transparency').inputValue(), '100');
      await page.addInitScript(() => { Storage.prototype.getItem = () => { throw new Error('blocked'); }; Storage.prototype.setItem = () => { throw new Error('blocked'); }; });
      await page.reload(); await page.locator('#background-settings > summary').click();
      await page.locator('#background-transparency').focus(); await page.keyboard.press('Home');
      assert.equal(await page.locator('#background-transparency').inputValue(), '0'); assert.equal(await page.locator('#goal').isEditable(), true);
    });
    await scenario('mobile background controls remain reachable and trap focus only on visible controls', async page => {
      await page.locator('#sidebar-toggle').click(); const summary = page.locator('#background-settings > summary'); await summary.focus(); await page.keyboard.press('Tab');
      assert.equal(await page.locator('.brand').evaluate(node => node === document.activeElement), true, 'closed settings included hidden controls in the focus trap');
      await summary.click(); const slider = page.locator('#background-transparency'); await slider.focus(); await page.keyboard.press('End'); await page.keyboard.press('Tab');
      assert.equal(await page.locator('.brand').evaluate(node => node === document.activeElement), true, 'expanded settings did not wrap to the first visible navigation link');
      const box = await slider.boundingBox(); assert.ok(box.x >= 0 && box.x + box.width <= 390);
      await slider.evaluate(node => { node.value = '80'; node.dispatchEvent(new Event('input', { bubbles: true })); });
      await page.keyboard.press('Escape'); await page.locator('#theme').evaluate(node => { node.value = 'dark'; node.dispatchEvent(new Event('change')); });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: out + '/portrait-mobile.png', animations: 'disabled' });
    }, '', true);
    await scenario('portrait reserves desktop subject space beside forms and task results', async page => {
      await page.locator('#theme').selectOption('dark');
      await page.locator('#background-transparency').evaluate(node => { node.value = '0'; node.dispatchEvent(new Event('input', { bubbles: true })); });
      for (const viewport of [{ width: 1440, height: 1050 }, { width: 2048, height: 1196 }, { width: 1181, height: 900 }]) {
        await page.setViewportSize(viewport);
        const pane = await page.locator('.main-pane').boundingBox(), panel = await page.locator('#setup').boundingBox();
        assert.ok(panel.x - pane.x <= 40, 'portrait form is still centered over the subject');
        assert.ok(panel.x + panel.width <= pane.x + pane.width * .58 + 1, 'form covers the reserved portrait region');
        assert.equal(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').backgroundSize), 'contain');
        assert.equal(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').backgroundRepeat), 'no-repeat');
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
        if (viewport.width === 2048) await page.screenshot({ path: out + '/portrait-clear-desktop.png', animations: 'disabled' });
      }
      await page.goto(base + '/reliable#b');
      if (await page.locator('body').getAttribute('data-workspace-width') !== 'wide') await page.locator('#workspace-width').click();
      const wideResults = await page.locator('#workspace').boundingBox(); await page.locator('#workspace-width').click();
      const pane = await page.locator('.main-pane').boundingBox(), results = await page.locator('#workspace').boundingBox();
      assert.ok(results.width < wideResults.width && results.x + results.width < wideResults.x + wideResults.width, 'portrait mode does not create space beside task results');
      await page.setViewportSize({ width: 390, height: 844 });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.goto(base + '/reliable'); await page.setViewportSize({ width: 1440, height: 1050 });
      await page.locator('#background-settings > summary').click(); await page.locator('#background-image').selectOption('none');
      const centered = await page.locator('#setup').boundingBox(), main = await page.locator('.main-pane').boundingBox();
      assert.ok(centered.x - main.x > 80, 'no-background layout unnecessarily reserves portrait space');
    });
    await scenario('local background image survives reload without uploading or changing task input', async page => {
      const uploads = []; page.on('request', request => { if (request.method() !== 'GET') uploads.push(request.url()); });
      await page.locator('#background-settings > summary').click();
      await page.locator('#goal').fill('保留当前需求'); await page.locator('#theme').selectOption('dark');
      const picker = page.waitForEvent('filechooser'); await page.locator('#background-pick').click();
      await (await picker).setFiles(source + 'assets/teahouse-background.png');
      await page.waitForFunction(() => document.documentElement.dataset.background === 'custom' && document.querySelector('#background-status').textContent.includes('已保存'));
      assert.equal(await page.locator('#goal').inputValue(), '保留当前需求'); assert.equal(calls.length, 0); assert.deepEqual(uploads, []);
      assert.match(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').backgroundImage), /blob:/);
      await page.reload(); await page.waitForFunction(() => document.documentElement.dataset.background === 'custom' && document.documentElement.style.getPropertyValue('--custom-background-image').includes('blob:'));
      assert.equal(await page.locator('#theme').inputValue(), 'dark');
      await page.locator('#background-settings > summary').click(); await page.screenshot({ path: out + '/custom-background-desktop.png', animations: 'disabled' });
    });
    await scenario('custom background can be replaced and restored after choosing none or default', async page => {
      await page.locator('#background-settings > summary').click();
      await page.locator('#background-file').setInputFiles(source + 'assets/teahouse-background.png');
      await page.waitForFunction(() => document.querySelector('#background-status').textContent.includes('已保存'));
      await page.locator('#background-image').selectOption('none'); assert.equal(await page.locator('#background-transparency').isDisabled(), true);
      await page.locator('#background-image').selectOption('portrait');
      assert.doesNotMatch(await page.locator('.main-pane').evaluate(node => getComputedStyle(node, '::before').backgroundImage), /blob:/);
      await page.locator('#background-image').selectOption('custom');
      await page.locator('#background-file').setInputFiles(source + 'assets/pudding-avatar.png');
      await page.waitForFunction(() => document.querySelector('#background-status').textContent.includes('已保存'));
      await page.reload(); await page.waitForFunction(() => document.documentElement.style.getPropertyValue('--custom-background-image').includes('blob:'));
      const width = await page.evaluate(async () => { const image = new Image(); image.src = document.documentElement.style.getPropertyValue('--custom-background-image').match(/"([^"]+)"/)[1]; await image.decode(); return image.naturalWidth; });
      assert.equal(width, fs.readFileSync(source + 'assets/pudding-avatar.png').readUInt32BE(16));
      await page.setViewportSize({ width: 390, height: 844 }); await page.locator('#sidebar-toggle').click(); await page.locator('#background-settings > summary').click();
      await page.locator('#background-pick').scrollIntoViewIfNeeded(); assert.equal(await page.locator('#background-pick').isVisible(), true);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    });
    await scenario('invalid background files keep the current image and offer a readable error', async page => {
      await page.locator('#background-settings > summary').click();
      for (const file of [{ name: 'bad.svg', mimeType: 'image/svg+xml', buffer: Buffer.from('<svg/>') }, { name: 'broken.png', mimeType: 'image/png', buffer: Buffer.from('not an image') }, { name: 'huge.png', mimeType: 'image/png', buffer: Buffer.alloc(20 * 1024 * 1024 + 1) }]) {
        await page.locator('#background-file').setInputFiles(file, { timeout: 15000 });
        await page.waitForFunction(() => document.querySelector('#background-status').getAttribute('data-error') === 'true');
        assert.equal(await page.locator('#background-image').inputValue(), 'portrait'); assert.equal(await page.locator('#goal').isEditable(), true);
      }
    });
    await scenario('unavailable image storage uses a temporary image without promising persistence', async page => {
      await page.addInitScript(() => { Object.defineProperty(window, 'indexedDB', { get() { throw new Error('blocked'); } }); });
      await page.reload(); await page.locator('#background-settings > summary').click();
      await page.locator('#background-file').setInputFiles(source + 'assets/teahouse-background.png');
      await page.waitForFunction(() => document.documentElement.dataset.background === 'custom' && document.querySelector('#background-status').textContent.includes('临时'));
      await page.reload(); assert.equal(await page.locator('#background-image').inputValue(), 'portrait');
      await page.evaluate(() => localStorage.setItem('xiaobuding.background', JSON.stringify({ image: 'custom', transparency: 50 })));
      await page.reload(); await page.waitForFunction(() => document.querySelector('#background-status').getAttribute('data-error') === 'true');
      assert.equal(await page.locator('#background-image').inputValue(), 'portrait'); assert.equal(await page.locator('#background-transparency').inputValue(), '50');
    });
    await scenario('opening assignments refreshes stale Codex effort metadata without losing choices', async page => {
      const id = 'gpt-6.1-sol', efforts = ['low', 'medium', 'high', 'xhigh', 'max', 'ultra']; let refreshes = 0;
      const native = { ...providers[0], model: id, catalog_status: '已读取原生目录；已配置模型未列出', models: [{ id, name: id + '（已配置，未验证）', efforts: [] }] };
      await page.route('**/api/v1/reliable/config', route => route.fulfill({ json: { providers: [native, providers[1]], default_plan: plan, demo_project: 'fixture' } }));
      await page.route('**/api/v1/reliable/models/refresh', route => { refreshes++; return route.fulfill({ json: { providers: [{ ...native, models: [{ id, name: 'GPT-6.1-Sol', efforts, default_effort: 'low' }] }, providers[1]], default_plan: plan } }); });
      await page.reload(); await page.locator('#goal').fill('保留我的需求和模型');
      await settings(page,'backend');
      await page.locator('#new-plan-backend-effort').selectOption('ultra');
      assert.equal(await page.locator('#new-plan-backend-model').inputValue(), id);
      for (const role of ['backend', 'auditor', 'reviewer']) assert.equal(await page.locator('#new-plan-' + role + '-effort option').count(), 7);
      assert.equal(await page.locator('#goal').inputValue(), '保留我的需求和模型');
      await page.locator('#agent-settings-toggle').click(); await page.locator('#agent-settings-toggle').click();
      assert.equal(await page.locator('#new-plan-backend-effort').inputValue(), 'ultra'); assert.equal(refreshes, 1);
    });
    fs.writeFileSync(out + '/results.json', JSON.stringify(results, null, 2)); console.log(JSON.stringify(results)); if (results.some(result => !result.passed)) process.exitCode = 1;
  } finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
})().catch(err => { console.error(err); server.close(); process.exitCode = 1; });
