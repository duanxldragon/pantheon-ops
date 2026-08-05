import assert from 'node:assert/strict';
import test from 'node:test';
import { hasCssDeclaration } from './css-declarations.mjs';

test('hasCssDeclaration matches the exact property name', () => {
  const block = 'min-height: 64px;\npadding: 16px 24px !important;';

  assert.equal(hasCssDeclaration(block, 'height', '64px'), false);
  assert.equal(hasCssDeclaration(block, 'min-height', '64px'), true);
  assert.equal(hasCssDeclaration(block, 'padding', '16px 24px'), true);
});
