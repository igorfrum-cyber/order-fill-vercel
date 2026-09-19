# Режимы оракула CHRISTINA PROFF

Постановка для трёх слоёв. Алгоритм `changeCost` / `CompleteChristinaLines` и
кнопки скачивания в BudgetPanel в этом документе только специфицируются, не
патчатся одним агентом.

Источник формул — `origin/main` (`planBudget` + `christinaLines.js`) и
`docs/plans/2026-09-18-budget-plan-complexity.md`. Сложность n² у PROFF — не
баг: `changeCost` пересчитывает весь заказ. Ускорение = тот же оракул
дешевле, не другой алгоритм добора.

---

## 1. Зачем

На CHRISTINA PROFF каждый шаг добора бюджета вызывает полный пересчёт заказа.
Это даёт 1:1 с `origin/main`, но на живых бланках тормозит превью.

Нужны три режима на компанию, без смены формул комплектов:

| JSON `christina_proff_mode` | Смысл |
| --- | --- |
| `standard` | Текущий полный оракул. Default. |
| `fast` | Инкрементальный оракул, те же формулы, тот же qty/total. |
| `compare` | Два независимых `PlanBudget` (standard и fast). В превью остаётся standard. Оба плана и список расхождений отдаются клиенту. |

Несовпадение fast и standard в compare — сигнал чинить fast, а не «похожий»
заказ. Прикладные количества всегда из standard.

---

## 2. Где выбирается режим

Как `matching_mode`: настройка компании, меняет только `platform_admin`.
Владелец и закупщик поле не меняют. В purchaser-payloads поле скрыто тем же
гейтом, что `matching_mode`.

Экран «Компании», отдельный fieldset **CHRISTINA PROFF**:

- **Текущий расчёт** — полный пересчёт на каждом шаге, как сейчас
- **Быстрый расчёт** — тот же заказ и та же сумма, без полного пересчёта
- **Сверка** — считает оба оракула; в превью остаётся текущий расчёт и
  показывает, совпали ли

JSON-поле: `christina_proff_mode`. Значения `standard` | `fast` | `compare`.
Пустое и неизвестное = `standard`. Protobuf `UNSPECIFIED` тоже standard.

Режим не снапшотится на job: это живая настройка превью бюджета, не
сопоставления файла.

---

## 3. Owners

Новый микросервис и новый экран не нужны.

| Слой | Owner | Делает | Не делает |
| --- | --- | --- | --- |
| Persistence / contract | `identity-service`, proto, gateway company JSON, `CompaniesScreen` | Поле компании, enum, Create/Update только platform_admin, скрытие от purchaser | `changeCost`, BudgetPanel, dual Excel |
| Math | `calculation-service` | `PlanBudget` по режиму; compare = два независимых прохода; список mismatches | HTTP, UI, Excel |
| Budget HTTP + panel | `gateway-service` `budget.go`, OpenAPI budget-plan, `frontend/src/ui/order/**` | Прокинуть mode из компании, JSON `compare` + `fast_*`, кнопки трёх JSON | Менять формулы |

Gateway budget HTTP и баннеры панели — отдельный агент. OpenAPI company
поле живёт здесь; schema compare у budget-plan — у HTTP-агента.

---

## 4. Контракт `PlanBudget`

`PlanBudgetRequest.christina_proff_mode = 8`.

`PlanBudgetResponse`:

- `rows` / `total` / `complete` / `reason` / `line_steps` — **всегда standard**
  (прикладной план превью и Excel).
- `christina_proff_mode = 9` — какой режим компания попросила.
- `compare = 10` — заполняется в compare.
- `fast_rows = 11`, `fast_total = 12`, `fast_complete = 13`,
  `fast_reason = 14`, `fast_line_steps = 15` — независимый fast-план, только
  compare. В `standard`/`fast` пустые.

Сверка — **не** один greedy-проход с shadow-оракулом и не два Excel-файла.
Два отдельных вызова ядра с одним входом: сначала standard, потом fast.
Прикладной результат = standard. Fast нужен, чтобы скачать и сверить.

JSON `compare` (HTTP, не proto wire names 1:1 для want/got):

```json
{
  "match": true,
  "standard_ms": 12,
  "fast_ms": 3,
  "mismatches": [
    { "where": "changeCost", "key": "SKU-1", "field": "added_cents", "want": "150", "got": "149" }
  ]
}
```

`where`: `changeCost` | `proposeLineStep`. `field`: `quantity`, `total`,
`comment`, `added_cents`. Список полный, не только первое расхождение.
`match` истинно, когда список пуст и прикладные qty/total совпали.

Proto `BudgetOracleMismatch.want` / `got` — строки: в одном списке и числа, и
комментарии. HTTP может отдать число, если поле числовое.

---

## 5. Три JSON-файла в compare

Excel dual-blank **нет**. UI (другой агент) даёт скачать три JSON:

1. **текущий** — standard plan (`rows`, `total`, `complete`, `reason`,
   `line_steps`);
2. **быстрый** — fast plan (`fast_rows`, `fast_total`, `fast_complete`,
   `fast_reason`, `fast_line_steps`);
3. **расхождения** — объект `compare` (match, timings, `mismatches`).

Имена файлов и кнопки — у UI-агента. Контракт обязан отдать оба плана и
список, иначе скачивать нечего.

---

## 6. Поведение режимов для math-агента

### `standard`

Как сейчас: PROFF `changeCost` = два полных `ProcurementTotalCents`.
1:1 с `origin/main`.

### `fast`

Тот же greedy и те же правила комплектов. Оракул стоимости шага
инкрементальный, но в копейке совпадает со standard. Иначе плывут фильтр
105% и сортировка «ближе к остатку».

### `compare`

1. Засечь `standard_ms`, прогнать `PlanBudget` в standard.
2. Засечь `fast_ms`, прогнать `PlanBudget` в fast на **копии** входа.
3. Собрать `mismatches` по ключам строк, total, comment, шагам линий.
4. Response.rows/total = standard. Response.fast_* = fast. `compare.match`
   = список пуст.

Если fast упал, а standard нет — mismatch, превью всё равно standard.

---

## 7. Тесты

Identity / gateway (этот слой):

- новая компания → `standard`;
- `platform_admin` ставит `compare`;
- owner не меняет поле;
- purchaser JSON без `christina_proff_mode`; admin JSON с полем.

Math-агент:

- fixture PROFF: fast qty/total/comments = standard;
- compare на том же входе: `match=true`, `fast_rows` совпадают с `rows`;
- искусственный рассинхрон оракула: `match=false`, в списке есть
  `where`/`field`, `rows` остаются standard.

HTTP/UI-агент:

- budget-plan JSON содержит `compare` и `fast_*` только когда компания в
  compare;
- три download-кнопки; не два `.xlsx`.
