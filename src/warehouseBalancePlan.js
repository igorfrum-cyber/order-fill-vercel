import { BRAND_RULES, brandRule, articleNormalizeOptions, normalizeArticle, chzNameSimilarity, levissimeVolumeSignature, compatibleTyumenNames, tyumenProductKey, sheetBounds, sheetCellValue, asText, normalizeHeader, parseNumber, detectColumns, isSourceTotalRow } from "./workbookProcessor.js";

const aliases = {
  angiopharm: /ангио|angiopharm/iu, christina: /кристин|кристи(?:\s|$)|christina/iu,
  levissime: /левис|levissime|leviss|лев\s/iu, sothys: /сотис|sothys/iu,
  novacutan: /новакутан|novacutan/iu, skin_synergy: /skin.?synergy|скин.?синер/iu,
  klapp: /клапп|klapp/iu,
};
export const MATCH_WARNING = 'Не удалось сопоставить, нужно рассчитать вручную';

function sameProduct(left, right, brand) {
  if (levissimeVolumeSignature(left.name) !== levissimeVolumeSignature(right.name)) return false;
  return compatibleTyumenNames(left, right, brand)
    || (/^\s*чз\s/iu.test(left.name) !== /^\s*чз\s/iu.test(right.name)
      && chzNameSimilarity(left.name, right.name, brandRule(brand)) >= 0.9);
}

export function balanceQuantity(delivery, tyumen) {
  if (![delivery, tyumen].every(v => Number.isFinite(v) && v >= 0)) return null;
  const quantity = Math.min(Math.floor(delivery), Math.max(0, Math.ceil((delivery - 4 * tyumen) / 5 - 1e-10)));
  return quantity === 1 ? 0 : quantity;
}

function readStock(workbook, fileName) {
  let detection;
  const metadata = [fileName];
  for (const sheet of workbook.sheets) {
    metadata.push(sheet.name);
    const bounds = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(bounds.maxRow, 120); row++) {
      const columns = {};
      for (let col = 1; col <= bounds.maxColumn; col++) {
        const value = asText(sheetCellValue(sheet, row, col));
        const header = normalizeHeader(value);
        if (!detection) {
          metadata.push(value);
          if (header === 'склад' && col < bounds.maxColumn) {
            const location = asText(sheetCellValue(sheet, row, col + 1));
            if (location) metadata.push(`Склад ${location}`);
          }
        }
        if (/наименование|номенклатура|название|товар/u.test(header)) columns.name ??= col;
        if (/артикул|^арт|^код/u.test(header)) columns.article ??= col;
        if (/остаток/u.test(header) && !/месяц|мес|офис:|склад:/u.test(header)) columns.stock ??= col;
      }
      if (columns.name && columns.stock && !detection) detection = { sheet, headerRow: row, columns };
    }
  }
  // Standard reports use the same right-hand product columns as recalculation.
  try { detection = detectColumns(workbook, 'source'); } catch { /* Simple stock-only tables use the minimal header above. */ }
  if (!detection) throw Error('Не найдены колонки названия и текущего остатка.');
  const text = metadata.join(' ');
  const brands = Object.keys(BRAND_RULES).filter(key => aliases[key].test(text));
  if (brands.length !== 1) throw Error(`В файле «${fileName}» не удалось однозначно определить бренд. Укажите бренд в таблице или имени файла.`);
  const brand = brands[0];
  const delivery = /доставк/iu.test(text);
  const tyumen = /тюмен/iu.test(text);
  // A valid filename must not hide a conflicting location inside the workbook.
  if (/сургут|вартовск|уренго|москв|екатеринбург|тобольск|нефтеюганск|ноябрьск|салехард|ханты.?мансийск/iu.test(text)) {
    throw Error(`В файле «${fileName}» указан другой город. Разрешены только склад Тюмень и склад Доставка.`);
  }
  const declaredWarehouses = new Set();
  for (const value of metadata) {
    const declaration = normalizeHeader(value).match(/^(?:склад|офис)\s*[:\-]?\s+(.+)$/u);
    if (!declaration) continue;
    const location = declaration[1];
    if (!/^(?:доставка|тюмень|тюмени)(?:\s|$)/u.test(location)) {
      throw Error(`В файле «${fileName}» указан неподдерживаемый склад: «${value}».`);
    }
    declaredWarehouses.add(/^доставк/u.test(location) ? 'delivery' : 'tyumen');
  }
  if (declaredWarehouses.size > 1) throw Error(`В файле «${fileName}» указаны разные склады. Нужна отдельная таблица одного склада.`);
  const warehouse = [...declaredWarehouses][0] || (delivery ? 'delivery' : tyumen ? 'tyumen' : null);
  if (!warehouse) {
    throw Error(`В файле «${fileName}» не удалось определить склад. Нужен склад Доставка или Тюмень.`);
  }
  const rule = brandRule(brand), groups = new Map();
  const { sheet, columns, headerRow } = detection;
  for (let row = headerRow + 1; row <= sheetBounds(sheet).maxRow; row++) {
    if (isSourceTotalRow(detection, row, sheetBounds(sheet).maxColumn)) continue;
    const name = asText(sheetCellValue(sheet, row, columns.name));
    if (!name || /^(итого|всего|total)(?:\s|:|$)/iu.test(name)) continue;
    const articleRaw = columns.article ? asText(sheetCellValue(sheet, row, columns.article)) : '';
    const article = normalizeArticle(articleRaw, articleNormalizeOptions(rule));
    const rawStock = sheetCellValue(sheet, row, columns.stock);
    const item = { name, article, articleRaw, stock: asText(rawStock) === '' ? 0 : parseNumber(rawStock) };
    const key = tyumenProductKey(item, brand);
    if (!groups.has(key)) groups.set(key, []);
    const group = groups.get(key);
    const compatible = group.find(other => sameProduct(other, item, brand));
    if (compatible) compatible.stock = compatible.stock === null || item.stock === null ? null : compatible.stock + item.stock;
    else group.push(item);
  }
  return { groups, brand, warehouse };
}

export function buildWarehouseBalanceResult({ deliveryWorkbook, tyumenWorkbook, deliveryFileName, tyumenFileName }) {
  const first = readStock(deliveryWorkbook, deliveryFileName);
  const second = readStock(tyumenWorkbook, tyumenFileName);
  if (first.warehouse === second.warehouse) throw Error('Загружены два одинаковых склада. Нужны склад Доставка и склад Тюмень.');
  const deliverySource = first.warehouse === 'delivery' ? first : second;
  const tyumenSource = first.warehouse === 'tyumen' ? first : second;
  if (deliverySource.brand !== tyumenSource.brand) throw Error('В таблицах указаны разные бренды. Загрузите два склада одного бренда.');
  const { brand } = deliverySource;
  const delivery = deliverySource.groups, tyumen = tyumenSource.groups;
  const rows = [], used = new Set();
  for (const [key, items] of delivery) {
    let candidates = tyumen.get(key);
    let targetKey = key;
    if (!candidates && items.length === 1) {
      const matches = [...tyumen].filter(([, values]) => values.length === 1 && !used.has(values[0])
        && (!items[0].article || !values[0].article) && sameProduct(items[0], values[0], brand));
      if (matches.length === 1) [targetKey, candidates] = matches[0];
    }
    for (const item of items) {
      const matches = (candidates || []).filter(other => !used.has(other) && sameProduct(item, other, brand));
      const other = matches.length === 1 ? matches[0] : null;
      const unique = other && items.filter(candidate => sameProduct(candidate, other, brand)).length === 1;
      if (unique) {
        used.add(other);
        const quantity = balanceQuantity(item.stock, other.stock);
        rows.push({ article: item.articleRaw || other.articleRaw, name: item.name, deliveryStock: item.stock, tyumenStock: other.stock, quantity, warning: quantity === null ? MATCH_WARNING : '' });
      } else rows.push({ article: item.articleRaw, name: item.name, deliveryStock: item.stock, tyumenStock: null, quantity: null, warning: MATCH_WARNING });
    }
  }
  for (const items of tyumen.values()) for (const item of items) if (!used.has(item)) rows.push({ article: item.articleRaw, name: item.name, deliveryStock: null, tyumenStock: item.stock, quantity: null, warning: MATCH_WARNING });
  return { rows, brandLabel: brandRule(brand).label };
}

export function buildWarehouseBalance(options) {
  return buildWarehouseBalanceResult(options).rows;
}
