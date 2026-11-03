'use strict';
// Native planning blocks write tools; plugin lifecycle hooks also need a process boundary.
const childProcess=require('node:child_process');
for(const name of ['spawn','spawnSync','exec','execSync','execFile','execFileSync','fork']){
 childProcess[name]=()=>{const error=new Error('ZCode read-only: subprocess hooks are disabled');error.code='EROFS';throw error;};
}
require('node:module').syncBuiltinESMExports();
