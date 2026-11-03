const { test } = require('node:test');
const assert = require('node:assert/strict');
const { taskTitle, goalText } = require('./page-state.js');

test('long multiline goals get a compact Unicode-safe title without changing the requirement', () => {
  const task = { goal: '😀'.repeat(40) + '\n\nNEXT BATCH REQUIREMENT:\n继续改进', project: 'E:\\项目' };
  const original = task.goal;
  assert.equal(taskTitle(task), '😀'.repeat(28) + '…');
  assert.equal(task.goal, original);
  assert.equal(taskTitle({ goal: '修复导出功能\n后续说明' }), '修复导出功能');
});

test('inherited requirement markers are localized for display and empty goals remain identifiable', () => {
  assert.equal(goalText({ goal: '原要求\n\nNEXT BATCH REQUIREMENT:\n继续修复' }), '原要求\n\n下一批要求：\n继续修复');
  assert.equal(taskTitle({ project: 'E:\\我的项目\\' }), '我的项目');
  assert.equal(taskTitle({}), '项目协作');
});
