'use strict';
const $ = id => document.getElementById(id);
const token = document.querySelector('meta[name="xbd-token"]').content;
const roles = [
  { id: 'auditor', title: '调查与规划', note: '先读当前项目，确定证据和修改范围' },
  { id: 'implementer', title: '实施修改', note: '唯一写代码的角色，在独立副本工作' },
  { id: 'reviewer', title: '独立审查', note: '使用新会话核对代码与真实检查结果' },
  { id: 'frontend', title: '前端界面', note: '负责页面与交互' },
  { id: 'backend', title: '后端功能', note: '负责接口与业务逻辑' },
];
const states = { preparing: '正在建立本地恢复点', queued: '准备开始', auditing: '调查与规划', implementing: '实施修改中', checking: '运行检查中', reviewing: '独立审查中', awaiting_acceptance: '等待你验收', accepted: '已验收并应用', blocked: '需要处理', interrupted: '运行已中断', exhausted: '三轮已用完', cancelling: '正在停止', cancelled: '已取消', applying: '正在应用' };
const tips = { preparing: '正在准备本地存档和独立工作副本，全程无需上传 GitHub。', queued: '正在准备独立工作副本。', auditing: '正在阅读当前项目，确定修改范围和计划。', implementing: '正在独立副本里完成目标，原项目保持不变。', checking: '程序正在实际运行约定的检查。', reviewing: '新会话正在核对修改、需求和检查日志。', awaiting_acceptance: '检查和独立审查已通过。查看结果后，再决定是否应用。', accepted: '经过核验的修改已应用到原项目。', blocked: '自动流程已暂停，请查看下面的原因。', interrupted: '运行中断，进度和已用轮次已保留。', exhausted: '本批的三轮自动尝试已用完。先查看遗留问题，再决定下一批目标。', cancelled: '任务已停止，副本和证据仍保留。', cancelling: '正在结束本任务的进程。', applying: '正在应用你已验收的修改。' };
const finalStates = ['accepted', 'cancelled', 'cancelling', 'applying', 'exhausted', 'awaiting_acceptance'];
let config = {}, tasks = [], selected = location.hash.slice(1), loading = false, creating = false, actionBusy = false;
let recoveringTask = '';
let detailSignature = '', evidenceTask = '', evidenceGeneration = 0;
let historySignature = '';
let proofSignature = '';
let selectionRevision = 0, refreshQueued = false;
const workspaceDetails = new Map();
const projectDisclosureState = new Map();
let summaryPagesWanted = 1, summaryNextOffset = -1, summaryTotal = 0, summaryQuery = '', searchTimer;
const workspaceReader = WorkbenchState.reader((path, options) => fetch('/api/v1/reliable/' + path, options));
const drafts = new Map();
states.inspected = '只读调查完成'; tips.inspected = '已完成项目阅读，调查报告已保存。'; finalStates.push('inspected');

function el(tag, cls, text) { const n = document.createElement(tag); if (cls) n.className = cls; if (text !== undefined) n.textContent = text; return n; }
function notice(id, message) { $(id).hidden = !message; $(id).textContent = message || ''; }
function friendly(message) {
  const raw = String(message || '');
  if (/project_dependency_pending/.test(raw)) return '请先完成并人工验收前置待办，再开始这一项。检查通过还不等于你已验收。';
  if (/project_dependency_cycle/.test(raw)) return '待办之间出现循环依赖，请取消相互等待的关系后再保存。';
  if (/project_dependency_invalid/.test(raw)) return '请选择已有待办作为前置项，不能重复、选择自身或超过 32 项。';
  if (/project_item_order/.test(raw)) return '待办顺序已经变化。请载入最新项目记录后重新排序，原输入仍保留。';
  if (/result_file_format/.test(raw)) return '此文件不能在页面显示。文本查看支持 UTF-8，最大 512 KiB；可打开工作副本查看原文件。';
  if (/preview_busy/.test(raw)) return '任务仍在执行，结束后可以打开本批成果和修改文件。';
  if (/audit_replan_changed/.test(raw)) return '副本在确认前又发生变化。请重新展开核对后再确认，原修改仍保留。';
  if (/audit_replan_scope/.test(raw)) return '当前变化超出最初授权范围，不能直接重新规划。请先核对原项目和范围，已有文件保留。';
  if (/audit_replan_/.test(raw)) return '本批暂不能按当前副本重新规划，请查看状态、证据与原始诊断；不会清零轮次或扩大授权。';
  if (/project_revision_conflict/.test(raw)) return '项目记录已在其他窗口或阶段更新。你的输入已保留，请先刷新项目记录后再保存。';
  if (/project_limit/.test(raw)) return '项目待办或记忆达到保存上限。已有记录保留，请精简本次新增内容后重试。';
  if (/project_memory_invalid/.test(raw)) return '这条项目记忆尚未保存。请选择设计决定、项目约束或待解决事项，并缩短内容后重试。';
  if (/project_item_invalid/.test(raw)) return '项目待办尚未保存。请填写具体目标，验收标准可选；完成状态由实际核验记录确定。';
  if (/project_item_(unavailable|conflict)/.test(raw)) return '这项待办已变化或已安排到其他批次。请刷新项目记录，已有任务和输入保留。';
  if (/storage_boundary/.test(raw)) return '临时文件目录的实际位置无法安全确认，本次没有执行清理。请检查数据目录是否被移动或链接。';
  if (/local_checkpoint_(evidence|receipt)/.test(raw)) return '已验收成果的证据记录不完整或发生变化，暂不能补存档。原项目与工作副本均保留，请查看执行记录中的具体原因。';
  if (/local_checkpoint_pending/.test(raw)) return '上次验收的本地存档尚未完成。请点击“完成本地存档”；不会重新修改代码或消耗轮次。';
  if (/local_checkpoint_/.test(raw)) return '本地存档暂未完成，可能有其他手动修改或 Git 正在操作。文件均已保留，请查看执行记录，处理后重试；不会夹带手动改动。';
  if (/attachment_/.test(raw)) return /size|large|limit/.test(raw) ? '附件超过大小或数量限制。最多 10 个文件，单个及总计不超过 100 MB；已选择的文件保留，请移除或换小一点的附件。' : '附件未能保存或读取。已有文字和文件选择已保留，请重试上传；也可移除这个附件后继续。';
  const recoveryErrors = {
    conflict: '另一个未完成任务正在使用这个项目或重叠目录。副本和已有修改已保留，请先处理那条任务，再点击“检查副本并继续”。',
    project_conflict: '另一个未完成任务正在使用这个项目或重叠目录。副本和已有修改已保留，请先处理那条任务，再点击“检查副本并继续”。',
    exhausted: '本批已用完三轮，不能继续修改。副本和证据已保留，请先查看结果，再决定下一批目标。',
    read_only: '只读任务没有可继续修改的副本。调查报告已保留，可在调查完成后明确提出下一批修改需求。',
    archived: '任务已归档，副本和证据已保留。先恢复到任务列表，再明确选择继续。',
    unavailable: '这条任务当前不能从停止状态继续。副本和记录已保留，请刷新状态后查看可用操作。',
    ineligible: '这条任务当前不能从停止状态继续。副本和记录已保留，请刷新状态后查看可用操作。',
    state: '任务状态已变化。副本和记录已保留，请刷新状态后查看可用操作。',
    pending: '副本仍在检查中，请等待检查完成。已有文件和轮次均保留。',
    running: '任务进程仍在运行或停止中。副本已保留，请等进程完全结束后再继续。',
    active: '任务进程仍在运行或停止中。副本已保留，请等进程完全结束后再继续。',
    stopping: '任务进程仍在停止中。副本已保留，请等进程完全结束后再继续。',
    process_unknown: '尚不能确认之前的任务进程已完全停止。副本和修改已保留，请先检查执行记录并确认进程结束，再重试。',
    original_changed: '停止期间原项目发生变化。副本和部分修改已保留，请先核对原项目与任务恢复点，处理差异后再重试。',
    candidate_missing: '没有找到任务原有的工作副本。已有记录和文件会保留，请检查副本是否被移动，不要创建新副本覆盖原路径。',
    candidate_invalid: '工作副本的归属或身份无法核实。已有修改已保留，请检查原副本路径和任务记录，再重试。',
    source_changed: '保留的副本包含超出本批授权范围的修改。文件已保留，请先核对修改范围与任务记录，再重试。',
    scope: '保留的副本包含超出本批授权范围的修改。文件已保留，请先核对修改范围与任务记录，再重试。',
    resource_changed: '保留副本的输入资源与冻结记录不一致。原项目和副本已保留，请检查资源路径与版本，再重试。',
    resource_invalid: '保留副本的输入资源与冻结记录不一致。原项目和副本已保留，请检查资源路径与版本，再重试。',
    evidence_invalid: '恢复所需的任务证据缺失或已变化。副本和修改已保留，请检查任务证据与原记录，再重试。',
  };
  const renewalErrors = {
    ineligible: '这条任务当前不符合保留修改开启下一批的条件。已有副本和记录已保留，请刷新状态后查看可用操作。',
    conflict: '另一个未完成任务正在使用这个项目或重叠目录。已有修改和要求已保留，请先处理那条任务，再明确重试。',
    source_changed: '原项目或上一批副本在检查期间发生变化。两份副本和已有修改已保留，请先核对变化，再明确重试。',
    evidence_invalid: '继承上一批修改所需的证据缺失或已变化。副本和要求已保留，请检查证据后再重试。',
    parent_active: '上一批仍有任务进程或只读回答在执行。副本和要求已保留，请等执行完全结束后再重试。',
  };
  const batchErrors = {
    next_unavailable: renewalErrors.ineligible,
    project_conflict: renewalErrors.conflict,
    parent_changed: renewalErrors.source_changed,
    parent_active: renewalErrors.parent_active,
    seed_invalid: '上一批修改的源码或继承证据尚未通过检查。两份副本和要求已保留，请查看执行记录并处理原因后继续。',
    original_changed: '原项目已偏离上一批保存的恢复点。两份副本和已有修改已保留，请先核对原项目变化后继续。',
    successor_exists: '本批已经开启下一批，已有修改和记录保留。请刷新后点击“打开下一批”，继续在那一批提出要求。',
  };
  const batch = raw.match(/batch_([a-z_]+):/);
  if (batch && batchErrors[batch[1]]) return batchErrors[batch[1]];
  renewalErrors.active = '上一批修改仍在继承检查中。已有副本和要求已保留，请等检查完成后再发送修改要求。';
  const renewal = raw.match(/renewal_([a-z_]+):/);
  if (renewal) return renewalErrors[renewal[1]] || '尚未通过上一批修改的继承检查。副本、证据和要求已保留，请查看原因并处理后再继续。';
  const recovery = raw.match(/recovery_([a-z_]+):/);
  if (recovery) { const code = Object.keys(recoveryErrors).find(key => recovery[1] === key || recovery[1].startsWith(key + '_')); return recoveryErrors[code] || '保留副本尚未通过恢复检查。文件和已用轮次已保留，请查看执行记录中的原因，处理后再明确重试。'; }
  if (/settings_stage_changed/.test(raw)) return '任务阶段刚刚变化。你的改选已保留，请载入最新安排后保存。';
  if (/message_invalid/.test(raw)) return '请填写问题或要求；内容最多约 4000 个汉字。';
  if (/message_conflict/.test(raw)) return '这条消息的发送记录发生冲突。请修改内容后重新发送。';
  if (/message_limit/.test(raw)) return '本批对话记录已达到容量。可在下一批继续，已有记录会保留。';
  if (/batch_exhausted/.test(raw)) return '本批已用完三轮自动尝试。可以查看遗留问题，再点击“保留修改，开始下一批”，沿用已有修改继续。';
  if (/new_batch_unavailable|requirement_unavailable/.test(raw)) return '请先完成当前调查或验收；当前仍可继续提问，已有记录会保留。';
  if (/Application restarted before delivery/.test(raw)) return '程序重启时这条问题尚未回答完。记录已保留，点“重新提问”即可继续。';
  if (/Question cancelled by the user/.test(raw)) return '你已停止这次回答，已有项目报告和修改记录保留。';
  if (/this repository tracks more than 30000 files/.test(raw)) return '这条任务上次被旧版的项目文件数量限制暂停。新版已支持题库资源按需取出，点击“处理后继续”重新检查即可；已有记录和轮次会保留。';
  if (/resource_changed/.test(raw)) return '输入资源与冻结的版本不一致。小布丁已暂停，原项目文件保留；请检查资源变化后再继续。';
  if (/resource_size/.test(raw)) return '这次需要的输入资源超过读取上限。请缩小本次验证所需的数据范围，原题库无需删除。';
  if (/resource_request/.test(raw)) return '这次需要的输入资源尚未准备好，或请求次数已达上限。请查看执行记录中的具体路径，补充要求后再继续。';
  if (/resource_inventory|resource_path/.test(raw)) return '工作副本的资源记录或路径需要检查。已有文件已保留，请查看执行记录；小布丁不会直接覆盖它们。';
  if (/source_snapshot_too_large|source snapshot exceeds/.test(raw)) return '这个项目的可修改源码仍超过大小上限。题库图片和数据已支持按需读取，无需删除；可先只读了解项目，或缩小本次修改范围。';
  if (/provide an absolute project folder path and a concrete goal/.test(raw)) return '请在页面上方选择项目文件夹，并填写这次要完成的目标。';
  if (/folder_picker_timeout/.test(raw)) return '选择文件夹已超时。请再次点击“选择文件夹”，或直接粘贴项目路径。';
  if (/folder_picker_failed/.test(raw)) return '文件夹选择窗口未能正常打开。请重试，也可以直接粘贴项目路径。';
  if (/folder_unavailable/.test(raw)) return '这个文件夹尚未创建，或已被移动。请先检查任务记录中的项目路径。';
  if (/folder_open_failed/.test(raw)) return '暂时无法打开文件管理器，请重新打开小布丁后重试。';
  if (/task_running/.test(raw)) return '任务仍在执行。请等它结束，或先停止任务；已有副本和记录会保留。';
  if (/task_archived/.test(raw)) return '这条任务已归档，请先恢复到任务列表再继续。';
  const localErrors = {
    boundary: '修改任务请选具体项目的源码文件夹，不能使用工具总目录或账号目录；只想查看普通总目录或已有工作副本，请勾选“只读了解项目”。',
    directory: '没有找到这个项目文件夹，请重新选择。',
    parent: '这个文件夹属于一个已有项目，请选择该项目的根文件夹。',
    staged: '发现已手动暂存的修改，已完整保留。请先处理这些暂存内容；若没有手动操作过，可把任务记录发给我检查。',
    external_dirty: '这是你原来已有的 Git 项目，其中有未提交修改。为保留你的版本管理安排，本次没有自动提交，请先保存这些修改。普通文件夹由小布丁自动存档。',
    private: '项目规则要求纳入本应排除的凭据或生成文件，已停止存档。请检查忽略规则，凭据文件会保留在原处。',
    nested_git: '文件夹里还有其他 Git 项目，请单独选择这次要修改的项目。',
    size: '这个文件夹超出当前支持的源码大小，请选择具体项目；依赖和常见构建产物会自动排除。',
    link: '项目包含暂不支持的文件链接或子模块，请先选择实际源码所在的项目文件夹。',
    worktree: '这不是小布丁管理的原项目，请选择原项目文件夹。',
    changed: '准备期间原项目发生变化，已暂停。请处理当前修改后再继续。',
    git: '本地 Git 工具暂时不可用，请重新打开小布丁后重试。',
    git_invalid: '项目已有 Git 存档，但目前无法读取，已保留原文件。请把任务记录发给我检查。',
    incomplete: '本地存档初始化未完成，原文件已保留。请把任务记录发给我恢复，不需要上传 GitHub。',
  };
  const local = raw.match(/local_snapshot_([a-z_]+):/);
  if (local && localErrors[local[1]]) return localErrors[local[1]];
  if (/automatic_checks_missing/.test(raw)) return '还没有可用的真实检查。程序会要求补齐测试，最多共三轮；没有检查结果就不能验收。';
  if (/model plan changed/.test(raw)) return '其他窗口已更新安排。请载入最新安排后再保存。';
  if (/selected model is not present/.test(raw)) return '所选模型不在这个工具的当前目录里。请刷新模型列表，再重新选择。';
  if (/reasoning effort is unsupported/.test(raw)) return '当前模型不支持这个思考强度，请选择自动或列表内支持的强度。';
  if (/scopes overlap/.test(raw)) return '调查给出的前后端修改范围重叠。请明确共享文件由谁修改，再继续。';
  if (/audit did not establish/.test(raw)) return '调查尚未确定每个工种的修改范围，请明确前后端目录后继续。';
  if (/No checks detected/.test(raw)) return '没有找到自动检查。请展开“检查命令与修改范围”补充检查命令，或把项目路径发给我帮你配置。';
  if (/unfinished task/.test(raw)) return '这个项目已有未完成任务。请从左侧继续处理那一条任务。';
  if (/uncommitted|not clean|working tree.*clean|dirty/.test(raw)) return '项目里有未保存到 Git 的修改。请先保存现有成果，再开始新任务。';
  if (/connection.*unavailable|no configured runner/.test(raw)) return '所选工具目前不可用，请查看顶部连接状态，修复后再继续。';
  if (/Failed to fetch|NetworkError/.test(raw)) return '暂时连接不到小布丁。请重新双击桌面快捷方式，再点刷新。';
  return raw || '操作未完成，请查看任务记录。';
}
async function api(path, body) {
  const response = await fetch('/api/v1/reliable/' + path, { method: body === undefined ? 'GET' : 'POST', headers: body === undefined ? {} : { 'Content-Type': 'application/json', 'X-XBD-Token': token }, body: body === undefined ? undefined : JSON.stringify(body) });
  const raw = await response.text();
  let data;
  try { data = JSON.parse(raw); } catch { throw Error(response.status === 403 ? '页面权限已失效，请刷新页面。' : raw); }
  if (!response.ok) throw Error(friendly(data.error));
  return data;
}
function roleTitle(id, domain) { return roles.find(role => role.id === (domain || id))?.title || id; }
function selectTask(id) { selected = id; selectionRevision++; location.hash = id; detailSignature = ''; proofSignature = ''; notice('global-success', ''); notice('global-error', ''); DesktopUI.closeMenu(false); DesktopUI.drawer(false); render(); $('main-content').scrollTop = 0; void refresh(); }
function section(parent, title, text) { const s = el('section', 'detail-section'); s.append(el('h3', '', title)); if (text) s.append(el('p', '', text)); parent.append(s); return s; }
function button(text, cls, action) { const b = el('button', cls, text); b.type = 'button'; b.onclick = action; return b; }

function planFields(parent, prefix, plan, disabled, onChange) {
  PlanUI.render(parent, prefix, plan, disabled, onChange);
}
function readPlan(prefix) { return { ...$(prefix)._plan }; }
function renderNewPlan(plan) {
  PlanUI.modePicker($('work-mode'), plan, mode => { try { renderNewPlan(AssistantState.changeWorkMode(readPlan('new-plan'), mode, config.providers || [], $('read-only').checked)); } catch (error) { notice('form-error', friendly(error.message)); } });
  planFields($('new-plan'), 'new-plan', plan, false, () => { $('assignment-summary').textContent = DesktopUI.planSummary(readPlan('new-plan')); AssistantUI.renderNew(); });
  $('assignment-summary').textContent = DesktopUI.planSummary(readPlan('new-plan'));
  AgentSettings.sync(tasks.find(item => item.id === selected));
  AssistantUI.renderNew();
}
function renderConnections() {
  $('connections').replaceChildren();
  for (const p of config.providers || []) {
    const item = el('span', 'connection' + (p.available ? '' : ' unavailable'));
    item.append(el('span', 'full-connection', p.name + (p.available ? ' · 已发现配置' : ' · 待连接')));
    item.append(el('span', 'compact-connection', (p.id === 'codex' ? 'Codex · ChatGPT' : 'Harness · DeepSeek') + (p.available ? '' : '（待连接）')));
    item.title = p.detail + '；模型：' + p.model; $('connections').append(item);
  }
}
function renderHistory() {
  DesktopUI.counts(tasks);
  const signature = JSON.stringify(tasks.map(t => [t.id, t.goal, t.project, t.state, t.round, t.archived_at])) + selected + DesktopUI.filterKey() + actionBusy + summaryNextOffset;
  if (signature === historySignature) return;
  historySignature = signature;
  const list = $('task-list'), scroll = list.scrollTop;
  list.replaceChildren();
  const visible = DesktopUI.visibleTasks(tasks);
  if (!visible.length) list.append(el('p', 'empty-history', $('task-search').value ? '没有找到匹配的任务。' : DesktopUI.filterKey().startsWith('archived') ? '归档的任务会保留在这里。' : '还没有任务。从一个想法开始吧。'));
  function taskRow(task) {
      const row = el('div', 'task-row'); row.dataset.taskRow = task.id;
      const choose = button('', 'task-button', () => selectTask(task.id)); choose.dataset.taskSelect = task.id; choose.dataset.task = task.id;
      const more = button('', 'icon-button task-menu-button', event => DesktopUI.showMenu(task.id, event.currentTarget));
      more.onclick = event => DesktopUI.showMenu(task.id, event.currentTarget); more.dataset.taskMenu = task.id; more.setAttribute('aria-label', '更多操作：' + task.goal); more.setAttribute('aria-haspopup', 'menu'); more.append(DesktopUI.icon('more'));
      row.append(choose, more);
    row.className = 'task-row' + (task.id === selected ? ' active' : '');
    choose.setAttribute('aria-pressed', String(task.id === selected)); choose.title = task.goal;
    choose.replaceChildren(el('span', 'task-goal', ModelState.taskTitle(task)));
    const meta = el('span', 'task-meta'), dot = el('span', 'status-dot' + (['queued','preparing','auditing','implementing','checking','reviewing'].includes(task.state) ? ' live' : task.state === 'awaiting_acceptance' || task.state === 'blocked' ? ' warning' : task.state === 'accepted' ? ' good' : ''));
    meta.append(dot, el('span', 'task-project', DesktopUI.folderName(task.project)), el('span', '', states[task.state] || task.state), el('span', 'task-round', task.round + '/3'));
    choose.append(meta); row.querySelector('[data-task-menu]').disabled = actionBusy;
    return row;
  }
  for (const group of WorkbenchState.groups(visible)) {
    const section = el('details', 'project-group'); section.dataset.projectGroup = group.id;
    section.open = projectDisclosureState.has(group.id) ? projectDisclosureState.get(group.id) : true;
    section.ontoggle = () => { if (section.isConnected) projectDisclosureState.set(group.id, section.open); };
    const heading = el('summary', 'project-group-heading', DesktopUI.folderName(group.project)); heading.title = group.project;
    heading.append(el('small', '', group.tasks.length + ' 批')); section.append(heading, taskRow(group.current));
    const older = group.tasks.filter(task => task.id !== group.current.id);
    if (older.length) {
      const batches = el('details', 'project-batch-history'); batches.dataset.projectHistory = group.id;
      batches.open = projectDisclosureState.has('history:' + group.id) ? projectDisclosureState.get('history:' + group.id) : older.some(task => task.id === selected);
      batches.ontoggle = () => { if (batches.isConnected) projectDisclosureState.set('history:' + group.id, batches.open); };
      batches.append(el('summary', '', '历史批次 · ' + older.length));
      for (const task of older) batches.append(taskRow(task)); section.append(batches);
    }
    list.append(section);
  }
  if (summaryNextOffset >= 0) list.append(button('载入更多历史批次（共 ' + summaryTotal + ' 批）', '', () => { if (loading) return; summaryPagesWanted++; historySignature = ''; void refresh(); }));
  list.scrollTop = scroll;
}
function render() {
  renderHistory();
  const task = tasks.find(item => item.id === selected);
  $('workspace-loading').hidden = !selected || !!task && !task.summary_only;
  if (selected && (!task || task.summary_only)) {
    AgentSettings.setLoading(); AssistantUI.setLoading(); TaskActivity.sync(null); TaskConversation.sync(null); ProjectUI.sync(null); $('task-more').hidden = true;
    $('setup').hidden = true; $('workspace').hidden = true;
    $('view-title').textContent = '读取项目'; $('view-project').textContent = task ? DesktopUI.folderName(task.project) : '';
    return;
  }
  WorkspaceUI.prepare(task);
  TaskActivity.sync(task);
  TaskConversation.sync(task);
  ProjectUI.sync(task);
  DesktopUI.syncMenu();
  $('view-title').textContent = task ? '任务' : '新建任务';
  $('view-project').textContent = task ? DesktopUI.folderName(task.project) : '';
  $('task-more').hidden = !task; $('task-more').disabled = actionBusy;
  $('setup').hidden = !!task; $('workspace').hidden = !task;
  if (!task) { AgentSettings.sync(null); AssistantUI.sync(null); ContextUI.sync(null); WorkspaceUI.sync(null); return; }
  renderPlanEditor(task);
  AgentSettings.sync(task);
  AssistantUI.sync(task);
  ContextUI.sync(task);
  const signature = JSON.stringify(task) + actionBusy;
  if (detailSignature === signature) { WorkspaceUI.sync(task); return; }
  detailSignature = signature;
  renderSummary(task); WorkspaceUI.sync(task); renderProof(task);
}
function retainedRenewalEligible(task) {
  return !!task && !task.local_checkpoint && !task.archived_at && !task.read_only && !task.recovery_pending && !task.seed_pending && !task.successor_task_id && task.round >= 3 && !!task.worktree && !!task.base && !!task.baseline && ['exhausted', 'cancelled', 'blocked', 'interrupted'].includes(task.state);
}
function renderSummary(task) {
  const root = $('task-summary');
  const open = root.dataset.task === task.id ? [...root.querySelectorAll('details[open]')].map(node => node.className) : [];
  root.dataset.task = task.id; root.replaceChildren();
  const panel = el('article', 'surface workspace-summary'), head = el('div', 'detail-title summary-heading');
  const recoveryRequired = !!task.recovery_pending;
  const checkpointPending = !!task.local_checkpoint && task.state !== 'accepted';
  const stageResume = ModelState.stageResumable(task) && !/stage_resume_source_changed/.test(task.reason || '');
  const recovering = recoveryRequired && ['queued', 'preparing', 'auditing', 'implementing', 'checking', 'reviewing'].includes(task.state) || recoveringTask === task.id;
  const renewable = retainedRenewalEligible(task), renewing = TaskConversation.isNextBatchPending(task.id);
  const seedRequired = !!task.seed_pending, seeding = seedRequired && ['queued', 'preparing', 'auditing', 'implementing', 'checking', 'reviewing'].includes(task.state);
  const recoverable = !task.successor_task_id && !task.archived_at && !task.read_only && task.state === 'cancelled' && task.round < 3;
  if (task.archived_at) {
    const banner = el('div', 'archive-banner'), text = el('div');
    text.append(el('strong', '', '已归档'), el('p', '', task.state === 'awaiting_acceptance' ? '仍待你验收。恢复后可查看并应用，归档不会应用修改。' : '项目和证据已保留。恢复到任务列表后，可以继续处理。'));
    banner.append(DesktopUI.icon('archive'), text); panel.append(banner);
  }
  head.append(el('h1', 'summary-title', ModelState.taskTitle(task)), el('span', 'badge' + (['accepted', 'awaiting_acceptance'].includes(task.state) ? ' good' : ['blocked', 'exhausted', 'interrupted'].includes(task.state) ? ' bad' : ''), states[task.state] || task.state));
  panel.append(head, el('p', 'path', task.project));
  if (task.attachments?.length) {
    const files = el('details', 'summary-attachments'); files.open = open.includes('summary-attachments'); files.append(el('summary', '', '本批附件 · ' + task.attachments.length + ' 个')); AttachmentUI.historical(files, task.attachments); panel.append(files);
  }
  if (task.read_only) panel.append(el('p', 'inspection-badge', '只读了解 · 直接读取所选文件夹 · 不创建工作副本'));
  const active = [...(task.runs || [])].reverse().find(run => ['running', 'preparing'].includes(run.status));
  const domains = !task.model_plan?.mode ? ['implementer'] : task.model_plan.mode === 'fullstack' ? ['frontend', 'backend'] : [task.model_plan.mode];
  const preparation = task.auto_snapshot ? 1 : 0;
  const steps = task.read_only ? ['阅读项目', '整理报告'] : [...(preparation ? ['本地存档'] : []), '调查', ...domains.map(d => roleTitle(d)), '实际检查', '独立审查', '你来验收'];
  const phase = recovering || seedRequired ? -1 - preparation : { inspected: 2, queued: -preparation, preparing: -preparation, auditing: 0, implementing: 1 + Math.max(0, domains.indexOf(active?.domain || 'implementer')), checking: 1 + domains.length, reviewing: 2 + domains.length, awaiting_acceptance: 3 + domains.length, applying: 3 + domains.length, accepted: 4 + domains.length }[task.state];
  const progress = el('ol', 'progress'), index = phase === undefined ? -1 : phase + preparation;
  progress.style.gridTemplateColumns = 'repeat(' + steps.length + ', minmax(0,1fr))';
  steps.forEach((text, i) => progress.append(el('li', i < index ? 'done' : i === index ? 'current' : '', text)));
  const status = el('div', 'current-status summary-status');
  const current = seedRequired ? seeding ? '正在检查并继承上一批修改' : '上一批修改的继承检查尚未完成。副本和记录保留，请处理下方原因后继续。' : task.successor_task_id ? '本批已开启下一批。已有修改和未解决问题继续保留，请打开下一批跟进。' : renewing ? '正在建立下一批，检查并继承已有修改。' : recovering ? '正在检查保留的副本' : recoveryRequired && task.state !== 'cancelling' ? '恢复检查尚未完成。副本和已用轮次保留，请通过任务操作重新检查。' : task.read_only && task.state === 'inspected' ? '只读调查完成。查看下方调查结论、事实和待确认项。' : tips[task.state] || states[task.state];
  $('current-announcement').textContent = current;
  const shortStatus = seedRequired ? '正在检查上一批修改，已有记录保留。' : task.successor_task_id ? '本批已开启下一批，可打开下一批继续。' : recovering ? '正在检查保留的副本。' : ({ awaiting_acceptance: '先查看成果和未完成项，再验收应用。', accepted: '修改已应用，可继续提出下一批要求。', inspected: '报告已保存，可继续提问或提出修改。', blocked: '流程已暂停，请查看原因后继续。', interrupted: '进度已保留，可处理后继续。', cancelled: '任务已停止，副本和记录保留。', exhausted: '本批尝试已结束，可以继续下一批。' })[task.state] || current;
  status.append(el('strong', '', active ? active.provider_name + ' · ' + roleTitle(active.role, active.domain) + ' · ' + active.model : shortStatus)); status.setAttribute('aria-busy', String(recovering || renewing || seeding));
  const statusDetails = el('div', 'summary-status-details'); statusDetails.append(el('p', '', current));
  if (checkpointPending) { status.querySelector('strong').textContent = '已验收，正在衔接本地存档'; statusDetails.append(el('p', '', '重试只完成上次验收的本地存档，不重新修改代码，也不消耗轮次。')); }
  if (task.state === 'accepted' && task.local_checkpoint?.complete) { const checkpoint = el('p', 'hint', '已自动保存本地 Git 版本 ' + task.local_checkpoint.commit.slice(0, 8) + '。可直接提出下一批需求，无需上传 GitHub。'); panel.append(checkpoint); }
  if (task.auto_snapshot && task.base) statusDetails.append(el('p', '', '本地恢复点已准备，存档保存在这台电脑上。'));
  if (active) statusDetails.append(el('p', '', '当前实际执行：' + active.provider_name + ' · ' + roleTitle(active.role, active.domain)), el('small', '', active.model + (active.effort ? '；思考：' + active.effort : '') + '；使用第 ' + active.plan_revision + ' 版安排。之后改选不会打断这次执行。'));
  else if (task.state === 'checking') statusDetails.append(el('p', '', '这一步由程序执行检查，不会把模型口头结论当成通过。'));
  const renewalBudget = '本批已用 ' + task.round + '/3 轮。项目可以继续修改；下一批从 0/3 轮开始，保留已有修改和未解决问题。';
  const recoveryBudget = '已用 ' + task.round + '/3 轮，还可修改 ' + Math.max(0, 3 - task.round) + ' 轮。' + (task.round >= 3 ? '本批已用完三轮，不能继续自动返工；可保留修改开启下一批，副本和证据仍保留。' : '部分修改和原有安排保留，检查通过后重新调查，再进入下一轮。');
  statusDetails.append(el('p', '', task.read_only ? '此任务只提供调查报告，不修改代码、不执行测试，也不进入验收应用。' : seedRequired ? '新批次已用 ' + task.round + '/3 轮。继承检查通过后重新调查，再实施修改、实际检查和独立审查；验收后才应用到原项目。' : renewable || task.successor_task_id ? renewalBudget : task.state === 'cancelled' || recoveryRequired || recovering ? recoveryBudget : task.round ? '本批已自动尝试 ' + task.round + '/3 轮。前后端共用本批轮次，下一批需求可以继续跟进。' : '本批尚未开始修改。最多首次尝试加两次自动返工，通过就停止。'));
  status.append(el('small', 'summary-budget', task.read_only ? '只读调查 · 不消耗自动返工轮次' : '本批自动尝试 ' + task.round + '/3 · 可持续开启下一批'));
  progress.setAttribute('aria-label', '本批协作阶段');
  const actions = el('div', 'actions' + (recoverable || recovering || renewable || renewing || task.successor_task_id ? ' recovery-actions' : ''));
  const add = (text, cls, action, disabled = false) => { const b = button(text, cls, () => perform(task.id, action)); b.disabled = actionBusy || disabled || AssistantState.blocksDispatch(AssistantUI.hasUnsaved(task.id), action); actions.append(b); };
  if (task.audit || task.review || task.checks?.length) actions.append(button(task.read_only ? '查看调查报告' : '查看成果与检查', task.state === 'awaiting_acceptance' ? 'primary' : '', () => WorkspaceUI.show('results')));
  if (task.successor_task_id) { const link = el('a', 'batch-link', '打开下一批'); link.href = '#' + encodeURIComponent(task.successor_task_id); actions.append(link); }
  else if (renewable || renewing) { const begin = button(renewing ? '正在建立下一批…' : '保留修改，开始下一批', stageResume ? '' : 'primary', () => TaskConversation.startNextBatch(task.id)); begin.id = 'begin-next-batch'; begin.disabled = actionBusy || TaskConversation.isPending(task.id) || TaskConversation.isNextBatchBlocked(task.id); actions.append(begin); }
  if (task.archived_at) add('恢复到任务列表', 'primary', 'unarchive');
  if (!task.archived_at && task.state === 'awaiting_acceptance') add(task.execution_active ? '等待当前执行结束' : '验收并应用到原项目', '', 'accept', task.execution_active);
  if (!task.archived_at && !task.read_only && recovering) add('正在检查保留的副本', 'primary', 'recover-cancelled', true);
  else if (recoverable) add('检查副本并继续', 'primary', 'recover-cancelled');
  if (!task.successor_task_id && !task.archived_at && !recovering && ['blocked', 'interrupted'].includes(task.state) && (task.round < 3 || checkpointPending || stageResume)) add(checkpointPending ? '完成本地存档' : stageResume ? '继续未完成阶段' : '处理后继续', 'primary', 'resume');
  if (task.worktree) add('查看工作副本', '', 'open');
  if (!checkpointPending && !task.successor_task_id && !task.archived_at && !['accepted', 'cancelled', 'cancelling', 'applying', 'exhausted'].includes(task.state)) add('停止此任务', 'danger', 'cancel');
  panel.append(actions);
  const stages = el('details', 'summary-stages'); stages.open = open.includes('summary-stages'); stages.append(el('summary', '', '阶段与规则'), progress, statusDetails, el('p', 'updated', '更新于 ' + new Date(task.updated).toLocaleString('zh-CN')));
  const statusRow = el('div', 'summary-status-row'); statusRow.append(status, stages); panel.insertBefore(statusRow, actions);
  if (task.reason && !['accepted', 'awaiting_acceptance'].includes(task.state)) WorkbenchUI.guidance(panel, task);
  WorkbenchUI.execution(panel, task);
  root.append(el('h2', 'context-caption', '本批概览'), panel);
}

function renderPlanEditor(task) {
  task = ModelState.settingsTask(task);
  const editor = $('task-plan-editor');
  const draft = ModelState.reconcileDraft(drafts.get(task.id), task, config.default_plan);
  drafts.set(task.id, draft);
  const editable = task.settings_editable;
  $('task-assignment-summary').textContent = DesktopUI.planSummary(draft.plan);
  if (editor.dataset.task === task.id && editor.dataset.scope === task.settings_scope && editor.dataset.editable === String(editable) && (draft.dirty || editor.dataset.revision === String(task.plan_revision || 0))) {
    updateDraftHint(task, draft); return;
  }
  editor.replaceChildren(); editor.dataset.task = task.id; editor.dataset.scope = task.settings_scope; editor.dataset.revision = String(task.plan_revision || 0); editor.dataset.editable = String(editable);
  editor.append(el('h2', '', task.settings_scope === 'future' ? '后续追问与下一批' : '后续阶段的安排'), el('p', 'hint', !editable ? task.successor_task_id ? '本批已开启下一批，请到下一批调整 Agent。这里的历史安排仅供查看。' : task.archived_at ? '归档任务只供查看，恢复后可以调整。' : '当前正在结束或应用，请等待后续设置恢复。' : task.settings_scope === 'future' ? '可以调整工具、模型和思考强度。规划设置用于后续追问，其他角色用于下一批修改，已完成记录保留。' : '改选后保存，从下一次派发起生效；正在执行的 Agent 保持当前设置。'));
  if (draft.plan.mode) editor.append(el('p', 'hint', '本次分工：' + PlanUI.modeNames[draft.plan.mode] + '。前后端依次接手同一份副本。'));
  const fields = el('div', 'task-plan-grid'); fields.id = 'task-plan';
  planFields(fields, 'task-plan', draft.plan, !editable || draft.saving, () => {
    const local = drafts.get(task.id); local.plan = { ...fields._plan }; local.dirty = true; updateDraftHint(task, local);
  });
  editor.append(fields);
  const footer = el('div', 'plan-footer'), save = button(task.settings_scope === 'future' ? '保存后续设置' : '保存后续安排', 'primary', () => savePlan(task.id)); save.id = 'save-plan'; save.disabled = !editable;
  const reset = button('载入最新安排', '', () => { drafts.delete(task.id); editor.dataset.task = ''; render(); }); reset.id = 'reset-plan';
  const hint = el('span', 'draft-hint'); hint.id = 'draft-hint'; hint.setAttribute('role', 'status'); footer.append(save, reset, hint); editor.append(footer);
  if (!editable) { save.hidden = true; reset.hidden = true; }
  updateDraftHint(task, draft);
}
function updateDraftHint(task, draft) {
  task = task.settings_scope ? task : ModelState.settingsTask(task);
  if (!$('draft-hint')) return;
  $('plan-unsaved').hidden = !draft.dirty || !task.settings_editable;
  AgentSettings.markDirty(draft.dirty && task.settings_editable);
  AssistantUI.sync(tasks.find(item => item.id === task.id));
  $('draft-hint').textContent = task.successor_task_id ? '本批已开启下一批，请到下一批调整 Agent。历史安排保留，未保存的旧选择不会应用到下一批。' : draft.conflict ? '其他窗口或阶段更新了安排。改选已保留，请先载入最新安排。' : draft.dirty ? '尚未保存。正在执行的 Agent 不会被切走。' : task.settings_scope === 'future' ? task.future_plan ? '后续设置已保存。下一次追问或下一批使用。' : '沿用原安排，后续模型和思考强度可以调整。' : '已保存第 ' + (task.plan_revision || 0) + ' 版安排，两个工具共用此任务的进度。';
  $('save-plan').disabled = !task.settings_editable || !draft.dirty || draft.conflict || !!draft.saving;
  $('reset-plan').disabled = !!draft.saving;
  $('reset-plan').hidden = !draft.dirty;
}
async function savePlan(id) {
  const draft = drafts.get(id); if (!draft?.dirty || draft.saving || draft.conflict) return;
  draft.saving = true; $('save-plan').disabled = true;
  setPlanSaving(id, true);
  const scope = draft.scope, revision = draft.revision;
  try {
    await api('tasks/' + id + (scope === 'future' ? '/future-models' : '/models'), { expected_revision: revision, plan: { ...draft.plan } });
    AssistantUI.modelSaveSucceeded(id);
    const current = drafts.get(id); if (current?.scope === scope && current.revision === revision) drafts.delete(id);
    if (selected === id) $('task-plan-editor').dataset.task = '';
    if (selected === id) notice('global-success', scope === 'future' ? '后续设置已保存，下一次追问或下一批使用。' : '后续安排已保存。当前阶段继续执行，新选择从下一次派发起生效。');
    await refresh();
  } catch (err) { if (selected === id) AssistantUI.modelSaveError(id, friendly(err.message)); }
  finally { const current = drafts.get(id); if (current) { current.saving = false; if (selected === id) updateDraftHint(tasks.find(t => t.id === id), current); } setPlanSaving(id, false); }
}
function setPlanSaving(id, saving) {
  if (selected !== id) return;
  const task = ModelState.settingsTask(tasks.find(t => t.id === id));
  for (const select of $('task-plan-editor').querySelectorAll('select')) select.disabled = saving || !task.settings_editable;
  if (task.model_plan?.mode) $('task-plan-reviewer').disabled = true;
  if ($('reset-plan')) $('reset-plan').disabled = saving;
}

function renderProof(task) { if (!task || WorkspaceUI.current() !== 'results') return; const signature = JSON.stringify(task); if (signature === proofSignature) return; proofSignature = signature; TaskProof.render(task); }
function evidenceURL(id, path) { return '/api/v1/reliable/tasks/' + encodeURIComponent(id) + '/evidence?path=' + encodeURIComponent(path); }

async function refresh() {
  if (loading) { refreshQueued = true; return; } loading = true;
  const revision = selectionRevision, id = selected;
  try {
    const result = await WorkbenchState.summaryPages(path => workspaceReader.read(path), summaryPagesWanted, summaryQuery);
    const summaries = result.tasks; summaryNextOffset = result.next_offset; summaryTotal = result.total;
    tasks = summaries.map(item => {
      const detail = workspaceDetails.get(item.id);
      return WorkbenchState.navigationItem(item, detail, id);
    });
    if (revision !== selectionRevision || id !== selected) { refreshQueued = true; return; }
    render();
    if (id) {
      const detail = await workspaceReader.read('tasks/' + encodeURIComponent(id) + '?view=workspace');
      workspaceDetails.set(id, detail);
      if (revision !== selectionRevision || id !== selected) { refreshQueued = true; return; }
      const summary = tasks.find(item => item.id === id);
      const current = { ...detail, project_id: summary?.project_id };
      tasks = tasks.some(item => item.id === id) ? tasks.map(item => item.id === id ? current : item) : [current, ...tasks];
      render();
    }
    WorkbenchUI.notify(summaries);
  }
  catch (err) { notice('global-error', friendly(err.message)); }
  finally { loading = false; if (refreshQueued) { refreshQueued = false; void refresh(); } }
}
async function perform(id, action) {
  if (!AssistantUI.allowDispatch(id, action)) return;
  if (actionBusy) return; actionBusy = true; if (action === 'recover-cancelled') recoveringTask = id; render(); notice('global-error', ''); notice('global-success', '');
  try { const result = await api('tasks/' + id + '/' + action, {}); if (action === 'recover-cancelled') tasks = tasks.map(task => task.id === id ? result : task); const messages = { archive: '任务已归档，项目与证据保留。', unarchive: '任务已恢复，不会自动开始执行。', 'open-project': '已向文件管理器发送打开原项目的请求。', open: '已向文件管理器发送打开工作副本的请求。', 'open-evidence': '已向文件管理器发送打开证据文件夹的请求。' }; if (messages[action]) notice('global-success', messages[action]); }
  catch (err) { notice('global-error', friendly(err.message)); }
  finally { actionBusy = false; recoveringTask = ''; detailSignature = ''; await refresh(); render(); }
}
function updateNewAttachmentControls() { $('goal').required = !AttachmentUI.ids('new').length; $('submit').disabled = creating || !config.providers || AttachmentUI.blocked('new'); AttachmentUI.disabled('new', creating); }
async function createTask(sample) {
  if (creating || !sample && AttachmentUI.blocked('new')) return; creating = true; $('new-fields').disabled = true; $('submit').disabled = true; $('demo').disabled = true; notice('form-error', '');
  AgentSettings.setCreating(true); AssistantUI.setCreating(); AttachmentUI.disabled('new', true);
  $('submit').textContent = sample ? '正在准备示例…' : '正在准备本地存档…';
  const model_plan = readPlan('new-plan');
  try {
    const payload = sample ? { model_plan } : { attachment_ids: AttachmentUI.ids('new'), project: $('project').value.trim(), goal: $('goal').value.trim(), read_only: $('read-only').checked, commands: $('commands').value.split(/\r?\n/).map(s => s.trim()).filter(Boolean), allowed_paths: $('scope').value.split(',').map(s => s.trim()).filter(Boolean), model_plan };
    const task = await api(sample ? 'demo' : 'tasks', payload); selected = task.id; location.hash = task.id; if (!sample) AttachmentUI.clear('new'); await refresh();
  } catch (err) { notice('form-error', friendly(err.message)); }
  finally { creating = false; AgentSettings.setCreating(false); AssistantUI.setCreating(); $('new-fields').disabled = false; $('submit').disabled = false; $('demo').disabled = !config.demo_project; $('submit').textContent = $('read-only').checked ? '开始了解项目' : '开始协作'; updateNewAttachmentControls(); }
}
async function browse() {
  const b = $('browse'); if (b.disabled) return; b.disabled = true; b.textContent = '正在选择…';
  notice('form-error', '');
  try { const result = await api('pick-project', {}); if (result.path) $('project').value = result.path; }
  catch (err) { notice('form-error', friendly(err.message)); }
  finally { b.disabled = false; b.textContent = '选择文件夹'; }
}
async function initialize() {
  $('new-fields').disabled = true; $('submit').disabled = true; $('demo').disabled = true;
  try { config = await api('config'); renderConnections(); renderNewPlan(config.default_plan); $('new-fields').disabled = false; $('submit').disabled = false; $('demo').disabled = !config.demo_project; await refresh(); updateNewAttachmentControls(); }
  catch (err) { notice('global-error', friendly(err.message)); }
}
$('form').onsubmit = event => { event.preventDefault(); createTask(false); };
$('read-only').onchange = () => { const inspection = $('read-only').checked; $('work-mode').hidden = inspection; $('new-plan').dataset.inspection = String(inspection); document.querySelector('.composer-note').textContent = inspection ? '直接使用所选目录，由原生只读策略和读取工具限制约束，不再准备另一份副本。' : '先在独立副本完成，你验收后才应用到原项目。'; $('submit').textContent = inspection ? '开始了解项目' : '开始协作'; const plan=readPlan('new-plan'), choice=AssistantState.selection(plan,config.providers||[]); try { renderNewPlan(choice.mode==='custom'?plan:AssistantState.apply(plan,choice.mode,config.providers||[],choice.model,choice.effort,inspection)); } catch(error) { notice('form-error',friendly(error.message)); AssistantUI.renderNew(); } };
$('demo').onclick = () => createTask(true); $('browse').onclick = browse;
async function refreshModels(quiet = false) {
  const b = $('refresh-models'); if (b.disabled) return; b.disabled = true; b.textContent = '正在读取模型…';
  try { const result = await api('models/refresh', {}); const plan = readPlan('new-plan'); config.providers = result.providers; config.default_plan = result.default_plan; renderConnections(); renderNewPlan(plan); $('task-plan-editor').dataset.task = ''; render(); const failed = result.providers.some(p => p.catalog_error); if (!quiet) notice('global-success', failed ? '' : '模型列表已刷新。你的安排已保留。'); notice('global-error', failed ? '有模型目录未刷新。已保留配置项，请检查原工具后重试。' : ''); }
  catch (err) { notice('global-error', friendly(err.message)); }
  finally { b.disabled = false; b.textContent = '刷新模型列表'; }
}
$('refresh-models').onclick = () => refreshModels();
DesktopUI.init({ tasks: () => tasks, selected: () => selected, changed: () => { historySignature = ''; renderHistory(); const query = $('task-search').value.trim(); if (query !== summaryQuery) { clearTimeout(searchTimer); searchTimer = setTimeout(() => { summaryQuery = query; summaryPagesWanted = 1; selectionRevision++; void refresh(); }, 250); } }, action: perform });
AgentSettings.init();
AttachmentUI.mount('new', $('new-attachments'), { id: 'new-attachments', pasteTarget: $('goal'), dropTarget: $('goal').closest('.composer'), onChange: updateNewAttachmentControls });
$('new-task').onclick = () => { selectTask(''); $('goal').focus(); }; $('refresh').onclick = () => config.providers ? refresh() : initialize();
window.addEventListener('hashchange', () => { const next = location.hash.slice(1); if (next !== selected) selectionRevision++; selected = next; detailSignature = ''; proofSignature = ''; render(); void refresh(); });
document.addEventListener('visibilitychange', () => { if (!document.hidden && config.providers) refresh(); });
function pollWorkspace() {
  const active = tasks.some(task => task.execution_active || ['queued','preparing','auditing','implementing','checking','reviewing','applying'].includes(task.state));
  setTimeout(() => { if (!document.hidden && config.providers) void refresh(); pollWorkspace(); }, active ? 3000 : 10000);
}
pollWorkspace();
initialize();
