'use strict';
let input='';process.stdin.setEncoding('utf8');process.stdin.on('data',data=>input+=data);
process.stdin.on('end',async()=>{
 try{const options=JSON.parse(input);options.observe=event=>{console.log(JSON.stringify({type:'zcode.event',event}));if(event.type==='session')console.log(JSON.stringify({type:'thread.started',thread_id:event.sessionId}));};
 const result=await require('./zcode-client.cjs').run(options);console.log(JSON.stringify({type:'result',...result}));
 }catch(error){console.error(/^ZCode (did not confirm|selected|request timed out|native process)/.test(error.message)?error.message:'ZCode bridge failed. Check the native event history.');process.exitCode=1;}
});
