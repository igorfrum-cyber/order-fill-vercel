# Бюджет / CHRISTINA PROFF: поведение и почему n²

Постановка для дальнейшей работы. Не план патча: алгоритм, E2E и оптимизацию
в этом изменении не трогать. Источник истины — runtime-код
`calculation-service`, эталон комментария — `origin/main:src/budgetPlanner.js`.

Owner: `calculation-service` (`PlanBudget` / `PlanReportBudget`). Gateway
только валидирует скидку и мапит JSON→gRPC. Браузер Excel не считает.

---

## 1. Цель

Зафиксировать текущее поведение **1:1** с `origin/main` (`planBudget` +
`christinaLines.js`):

- те же `quantity` по ключам строк;
- тот же итоговый `total` в копейках;
- те же комментарии (`BudgetChangeComment` / `lineCompletion`);
- тот же порядок выбранных кандидатов (не «похожий» greedy).

Позже можно ускорить расчёт **без смены** qty / total / comments. Любая
оптимизация, которая меняет выбранную строку на шаге, ломает 1:1 — даже если
сумма совпала.

Сложность n² здесь не баг, а следствие оракула стоимости: для PROFF
`changeCost` пересчитывает весь заказ. Ускорение = тот же оракул дешевле, не
другой алгоритм добора.

---

## 2. Как работает сейчас

### 2.1 Вход: Price / group / line

Путь preview: UI → `POST /api/v1/order/budget-plan` (`gateway-service`
`planOrderBudget`) → gRPC `PlanBudget` →
`calculation.PlanReportBudget`.

`PlanReportBudget` (`budget.go`) ценит сырые строки отчёта и больше нигде
pricing не живёт:

| Поле | Откуда |
| --- | --- |
| `Price` | `round(BasePrice × (1 − Discount/100) × 100) / 100` — цена **после** основной скидки, рубли с копейками |
| `Delivery` | `DeliveryWeeks × 0.25` (месяцы) |
| `Unit` / `Step` / `Minimum` | `BudgetOrderRules(brand, name, box_size)` |
| `Group` | как пришло в запросе (`BudgetRow.group`) |
| `Line` | как пришло (`ChristinaLine`: id, name, article, required) |

`price` / `unit` / `step` / `minimum` / `delivery` на входе proto **игнорируются**.

Комплектная скидка включается только при **`Group == "proff"` и `Line != nil`**.
HOME и прочие бренды идут в чистый `planBudgetCore`.

Как `group` реально кладут вызывающие (факт контракта, не часть ядерного
цикла):

- North: `variant` (`"proff"` / `"home"`) — совпадает с проверкой.
- Обычный заказ: frontend шлёт `group: blankId`. `PlanBlanks` ставит id
  `blank-1`. В `origin/main` у CHRISTINA два слота и `group: result.blankId`
  равен `'proff'` / `'home'`. Это расхождение входа, не формулы Sets.

`Line` для PROFF собирает `document-service`
`ChristinaProffLinesByRow`: цветной/именной заголовок линии, члены `CHR…`,
`Required` = полный список артикулов линии с бланка (не только сопоставленные).

### 2.2 `PlanBudget` — три слоя

`PlanBudget` копирует вход, не мутирует его.

1. Нет ни одной PROFF-строки с `line`, **или** `target < before` → сразу
   `planBudgetCore` (снижение суммы комплекты не добирает).
2. Иначе валидация тем же контрактом строк: `planBudgetCore(original, before)`
   (цель = текущая сумма → цикл добора пустой, проверяются цены/NaN).
3. Цикл `pass = 0 … L+1` (`L = |baselines|` = число PROFF-линий):

```
CompleteChristinaLines          # атомарный добор комплектов
если total ≥ target: выход
planBudgetCore                  # обычный coverage-добор по SKU
если !complete: выход
CompleteChristinaLines          # финальный first-set; может снизить total
если total ≥ target: выход
```

Комментарий 1:1 с `origin/main`: финальное закрытие первых трёх комплектов
может **уменьшить** сумму (появилась 5% скидка). Тогда снова запускают
coverage-добор. Линия, уже закрывшая first-set, исключение повторно не
получает. Потолок проходов `L+2` — чтобы не крутиться, если скидка снова
опускает итог.

После цикла: `Before` = стартовый `ProcurementTotalCents`;
`LineCompletion` = имя линии, если qty вырос и линия есть в принятых шагах;
`Complete` = флаг ядра **и** `round(target×100) ≤ totalCents ≤ floor(target×105 + 1e-8)`.

### 2.3 Формулы CHRISTINA PROFF

Константы (`christina.go`): размер комплекта `3`, доп. скидка `0.05` поверх
уже заложенной основной. HOME скидку комплекта не получает.

`ChristinaLineGroups`:

- Группировка по `Line.ID`. `Valid`, только если `Required` без дублей,
  множество артикулов в строках = `Required`, нет unsafe / нулевой цены.
  Неполная или дублированная мета → `Sets = 0`, скидки нет.
- `Sets = floor(min(qty по членам линии))`. Если `Sets < 3` → `Sets = 0`.
  Некратные qty всё равно дают целые комплекты: `(18,18,18,3) → Sets = 3`
  (тест `TestChristinaLineGroupsAndTotal`).
- На **каждую** строку линии:

```
BaseCents   += round(Price × Quantity × 100)
SavingCents += round(Price × Sets × 0.05 × 100)   # если Sets > 0
NetCents     = BaseCents − SavingCents
```

Округление **по строке**, не по комплекту и не по сумме линии.
`ProcurementTotalCents` = Σ `round(Price×Qty×100)` по **всем** n строкам
минус Σ `SavingCents` по линиям. Комментарий в коде: never rounds each set
separately.

`BudgetBaselines`: `NetCents` каждой линии **на старте** `PlanBudget`. Потолок
роста 130% считается от этой карты и между шагами не сбрасывается.

`ProposeLineStep` — атомарный прыжок линии к следующему кратному 3:

```
nextSets = max(3, (Sets/3 + 1) × 3)    # 0→3, 3→6, 6→9, …
```

Все члены с `qty < nextSets` поднимаются до `nextSets`. Отказ целиком (ничего
не добавляется), если: incomplete metadata; цель уже достигнута (кроме
first-single); locked/excluded; coverage > норма категории + поставка или > 6
мес.; `Net` после шага > `floor(baseline×1.3)`; `added ≤ 0` или
`saved×100 < added×15` (прирост скидки < 15% добавки); итог >
`floor(цель_копейки × 1.05)`.

Исключение first-single: `Sets == 0`, меняется ровно один артикул, его qty
сейчас 0 — первые три комплекта можно закрыть без продаж и при полном складе.
Повторно нельзя.

`CompleteChristinaLines`: пока есть принятые предложения, берёт одно.
Порядок: больше A+, затем A, затем B в линии; затем `saved/added` по убыванию;
тай-брейк по `ID` по возрастанию. Жёсткий потолок **250 000** шагов комплектов.

### 2.4 `planBudgetCore` — coverage greedy

Чистый добор без знания линий, кроме того что стоимость шага уже с
комплектной скидкой.

Старт: `amount = ProcurementTotalCents`, `goal = round(target×100)`,
`up = goal > start`.

Пока не достигли цели, итерация:

1. `collectBudgetCands` — для каждой eligible строки (`!locked/excluded/unsafe`,
   `Price>0`, `Demand>0`, категория из `{C,B,A,A+}`) шаг `nextQuantity` (±
   кратность `Step`, вниз не ниже `Minimum` иначе 0). Кандидат с
   `cost = changeCost(...)`.
2. Вверх: сначала потолок покрытия 6 мес. (`nextMonths ≤ 6`); иначе `overSix`
   если нет `AllowOverSix`. Затем этап `C→B→A→A+`: первая категория, у которой
   `months < norm+delivery`. Затем `amount+cost ≤ floor(goal×1.05)`. Сортировка
   `sortBudgetUp`: меньше `months/(norm+delivery)`, затем ближе к остатку до
   цели по `|goal − amount − cost|`. Берётся `[0]`.
3. Вниз: сначала «защищённые» `nextMonths ≥ 1+delivery`; иначе `belowOne` если
   нет `AllowBelowOne`; иначе режут одну текущую категорию `C→B→A→A+`. Сортировка
   `sortBudgetDown`: больше `months`, затем ближе к остатку. Берётся `[0]`.
4. `rows[i].Quantity = q`, `amount += cost`.

Жёсткий потолок итераций **250 000** (`Достигнут предел вычислений. Уточните сумму.`).
Тот же потолок в `origin/main`.

Если вниз проскочили `amount < goal` и `reason == ""`: восстановление вверх по
категориям `A+→A→B→C`, пока `qty < Before` и `amount+cost ≤ goal`. Снова
`changeCost`.

Нормы покрытия (`budgetNorms`): C=2, B=2.5, A=3, A+=3.5 месяца.
`Coverage = (Stock + Transit − Outbound + qty×Unit) / Demand`.

### 2.5 `changeCost` — оракул шага

Эталон `origin/main:src/budgetPlanner.js`:

```javascript
// Changing the minimum of a PROFF line changes the discount on other rows.
// Evaluate the complete order; a fixed unit-price delta is incorrect here.
const changeCost = (r, q) => {
  if (r.group !== 'proff' || !r.line) return cents(r.price*q) - cents(r.price*r.quantity);
  const before = r.quantity, oldTotal = total();
  r.quantity = q;
  const cost = total() - oldTotal;
  r.quantity = before;
  return cost;
};
```

Go (`budget.go` `changeCost`) — то же:

- не PROFF: `round(Price×q×100) − round(Price×qty×100)` (уже не `round(Price×Δ)`);
- PROFF: два полных `ProcurementTotalCents`, qty временно подставляется.

`amount` в ядре — сумма этих дельт, без сверки с полным total до конца
`PlanBudget`. Оракул обязан совпадать с полным пересчётом **в копейке**, иначе
плывут фильтр 105% и ключ сортировки «ближе к остатку».

---

## 3. Почему n²

Обозначения:

- `n` — число строк заказа (SKU), вход `PlanBudget`.
- `L` — число PROFF-линий (`ChristinaLineGroups`).
- `s` — размер линии (`|Required|` / число членов). `Σ s ≤ n` (строка в одной
  линии). На актуальном PROFF-бланке типично `s ≈ 3…5`, `L ≈ 20`.

### 3.1 Стоимость `ProcurementTotalCents`

`christina.go` `ProcurementTotalCents`:

1. цикл по всем `n` строкам — `round(Price×Qty×100)`;
2. `ChristinaLineGroups` — один проход по `n`, затем по каждой линии `O(s)`
   (valid, min qty, base/saving).

Итого **`Θ(n)`** (эквивалентно `O(n + L·s)` при `Σ s ≤ n`). Не `O(L)` отдельно:
группы строятся из тех же строк. Это полный пересчёт заказа, не дельта.

`ChristinaLineGroups` внутри total вызывается **каждый раз заново**.

### 3.2 Сколько раз вызывается полный total

| Место | Вызовы `ProcurementTotalCents` | Когда |
| --- | --- | --- |
| `changeCost` PROFF | **2** на кандидата | каждый eligible PROFF в `collectBudgetCands` |
| `changeCost` не-PROFF | 0 (O(1) по своей строке) | — |
| старт `planBudgetCore` | 1 | `start` |
| восстановление вниз | 2 на шаг PROFF | вложенный цикл |
| `PlanBudget` обёртка | несколько | `before`, проверки `total ≥ target`, финальный total |
| `ProposeLineStep` | 1–2 (+ 2× `ChristinaLineGroups`) | на каждую из `L` линий за шаг комплектов |

Узкое место ядра: **каждая итерация** `planBudgetCore` для PROFF-заказа
считает total **`2p` раз**, где `p ≤ n` — число eligible PROFF-кандидатов
(обычно `p ~ n` на чистом PROFF).

### 3.3 Кандидаты × итерации

На одну итерацию ядра: до `n` кандидатов (все eligible с ненулевым шагом).
После фильтров меньше, но `changeCost` зовут **до** фильтров 6 мес. / этапа /
105% — в `collectBudgetCands`.

Итераций:

- шаг меняет **одну** строку на одну кратность;
- вверх ограничены покрытием 6 мес. (или `AllowOverSix`);
- жёсткий потолок **`I ≤ 250_000`** (константа, как в JS);
- типичный `I` — зазор суммы / цена шага, не `n` (десятки–тысячи, не 250k);
- патология: мелкий `Step`, большой зазор, все строки далеко от 6 мес. → `I`
  упирается в потолок.

Восстановление вниз: ещё `O(n × шагов_до_Before)` вызовов `changeCost`.

`CompleteChristinaLines`: за шаг `L` раз `ProposeLineStep` ≈ `L · Θ(n)`.
Практический `I_line` мал (рост линии ≤ 130% базы, шаг +3 комплекта). Потолок
тот же 250 000. Снаружи ещё до `L+2` проходов `PlanBudget`.

### 3.4 Итоговая асимптотика

```
T_total     = Θ(n)                         # полный ProcurementTotalCents
T_change    = Θ(n)  если PROFF, иначе O(1)
за итерацию ядра:  p ≤ n вызовов changeCost
                 → Θ(n²) на PROFF, Θ(n) без PROFF
весь planBudgetCore: O(I · n · T_change) ≤ O(250000 · n²) на PROFF
```

Коллоквиально **n²**: на итерацию `n` кандидатов × `O(n)` total. Так же
написано в коде (`ponytail` у `changeCost`).

Честно с потолком: **`O(I n²)` при `I ≤ 250000`**. Если снять потолок, `I` может
быть `Θ(n·Q)` (`Q` — шаги SKU до потолка покрытия) → уже `O(n³ Q⁰)` в
искусственном входе. Сортировка `O(n log n)` на фоне total не видна.

Линейный слой комплектов: `O(I_line · L · n)` с малым `I_line`. На реальном
PROFF он дешевле ядра, если ядро крутится сотни шагов.

### 3.5 Реальный бланк vs худший случай

Оценка размера **структуры файла** (число строк/листов), без сумм из ячеек.
Каталог `testdata/private/`, локально:

| Файл бланка | Лист | max_row | n (порядок) | PROFF |
| --- | --- | --- | --- | --- |
| `2026 08 25 Бланк заказа ANGIOPHARM.xlsx` | Бланк | 431 | ~350 SKU | нет (`p=0` → ядро `O(I n)`) |
| `_Бланк заказа Skin Synergy от 26.08.2026.xlsx` | Бланк заказа | 139 | ~100 | нет |
| `Бланк Заказа KLAPP август 2026 (1).xlsx` | БЛАНК ЗАКАЗА | 220 | ~200 | нет |
| `Актуальный_бланк PROFF.xlsx` и `… PROFF (1).xlsx` | Лист1 | 116 | ~88 `CHR` SKU, ~20 заголовков линий | да, `L ~ 20`, `s ~ 4` |

Таблица продаж задаёт спрос, не `n` заказа: 158…540 строк источника.

На PROFF ~90 SKU: итерация ≈ `90 × 2 × Θ(90)` обходов строк. При `I` в сотни —
миллионы операций, для unary RPC незаметно. На ANGIOPHARM ~350 без PROFF n²
нет.

Худший синтетика: `n = 400` все eligible PROFF, `I = 250000` →
`250000 × 400 × 2 × Θ(400)` ≈ 8·10¹⁰ касаний строк — уже «висит, уточните
сумму» на потолке, не на реальном бланке. Комментарий в коде: revisit only if
PROFF line counts grow large enough.

Циклы (для таблицы в ревью):

| Цикл | Файл:функция | Стоимость тела | Сколько раз |
| --- | --- | --- | --- |
| Σ gross + groups | `christina.go` `ProcurementTotalCents` | Θ(n) | см. таблицу вызовов |
| кандидаты | `budget.go` `collectBudgetCands` | `n × changeCost` | раз за итерацию ядра |
| PROFF delta | `budget.go` `changeCost` | 2× total | каждый PROFF-кандидат |
| greedy шаг | `budget.go` `planBudgetCore` | выбрать 1 из кандидатов | `I ≤ 250000` |
| restore | `planBudgetCore` (вниз, после цикла) | `changeCost` | шаги до `Before` |
| комплекты | `christina.go` `CompleteChristinaLines` | `L × ProposeLineStep` | `I_line ≤ 250000`, обычно ≪ L |
| обёртка | `budget.go` `PlanBudget` | complete + core + complete | `≤ L+2` |

---

## 4. Почему наивный инкремент ломает 1:1

«Прибавить `Price × Step` к сумме» не равно `changeCost`, уже на публичном
фиxture `muse(18,18,18,3)` при `Price=100`:

1. **Копейки по строке.** `round(p·q') − round(p·q) ≠ round(p·Δq)`. Не-PROFF
   ветка `changeCost` уже считает так. Полкопейки накапливаются в ключе
   сортировки `|goal − amount − cost|` → другой победитель → другая
   последовательность qty.
2. **Sets по всей линии.** `Sets = floor(min qty)`. Шаг по текущему минимуму
   может поднять Sets у **всех** братьев. `SavingCents` пересчитывается по
   каждому SKU: `round(Price × Sets × 0.05 × 100)`. Дельта одной строки —
   ещё и скидка соседей с неизменным qty.
3. **First-set снижает total.** Появление `Sets=3` вычитает скидку; `cost`
   шага может быть меньше валовой добавки и даже отрицательным относительно
   «наивной» цены. Ядро и финальный complete завязаны на это.
4. **Фильтры едят cost.** Отсев `amount+cost ≤ 105%` и тай-брейк «ближе к
   цели» используют оракул. Неверная дельта на 1 копейку = другой набор
   кандидатов или другой `[0]`.
5. **`amount` не пересчитывается.** Ошибка оракула не самоисправляется до
   финального `ProcurementTotalCents` в обёртке; `planBudgetCore.Total`
   есть накопленный `amount`.

Поэтому оптимизация без оракула ≡ полному total — это другой алгоритм, не
ускорение текущего.

---

## 5. Корректный путь оптимизации (только постановка)

Цель: `changeCost_fast(rows, i, q) == changeCost_full(rows, i, q)` как `int64`
на всех шагах, которые ядро реально задаёт. Тогда qty/total/comments совпадут,
потому что сортировка и фильтры видят те же `cost`.

Минимальная идея (не делать сейчас):

- хранить per-row gross cents и per-line `{min, Sets, SavingCents, NetCents}`;
- не-PROFF: как сейчас, O(1);
- PROFF: обновить одну линию за `O(s)` (новый min/Sets, saving по членам с тем
  же `round` по строке), остальной заказ не трогать;
- сверка: прогон полного total vs оракул на фикстурах `christina_test.go` и
  на реальном PROFF (локально, не в git).

Нельзя менять (иначе это уже смена спецификации, не ускорение):

- порядок фильтров вверх: 6 мес. **до** этапа C→B→A→A+ **до** 105%;
- компараторы `sortBudgetUp` / `sortBudgetDown`;
- выбор ровно `candidates[0]` (без «почти так же близко»);
- ранжирование `CompleteChristinaLines` (A+/A/B, затем ratio, затем id);
- first-single, 130% от **стартовых** baselines, порог 15%;
- потолок 250 000 и тексты `reason`;
- формулу `Sets` / поэлементный `round`.

Не считать оптимизацией: эвристику «сначала дорогие», параллельный перебор
кандидатов с другой сортировкой, кэш total без инвалидации Sets.

---

## 6. E2E: брендовые пары бланк × таблица продаж

Бюджет гоняется на **matched pairs** — бланк и таблица продаж одного бренда.
Не декартово всех файлов `Бланки/` × `таблицы продаж/`. Золотые ₽ в git не кладут.

Opt-in Playwright: `npm run test:e2e:real --prefix frontend`
(`order-real-data.real.spec.js`). Пары собирает `frontend/e2e/brandPairs.js`
(только basename). Нет `testdata/private/Бланки` и `таблицы продаж` или нет
`REAL_E2E_*` — skip всего real-сьюта. Mock `npm run test:e2e` / `test:ui` real
не запускают (`testIgnore: **/*.real.spec.js`).

На каждую пару: логин, бланк+продажи этого бренда, экран заказа, «Заказ до
суммы». Для Christina PROFF обязательны скидка, превью и комплекты, если блок
есть. Ассерты: `complete`, «Изменено позиций» ≥ 1, комментарий комплекта если
UI его показал. Абсолютные ₽ не фиксируют.

Северного склада в этих папках нет — North сюда не добавляли.

`TestPrivateNorthBudgetPairs` в document-service по-прежнему четыре брендовые
пары и skip, если каталога нет.

Чужой бренд не смешивать. Файл без бренда в имени — сирота, в suite не входит.

### 6.1 Правило матча по имени файла

Смотреть **только basename** (не ячейки книги). Бренд из имени; город, дата,
`(1)`, ведущий `_` — не ключ.

Нормализация `norm(name)`:

1. Unicode NFC (на диске «Актуальный» у PROFF-бланков — NFD);
2. нижний регистр; `ё` → `е`;
3. не-буквы/цифры → пробел, схлопнуть пробелы, trim
   (хвостовой пробел в `Тюмень .xlsx` отпадает).

Ключ бренда — первое совпадение по `contains` в `norm` (как
`KeyFromNomenclatureGroup` + alias `proff` для бланков без слова Christina):

| Порядок | Маркеры в `norm` | Ключ |
| --- | --- | --- |
| 1 | `ангио` или `angio` | `angiopharm` |
| 2 | `кристин` или `christina` | `christina` |
| 3 | `klapp` или `клапп` | `klapp` |
| 4 | (`skin` и `synerg`) или (`скин` и `синердж`) | `skin_synergy` |
| 5 | `levissim` или `левисим` | `levissime` |
| 6 | `sothys` или `сотис` | `sothys` |
| 7 | `novacutan` или `новакутан` | `novacutan` |
| 8 | `proff` / `проф` / `prof` (бланк без п.2) | `christina` |

Нет маркера → сирота. Два маркера разных брендов не ожидаются; если
случится — сирота, не угадывать.

Пара: один ключ у бланка и у таблицы. Несколько файлов бренда — **внутри
бренда все × все**, город не режет. Чужой бренд — никогда.

### 6.2 Каталог локально (имена, без цифр из ячеек)

Бланки (5) → ключ:

| Файл | Ключ |
| --- | --- |
| `2026 08 25 Бланк заказа ANGIOPHARM.xlsx` | `angiopharm` |
| `_Бланк заказа Skin Synergy от 26.08.2026.xlsx` | `skin_synergy` |
| `Актуальный_бланк PROFF.xlsx` | `christina` (п.8, `proff`) |
| `Актуальный_бланк PROFF (1).xlsx` | `christina` (копия листа п.3) |
| `Бланк Заказа KLAPP август 2026 (1).xlsx` | `klapp` |

Таблицы продаж (8) → ключ:

| Файл | Ключ |
| --- | --- |
| `Ангио Сургут.xlsx` | `angiopharm` |
| `Ангио Тюмень .xlsx` | `angiopharm` |
| `Клапп Тюмень .xlsx` | `klapp` |
| `Сургут Клапп.xlsx` | `klapp` |
| `Кристина Сургут .xlsx` | `christina` |
| `Кристина Тюмень .xlsx` | `christina` |
| `Скин Синерджи Сургут.xlsx` | `skin_synergy` |
| `Скин Синерджи Тюмень .xlsx` | `skin_synergy` |

Matched pairs — **10**, не 40:

| Бренд | Бланки | Таблицы продаж | Пар |
| --- | --- | --- | --- |
| `angiopharm` | 1 | 2 (Сургут, Тюмень) | 2 |
| `skin_synergy` | 1 | 2 | 2 |
| `klapp` | 1 | 2 | 2 |
| `christina` | 2 PROFF | 2 | 4 |

Развёртка:

1. ANGIOPHARM × `Ангио Сургут.xlsx`
2. ANGIOPHARM × `Ангио Тюмень .xlsx`
3. Skin Synergy × `Скин Синерджи Сургут.xlsx`
4. Skin Synergy × `Скин Синерджи Тюмень .xlsx`
5. KLAPP × `Клапп Тюмень .xlsx`
6. KLAPP × `Сургут Клапп.xlsx`
7. `…PROFF.xlsx` × `Кристина Сургут .xlsx`
8. `…PROFF.xlsx` × `Кристина Тюмень .xlsx`
9. `…PROFF (1).xlsx` × `Кристина Сургут .xlsx`
10. `…PROFF (1).xlsx` × `Кристина Тюмень .xlsx`

**Сироты:** нет. Novacutan / Levissime / Sothys в этих папках нет — это
не сироты, а пустой бренд.

Пробел в покрытии, не сирота: HOME-бланка CHRISTINA в `Бланки/` нет;
обе таблицы «Кристина» идут с PROFF.

### 6.3 Ограничения, которые остаются

- `testdata/private/` в `.gitignore`. В CI файлов нет.
- `.github/workflows/verify.yml` не гоняет `test:e2e:real` и не читает private.
- Opt-in: `REAL_E2E_LOGIN` / `REAL_E2E_PASSWORD` / `REAL_E2E_OWNER_*`, каталог
  на машине или `ORDER_FILL_PRIVATE_TESTDATA`.
- В git не класть абсолютные суммы заказа, скидки с бланка, эталонные qty
  коммерческого прогона. Сверять локально: `complete`, знак дельты, стабильность
  qty между полным total и будущим оракулом.

---

## 7. Не делать сейчас

- Не менять `planBudgetCore` / `changeCost` / `CompleteChristinaLines` /
  формулы Sets/saving.
- Не писать инкрементальный оракул и бенчмарки «вместо» постановки.
- Не добавлять `test:e2e:real` в CI: private xlsx нет в git.
- Не сливать в `artemch` / не коммитить этот файл без явной просьбы (файл
  можно оставить в рабочей ветке `feat/go-christina-budget`).
- Не чинить в этом изменении контракт `group: blank-1` vs `"proff"` — это
  отдельный вход, не сложность цикла.
- Не подгонять порядок кандидатов «побыстрее».
