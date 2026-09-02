#!/usr/bin/env python3
"""Build a large 1C-shaped source workbook by repeating product rows of a Tyumen export."""

from __future__ import annotations

import argparse
import io
import re
import zipfile
from pathlib import Path

ROW_RE = re.compile(r"<row\b[^>]*>.*?</row>", re.DOTALL)
ROW_NUM_RE = re.compile(r'\sr="(\d+)"')
CELL_RE = re.compile(r"<c\b[^>]*/>|<c\b[^>]*>.*?</c>", re.DOTALL)
CELL_COL_RE = re.compile(r'r="([A-Z]+)\d+"')
CELL_REF_RE = re.compile(r'(r=")([A-Z]+)(\d+)(")')
DIM_RE = re.compile(r'(<dimension[^>]*ref=")([^"]+)(")')
SHEET = "xl/worksheets/sheet1.xml"

# Keep the Tyumen header intact. Product rows only need the 1C columns the
# fill engine reads (article, name, stock, in-transit, recommended, fact).
PRODUCT_COLUMNS = {"A", "D", "AB", "AC", "AE", "AF", "AG", "AH", "AI"}


def remap_row(xml: str, new_row: int) -> str:
    old = ROW_NUM_RE.search(xml)
    if not old:
        return xml
    old_row = old.group(1)
    xml = ROW_NUM_RE.sub(f' r="{new_row}"', xml, count=1)
    return CELL_REF_RE.sub(lambda m: f"{m.group(1)}{m.group(2)}{new_row}{m.group(4)}" if m.group(3) == old_row else m.group(0), xml)


def slim_product_row(xml: str) -> str:
    opening = re.match(r"<row\b[^>]*>", xml)
    if not opening:
        return xml
    kept = []
    for cell in CELL_RE.findall(xml):
        column = CELL_COL_RE.search(cell)
        if column and column.group(1) in PRODUCT_COLUMNS:
            kept.append(cell)
    return opening.group(0) + "".join(kept) + "</row>"


def generate(src: Path, dest: Path, rows: int, slim: bool = False) -> None:
    with zipfile.ZipFile(src) as zin:
        sheet = zin.read(SHEET).decode("utf-8")
        others = {name: zin.read(name) for name in zin.namelist() if name != SHEET}

    found = list(ROW_RE.finditer(sheet))
    if not found:
        raise SystemExit("no <row> elements in the Tyumen sheet")

    header: list[str] = []
    products: list[str] = []
    for match in found:
        xml = match.group(0)
        number = int(ROW_NUM_RE.search(xml).group(1))
        if number <= 15:
            header.append(xml)
        else:
            products.append(xml)
    if not products:
        raise SystemExit("no product rows after the Tyumen header")

    prefix = sheet[: found[0].start()]
    suffix = sheet[found[-1].end() :]

    out = io.StringIO()
    out.write(prefix)
    for xml in header:
        out.write(xml)
    next_row = 16
    remaining = rows
    while remaining > 0:
        for xml in products:
            if remaining <= 0:
                break
            body = slim_product_row(xml) if slim else xml
            out.write(remap_row(body, next_row))
            next_row += 1
            remaining -= 1
    out.write(suffix)
    body = out.getvalue()
    last_col = "AI"
    body = DIM_RE.sub(rf"\g<1>{last_col}{next_row - 1}\3", body, count=1)

    dest.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(dest, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as zout:
        for name, data in others.items():
            zout.writestr(name, data)
        zout.writestr(SHEET, body.encode("utf-8"))
    print(f"wrote {dest} ({dest.stat().st_size} bytes, product rows={rows}, last_row={next_row - 1})")


def benchmark_sizes(max_rows: int, step: int) -> list[int]:
    if max_rows < 1:
        raise SystemExit("--rows must be positive")
    if step < 1:
        raise SystemExit("--series-step must be positive")

    sizes = [size for size in (1000, 5000) if size <= max_rows]
    current = 10000
    while current <= max_rows:
        sizes.append(current)
        current += step
    if not sizes:
        sizes.append(max_rows)
    return sizes


def main() -> None:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=root / "testdata/private/Ангио Тюмень .xlsx")
    parser.add_argument("--out", type=Path, default=root / "testdata/private/source_100000.xlsx")
    parser.add_argument("--rows", type=int, default=100_000)
    parser.add_argument("--series", action="store_true", help="generate benchmark files up to --rows")
    parser.add_argument("--series-step", type=int, default=5000)
    parser.add_argument("--out-dir", type=Path, default=root / "testdata/private")
    parser.add_argument("--slim", action="store_true", help="drop unused 1C month columns from product rows")
    args = parser.parse_args()
    if args.series:
        for rows in benchmark_sizes(args.rows, args.series_step):
            generate(args.source, args.out_dir / f"source_{rows}.xlsx", rows, slim=args.slim)
        return
    generate(args.source, args.out, args.rows, slim=args.slim)


if __name__ == "__main__":
    main()
