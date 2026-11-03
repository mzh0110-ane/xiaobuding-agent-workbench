const fs = require('node:fs');
const assert = require('node:assert/strict');
const path = require('node:path');
const { createRequire } = require('node:module');
const root = path.resolve(__dirname, '../..');
const { chromium } = createRequire(path.join(root, 'apps/web/package.json'))('@playwright/test');
const [base, project, release] = process.argv.slice(2);
const out = process.env.XIAOBUDING_VALIDATION_DIR || path.join(root,'.tmp/reliable-conversation');
fs.mkdirSync(out,{recursive:true});
(async()=>{
  const browser=await chromium.launch({channel:'msedge',headless:true});
  const errors=[];
  try {
    const page=await browser.newPage({viewport:{width:1440,height:1050}});
    page.on('pageerror',error=>errors.push(error.message)); page.setDefaultTimeout(10000);
    await page.goto(base+'/reliable');
    await page.locator('#project').fill(project); await page.locator('#goal').fill('协议夹具：先了解当前项目');
    await page.locator('#read-only').check(); await page.locator('#submit').click();
    await page.locator('#activity-feed [data-kind="command"]').waitFor();
    fs.writeFileSync(release,'continue');
    await page.waitForFunction(()=>document.querySelector('#task-summary .badge')?.textContent==='只读调查完成');
    let answered=0;
    for (const question of ['协议夹具：进一步解释现状','协议夹具：指出下一步建议和依据']) {
      answered++;
      await page.locator('#followup-content').fill(question); await page.locator('#followup-send').click();
      await page.waitForFunction(expected=>document.querySelectorAll('.conversation-reply').length===expected&&document.querySelectorAll('[data-message-id]').length===expected,answered);
    }
    const id=new URL(page.url()).hash.slice(1);
    const task=await (await fetch(base+'/api/v1/reliable/tasks/'+id)).json();
    assert.equal(task.state,'inspected'); assert.equal(task.round,0); assert.equal(task.messages.length,2);
    assert.ok(task.messages.every(message=>message.status==='answered'&&message.reply.read_files.includes('app.txt')));
    assert.equal(task.runs.length,1); assert.equal(task.worktree,'');
    await page.reload(); await page.locator('.conversation-reply').first().waitFor();
    assert.equal(await page.locator('.conversation-reply').count(),2);
    assert.equal(await page.evaluate(()=>document.body.textContent.includes('PRIVATE_REASONING_MUST_NOT_DISPLAY')),false);
    await page.screenshot({path:path.join(out,'conversation-real-http.png'),fullPage:true});
    await page.setViewportSize({width:375,height:812});
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
    assert.deepEqual(errors,[]);
    const result={protocol_fixture:true,paid_model_call:false,real_http:true,persisted_questions:task.messages.length,task_state:task.state,round:task.round,original_dispatches:task.runs.length,source_unchanged:fs.readFileSync(path.join(project,'app.txt'),'utf8')==='original source\n',browser_errors:errors};
    fs.writeFileSync(path.join(out,'conversation-integration.json'),JSON.stringify(result,null,2));
    console.log(JSON.stringify(result));
  } finally {await browser.close()}
})().catch(error=>{console.error(error);process.exitCode=1});
