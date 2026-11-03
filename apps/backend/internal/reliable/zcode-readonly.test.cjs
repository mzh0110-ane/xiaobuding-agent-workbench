const {test}=require('node:test'),assert=require('node:assert/strict');
const base=__dirname+'/';
test('ZCode read-only selection uses the actual ZCode planner and model',()=>{
 const s=require(base+'page-assistant-state.js'),providers=[{id:'zcode',available:true,model:'flash',models:[{id:'flash',efforts:['high']}]}];
 const p=s.apply({mode:'fullstack',auditor:'codex',reviewer:'codex'},'zcode',providers,'flash','high',true);
 assert.equal(p.auditor,'zcode');assert.equal(p.auditor_model,'flash');assert.equal(s.selection(p,providers).model,'flash');
});
test('read-only proof requires matching native session with independent planning enabled',()=>{
 const c=require(base+'zcode-client.cjs');
 const frame={method:'v4/conversation/frame',params:{topic:'conversation/s',frame:{payload:{snapshot:{sessionId:'s',config:{mode:'build',planEnabled:true}}}}}};
 assert.equal(c.readOnlyConfirmed(frame,'s'),true);assert.equal(c.readOnlyConfirmed(frame,'other'),false);
 frame.params.frame.payload.snapshot.config.planEnabled=false;assert.equal(c.readOnlyConfirmed(frame,'s'),false);
 const tools=c.readOnlyTools(true);assert.deepEqual(tools.toolAllowlist,['Read','Glob','Grep','Skill']);assert.ok(tools.toolDenylist.includes('ExitPlanMode'));assert.deepEqual(c.readOnlyTools(false),{});
});

test('read-only native process rejects plugin hook subprocesses',()=>{
 const cp=require('node:child_process'),path=require('node:path');
 const result=cp.spawnSync(process.execPath,['--require',base+'zcode-readonly.cjs','-e',"require('node:child_process').execSync('echo MUTATED')"],{encoding:'utf8'});
 assert.notEqual(result.status,0);assert.match(result.stderr,/ZCode read-only: subprocess hooks are disabled/);
});
