const { test } = require('node:test');
const assert = require('node:assert/strict');
const state = require('./page-attachment-state.js');
const mib = 1024 * 1024;
// @covers AC-RELIABLE-ATTACHMENTS-001.1/.2
test('attachment selection accepts arbitrary formats and enforces file count and total byte limits', () => {
  assert.equal(typeof state.selectionError, 'function');
  assert.equal(state.selectionError([], [{ name: 'archive.unknown', size: 100 * mib }]), '');
  assert.match(state.selectionError([], [{ name: 'big.bin', size: 100 * mib + 1 }]), /100 MB/);
  assert.match(state.selectionError([{ name: 'a.pdf', size: 60 * mib }], [{ name: 'b.zip', size: 41 * mib }]), /总大小/);
  assert.match(state.selectionError(Array.from({length:10}, (_,i) => ({ name: String(i), size: 0 })), [{ name:'next',size:0 }]), /10/);
});
// @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
test('queued and failed attachments block submission while only ready identifiers can be sent', () => {
  assert.equal(typeof state.blocksSend, 'function');
  assert.equal(state.blocksSend([]), false);
  assert.equal(state.blocksSend([{status:'ready',attachment:{id:'saved'}}]), false);
  for (const status of ['queued','uploading','failed']) assert.equal(state.blocksSend([{status}]), true);
  assert.deepEqual(state.ids([{status:'ready',attachment:{id:'saved'}},{status:'uploading',attachment:{id:'partial'}}]), ['saved']);
});
// @covers AC-RELIABLE-ATTACHMENTS-001.2/.4
test('late upload completion cannot recreate a removed attachment or overwrite a newer attempt', () => {
  assert.equal(typeof state.settle, 'function');
  const current = [{key:'item',generation:2,status:'uploading'}];
  assert.deepEqual(state.settle([], 'item', 1, {status:'ready'}), []);
  assert.deepEqual(state.settle(current, 'item', 1, {status:'ready'}), current);
  assert.equal(state.settle(current, 'item', 2, {status:'ready'})[0].status, 'ready');
  assert.equal(current[0].status, 'uploading');
});
