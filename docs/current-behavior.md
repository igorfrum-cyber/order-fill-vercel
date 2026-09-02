# Current Workbook Behavior

This document captures the behavior of the current frontend-only workbook tool before the service rewrite.

## Runtime Model

The app runs entirely in the browser. Users upload Excel workbooks through the UI, JavaScript parses and edits the workbook XML locally, and the browser downloads generated files. The current runtime does not upload files to a backend, persist jobs, or store source files.

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

Rows are included when they need source-data cleanup or manual review:

- name-only matches;
- article matches with suspicious name differences;
- rows present in the blank but missing from the source workbook;
- source duplicate rows.

The CSV uses semicolon separators and includes reason, blank row, blank article/name, source row, source article/name, similarity, recommended quantity, inserted quantity, and duplicate candidates.

### File Downloads

For standard order fill, downloads include:

- filled supplier blank;
- filled source order workbook.

For North mode, downloads include:

- one or more summary blanks;
- transfer files by city;
- NOVACUTAN order table when generated.

Workbook output forces formula recalculation on open and removes `calcChain.xml`.

## Report Row Statuses

### `matched`

The blank row matched one or more source rows by article. The best source candidate was selected, name similarity was acceptable, and the row can be filled or left blank according to quantity rules.

### `matched_by_name`

The blank row matched a source row by name fallback. This is used when there is no article candidate and the matched source item does not require a positive order quantity.

### `warning_name_differs`

The blank row matched by article, but the source name and blank name similarity is below the warning threshold. The row stays editable and is shown in the priority review section.

### `warning_name_only`

The blank row had no article match and was matched only by name to a source row with positive recommended quantity. The blank quantity is cleared until reviewed, and the row is shown as priority.

### `left_blank_nonpositive`

The row matched a source item, but the calculated inserted quantity is empty because the recommendation is below the minimum threshold or non-positive after brand rules. The blank quantity cell is cleared.

### `not_in_source`

The blank row could not be matched to the source workbook by article or accepted name fallback. The blank quantity cell is cleared and the row is not editable because there is no source row to update.

### `not_in_blank`

The source item is not represented by a matched blank row. These rows are synthesized in the UI after processing by comparing matched source rows with source items.

### `source_duplicate`

The source workbook has multiple rows for the same normalized article. Duplicate source candidates are included in the report so the user can correct source data or choose the right row manually.
