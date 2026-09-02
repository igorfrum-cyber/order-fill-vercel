# Test Data

`scripts/test-workbook.mjs` always runs smoke tests that generate minimal workbooks in memory. These smoke tests cover portable behavior and do not require private files.

Optional full-regression workbooks can be placed in `testdata/private/`. That directory is git-ignored because source workbooks may contain commercial data.

Expected optional file names:

- `angiopharm-source.xlsx`
- `angiopharm-blank.xlsx`
- `christina-home-blank.xlsx`
- `christina-proff-blank.xlsx`
- `levissime-blank.xlsx`
- `levissime-blank.xls`
- `levissime-current-source.xlsx`
- `levissime-current-blank.xls`
- `sothys-blank.xls`
- `angiopharm-urengoy-source.xlsx`
- `angiopharm-current-source.xlsx`
- `angiopharm-current-filled-blank.xlsx`
- `angiopharm-chz-source.xlsx`

When optional files are absent, the corresponding private regression block is skipped with a warning.
