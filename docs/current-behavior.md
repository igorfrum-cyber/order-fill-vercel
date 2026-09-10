# Historical Frontend-Only Workbook Behavior

> This is a historical reference for the frontend-only implementation before the
> service rewrite. For the current backend v2 runtime, see
> [Architecture](./ARCHITECTURE.md), [Service Boundaries](./service-boundaries.md),
> and the repository [README](../README.md).

## Runtime Model

That version ran entirely in the browser. Users uploaded Excel workbooks through the UI, JavaScript parsed and edited the workbook XML locally, and the browser downloaded generated files. It did not upload files to a backend, persist jobs, or store source files.

Main modules:

- `src/app.js`: UI state, file inputs, report rendering, manual edits, CSV export, and download links.
- `src/workbookProcessor.js`: workbook parsing/writing, period validation, column detection, normalization, matching, brand rules, quantity calculation, and final workbook edits.

## User Scenarios

### Standard Order Fill

1. User selects a brand and order month.
2. User uploads a source workbook from 1C and a supplier blank workbook.
3. The app validates that the source workbook period matches the selected order month.
4. The processor detects source columns by normalized headers: article, product name, recommended order, stock, in transit, ordered fact, and comment.
5. The processor detects blank columns by brand-specific rules.
6. Blank rows are matched to source rows by article first. If no article match exists, eligible no-article source rows can be matched by name.
7. Recommended quantities are rounded according to the brand rule and written to the blank quantity column.
8. The UI shows metrics, priority rows, and all other report rows.
9. User can download the filled blank and the filled source workbook after final validation.

### CHRISTINA HOME/PROFF

CHRISTINA uses two blank files: HOME and PROFF. The UI exposes separate file inputs for both. Each blank is processed with the same source workbook and brand rule, then report rows are combined for display and final edits.

CHRISTINA order quantities are rounded to a multiple of 3 when the adjustment rule allows it. The automatic comment is `до кратности 3`.

### North Mode

North mode merges filled city blanks into a shared plan.

1. User switches to the North mode.
2. User uploads one or more filled city blanks. CHRISTINA North mode accepts separate HOME and PROFF city blanks.
3. The app detects city from workbook content or file name. Supported cities are Tyumen, Surgut, Nizhnevartovsk/Vartovsk, and Urengoy.
4. The processor rejects duplicate city/type uploads.
5. The processor extracts city needs from uploaded blanks and builds a combined plan.
6. If a Tyumen source workbook is uploaded, the plan accounts for Tyumen stock, in-transit quantities, and target stock.
7. The plan separates quantities covered from Tyumen stock, quantities to order from supplier, and transfer quantities.
8. User can edit city quantities and supplier fact quantities before finalization.
9. Finalization writes the summary blank, transfer files, and for NOVACUTAN an order table when needed.

### Manual Edits

Report rows contain editable quantity and comment fields when the row is tied to a source row. Manual edits are validated before files are generated.

Rules:

- If a user changes a quantity away from the calculated inserted value, a non-empty comment is required.
- If a value is returned to the calculated recommendation, the source `Заказано по факту` and `Комментарий` cells are cleared.
- If a row already has `Заказано по факту`, that value is used as the inserted quantity and must keep a comment when confirmed manually.
- Invalid rows are highlighted in the report and file generation is blocked until fixed.

### CSV Issue Report

The UI can generate `отчет для исправления в 1С.csv`.

Rows are included when they help clean 1C or the blank:

- unresolved duplicates and ambiguous Chestny Znak merges (`needs_decision`);
- blank rows missing from 1C (`not_in_source`);
- source rows with need missing from the blank (`not_in_blank`);
- volume or form conflicts;
- name-only matches.

The CSV uses semicolon separators. Reason text is canonical (for example
`неоднозначный дубль`, `конфликт объёма`), not a raw similarity percent.

### File Downloads

Download is blocked until matching decisions and commented quantity edits are
resolved:

- unresolved duplicate articles;
- ambiguous Chestny Znak merges;
- positive name-only matches;
- manual quantity changes without a comment.

A single article match that only has `check_name_or_volume` does not block
download. `order_not_needed` is a normal result, not an error.

For standard order fill, downloads include:

- filled supplier blank;
- filled source order workbook.

For North mode, downloads include:

- one or more summary blanks;
- transfer files by city;
- NOVACUTAN order table when generated.

Workbook output forces formula recalculation on open and removes `calcChain.xml`.

## Report Row Categories

New reports use `category` and `match_reasons`. The same six categories appear
in `standard` and `smart` matching mode. `status` remains on the payload for
older clients.

Review tabs, in this order: `needs_decision`, `not_in_source`,
`check_name_or_volume`, `not_in_blank`, `to_order`, `order_not_needed`, then
all rows.

### `needs_decision`

The matcher cannot choose one source row. Typical reasons: duplicate articles
or an ambiguous Chestny Znak merge. Quantity is not written until the user
chooses.

### `not_in_source`

The blank row has no accepted source match. The blank quantity cell is cleared
and the row is not editable.

### `check_name_or_volume`

The pair is usable, but name, volume, or form should be checked. A single
article match in this bucket does not block download.

### `not_in_blank`

The source item needs an order and is not represented by a matched blank row.

### `to_order`

The pair is trusted and the calculated quantity is positive.

### `order_not_needed`

The pair is trusted, but the calculated quantity is empty (below the brand
minimum or non-positive). This is a normal outcome, not an error.

### Legacy `status`

Older completed jobs may only have `status`. The UI maps those values onto the
categories above: `source_duplicate` and `warning_name_only` → `needs_decision`;
`warning_name_differs` → `check_name_or_volume`; `left_blank_nonpositive` →
`order_not_needed`; `matched` / `matched_by_name` → `to_order` or
`order_not_needed` from whether a quantity was inserted.
