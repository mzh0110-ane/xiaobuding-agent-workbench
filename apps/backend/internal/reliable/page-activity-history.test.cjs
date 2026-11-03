const {test}=require('node:test');
const assert=require('node:assert/strict');
const {readFileSync}=require('node:fs');
const {join}=require('node:path');
const vm=require('node:vm');

class Element {
  constructor(tag){Object.assign(this,{tag,children:[],dataset:{},scrollTop:0,scrollHeight:500,textContent:'',checked:false,disabled:false});}
  append(...nodes){this.children.push(...nodes);}
  replaceChildren(...nodes){this.children=nodes;}
  setAttribute(){}
  addEventListener(name,handler){this.handlers??={};this.handlers[name]=handler;}
  getClientRects(){return [{}];}
  querySelectorAll(){return [];}
  querySelector(){return null;}
}
function all(node){return [node,...node.children.flatMap(all)];}
function fixture(){
  const host=new Element('section'), scroll=new Element('div'), requests=[], timers=[];
  const document={hidden:false,createElement:tag=>new Element(tag),createTextNode:text=>Object.assign(new Element('text'),{textContent:text}),getElementById:id=>id==='task-activity'?host:id==='workspace-content'?scroll:all(host).find(node=>node.id===id)};
  const context=vm.createContext({document,AbortController,Date,Set,roleTitle:value=>value,setTimeout:(fn,delay)=>{const timer={fn,delay};timers.push(timer);return timer;},clearTimeout:timer=>{if(timer)timer.cancelled=true;},fetch:(url,options)=>new Promise(resolve=>requests.push({url,options,resolve}))});
  vm.runInContext(readFileSync(join(__dirname,'page-activity.js'),'utf8')+'\nglobalThis.activity=TaskActivity;',context);
  return {context,host,requests,timers,find:id=>document.getElementById(id),sync:id=>context.activity.sync({id,state:'blocked'})};
}
const flush=()=>new Promise(resolve=>setImmediate(resolve));
function respond(request,events,cursor,before,earlier=true){request.resolve({ok:true,json:async()=>({events,cursor,before_cursor:before,has_earlier:earlier,active:false,limited:false})});}
function events(start,count){return Array.from({length:count},(_,i)=>({sequence:start+i+1,time:'2026-10-26T00:00:00Z',kind:'message',title:'event-'+(start+i)}));}

test('activity starts at latest page and never auto-drains old history',async()=>{
  const f=fixture();f.sync('task');
  assert.match(f.requests[0].url,/\?tail=1$/);
  respond(f.requests[0],events(900,100),1000,900);await flush();
  assert.equal(f.requests.length,1);
  assert.ok(f.timers.at(-1).delay>=1000,'a full initial tail must not trigger zero-delay backlog drain');
  f.timers.at(-1).fn();
  assert.match(f.requests[1].url,/\?after=1000$/);
});

test('older pages use explicit before cursor and latest action rejects delayed history',async()=>{
  const f=fixture();f.sync('task');respond(f.requests[0],events(900,100),1000,900);await flush();
  assert.ok(f.find('activity-earlier'),'an explicit earlier-page control is available');
  f.find('activity-earlier').onclick();
  assert.match(f.requests[1].url,/\?before=900$/);
  f.find('activity-latest').onclick();
  assert.match(f.requests[2].url,/\?tail=1$/);
  respond(f.requests[2],events(2000,2),2002,2000);await flush();
  respond(f.requests[1],events(800,100),900,800);await flush();
  const text=all(f.host).map(node=>node.textContent).join('\n');
  assert.match(text,/event-2001/);
  assert.doesNotMatch(text,/event-800/);
});

test('explicit older window stays bounded and a task switch isolates pending requests',async()=>{
  const f=fixture();f.sync('old');respond(f.requests[0],events(900,100),1000,900);await flush();
  for(let start=800;start>=300;start-=100){
    f.find('activity-earlier').onclick();respond(f.requests.at(-1),events(start,100),start+100,start);await flush();
  }
  assert.ok(f.find('activity-feed').children.filter(node=>node.tag==='li').length<=501);
  f.find('activity-earlier').onclick();const old=f.requests.at(-1);
  f.sync('new');respond(f.requests.at(-1),events(4000,2),4002,4000);await flush();
  respond(old,events(200,100),300,200);await flush();
  const text=all(f.host).map(node=>node.textContent).join('\n');
  assert.match(text,/event-4001/);assert.doesNotMatch(text,/event-200\b/);
});

test('an old failed forward response cannot reset the replacement task cursor',async()=>{
  const f=fixture();f.sync('old');respond(f.requests[0],events(900,100),1000,900);await flush();
  f.timers.at(-1).fn();const old=f.requests.at(-1);
  f.sync('new');respond(f.requests.at(-1),events(2000,2),2002,2000);await flush();
  old.resolve({ok:false,status:409});await flush();
  f.timers.at(-1).fn();assert.match(f.requests.at(-1).url,/new\/activity\?after=2002$/);
});

test('reading up enables the explicit latest action',async()=>{
  const f=fixture();f.sync('task');respond(f.requests[0],events(900,100),1000,900);await flush();
  assert.equal(f.find('activity-latest').disabled,true);
  f.find('activity-feed').handlers.wheel({deltaY:-1});
  assert.equal(f.find('activity-latest').disabled,false);
});
