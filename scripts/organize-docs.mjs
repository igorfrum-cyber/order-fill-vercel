import { readFile, writeFile, mkdir } from 'node:fs/promises';
for(const name of ['tyumen','budget-planner','christina-lines','table-recalculation']){
  const original=await readFile(`docs/${name}.md`,'utf8');await mkdir('docs/product',{recursive:true});
  await writeFile(`docs/product/${name}.md`,`Статус: действующий. Область: браузерная версия Order Fill. Проверено: 13 октября 2026 года.\n\n${original}`);
  await writeFile(`docs/${name}.md`,`# Документ перенесён\n\nСтатус: заменён. Проверено: 13 октября 2026 года.\n\nАктуальная версия: [${name}](product/${name}.md).\n`);
}
await mkdir('docs/archive',{recursive:true});
await writeFile('docs/archive/code-map-before-refactor.md','Статус: исторический. Область: структура до реорганизации. Проверено: 13 октября 2026 года.\n\n'+await readFile('docs/code-map.md','utf8'));
await writeFile('docs/code-map.md','# Карта кода перенесена\n\nСтатус: заменён. Проверено: 13 октября 2026 года.\n\nАктуальная версия: [Карта модулей](architecture/modules.md).\n');
const spec=await readFile('docs/plans/2026-10-13-internal-structure-refactor.md','utf8');
await writeFile('docs/plans/2026-10-13-internal-structure-refactor.md',spec.replace('Статус: готово к реализации.','Статус: план; реализация завершена в рабочей ветке, ожидает review.'));
