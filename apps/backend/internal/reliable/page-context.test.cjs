const { test } = require('node:test');
const assert = require('node:assert/strict');
const { summary } = require('./page-context.js');

test('saved messages are distinct from a prepared handoff archive', () => {
  const text = summary({ saved_batches: 3, saved_messages: 110, prepared: false });
  assert.equal(text.saved, '3 批 · 110 条项目消息已保存');
  assert.match(text.archive, /尚未准备/);
});

test('an archive reports its own snapshot count and the update boundary', () => {
  const text = summary({ saved_batches: 3, saved_messages: 111, prepared: true, complete: true, archived_messages: 110, summary: '新消息在下次派发更新。' });
  assert.match(text.archive, /110 条/);
  assert.match(text.detail, /下次派发/);
});
