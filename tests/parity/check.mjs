import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import * as api from '../../src/workbookProcessor.js';
import { scenarios } from './scenarios.mjs';
import { planBudget } from '../../src/domain/budget/planner.js';
import { priceOrderRows } from '../../src/domain/budget/pricing.js';
const baseline=JSON.parse(await readFile(new URL('./baseline.json',import.meta.url),'utf8'));
assert.deepEqual(await scenarios(api,{planBudget,priceOrderRows}),baseline.data);
console.log('Public exports, reports, calculations and every ZIP part match baseline',baseline.commit);
