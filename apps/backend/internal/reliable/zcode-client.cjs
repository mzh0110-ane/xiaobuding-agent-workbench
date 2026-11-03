'use strict';
const {spawn}=require('node:child_process');
const path=require('node:path');
function readOnlyTools(enabled){return enabled?{toolAllowlist:['Read','Glob','Grep','Skill'],toolDenylist:['ExitPlanMode','EnterPlanMode','Bash','Write','Edit','NotebookEdit','Task','Agent','Workflow']}:{};}
function readOnlyConfirmed(message,session){
 const snapshot=message?.params?.frame?.payload?.snapshot;
 return message?.method==='v4/conversation/frame'&&message.params.topic==='conversation/'+session&&snapshot?.sessionId===session&&snapshot.config?.planEnabled===true&&snapshot.config.mode!=='yolo';
}
function completedResult(snapshot,session,completed){
 if(!completed||snapshot.session?.sessionId!==session)throw Error('Native session did not complete');
 const message=[...(snapshot.messages||[])].reverse().find(m=>m.info?.role==='assistant');
 const response=(message?.parts||[]).filter(p=>p.type==='text').map(p=>p.text||'').join('\n').trim();
 if(!response)throw Error('Native completed session has no assistant response');
 return {session,response};
}
async function run(options){
 const {entry,node=process.execPath,home,provider,model,effort='high',directory,prompt,observe=()=>{}}=options;
 const args=options.readOnly?['--require',path.join(__dirname,'zcode-readonly.cjs'),entry,'app-server','--stdio']:[entry,'app-server','--stdio'];
 const child=spawn(node,args,{cwd:directory,windowsHide:true,env:{...process.env,ZCODE_BUILTIN_PROVIDER_CONFIG_FILE:path.resolve(path.dirname(entry),'../config/provider/zcode-builtin.json'),ZCODE_PERSONAL_PROVIDER_CONFIG_FILE:path.join(home,'v2/provider_config.json')},stdio:['pipe','pipe','pipe']});
 child.stdout.setEncoding('utf8');
 let buffer='',sequence=0;const pending=new Map(),frames=new Map(),frameWaiters=new Map();
 const fail=()=>{for(const p of pending.values()){clearTimeout(p.timer);p.reject(Error('ZCode native process closed'))};pending.clear()};
 child.on('error',fail);child.on('exit',fail);child.stderr.on('data',()=>{});
 child.stdout.on('data',bytes=>{
  buffer+=bytes;let i;
  while((i=buffer.indexOf('\n'))>=0){const line=buffer.slice(0,i);buffer=buffer.slice(i+1);let v;try{v=JSON.parse(line)}catch{continue}
   if(v.method==='v4/conversation/frame'&&v.params?.frame?.payload?.snapshot){frames.set(v.params.topic,v);frameWaiters.get(v.params.topic)?.(v);}
   if(v.method&&v.id){child.stdin.write(JSON.stringify({id:v.id,error:{code:-32601,message:'Host capability unavailable'}})+'\n');continue}
   if(v.id&&pending.has(v.id)){const p=pending.get(v.id);pending.delete(v.id);clearTimeout(p.timer);v.error?p.reject(Error('ZCode protocol failure: '+v.error.code+' '+v.error.message)):p.resolve(v.result)}
  }
 });
 const rpc=(method,params)=>new Promise((resolve,reject)=>{const id=++sequence,timer=setTimeout(()=>{pending.delete(id);reject(Error('ZCode request timed out: '+method))},120000);pending.set(id,{resolve,reject,timer});child.stdin.write(JSON.stringify({id,method,params})+'\n')});
 try{
  await rpc('runtime/capabilities',{});
  const snapshot=await rpc('session/create',{workspace:{workspacePath:directory,workspaceKey:directory},persistence:'deferred',model:{providerId:provider,modelId:model,options:{reasoningLevel:effort}},thoughtLevel:effort,titleGenerationEnabled:false,dynamicWorkflowEnabled:false,...readOnlyTools(options.readOnly)});
  const id=snapshot.session.sessionId;
  if(snapshot.settings.model.current.providerId!==provider||snapshot.settings.model.current.modelId!==model)throw Error('ZCode selected a different model');
  const mode=options.readOnly?'plan':'yolo';
  const state=await rpc('session/setMode',{sessionId:id,mode});
  if(options.readOnly){
   const topic='conversation/'+id;let timer;
   const received=new Promise(resolve=>{frameWaiters.set(topic,resolve);timer=setTimeout(()=>resolve(undefined),10000)});
   let frame;
   try {await rpc('v4/conversation/subscribe',{topic,connectionId:'xiaobuding-readonly-'+id,clientMode:'desktop-continuous'});frame=frames.get(topic)||await received;}finally{clearTimeout(timer);frameWaiters.delete(topic)}
   if(!readOnlyConfirmed(frame,id))throw Error('ZCode did not confirm the requested read-only planning state');
   observe({type:'native_readonly_confirmed',session:id,planEnabled:true,tools:readOnlyTools(true).toolAllowlist});
  }else if(state.settings?.mode?.current!==mode)throw Error('ZCode did not confirm the requested permission mode');
  observe({type:'session',sessionId:id,model,mode});
  if(!prompt)return {catalog:snapshot.settings.model.available,session:id,modeState:state};
  await rpc('session/send',{sessionId:id,content:prompt});
  let cursor=0;
  for(;;){
   const page=await rpc('session/events',{sessionId:id,afterSeq:cursor,limit:1000});
   for(const event of page.events||[]){observe(event);cursor=Math.max(cursor,event.seq||event.sequence||0);
    if(event.type==='turn.failed')throw Error('ZCode native turn failed');
    if(event.type==='turn.completed')return completedResult(await rpc('session/read',{sessionId:id}),id,true);
   }
   await new Promise(resolve=>setTimeout(resolve,500));
  }
 }finally{for(const p of pending.values())clearTimeout(p.timer);pending.clear();child.stdin.end();child.kill()}
}
module.exports={run,completedResult,readOnlyTools,readOnlyConfirmed};
