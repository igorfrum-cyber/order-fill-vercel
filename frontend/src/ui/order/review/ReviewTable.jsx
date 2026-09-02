import { rowKey } from "../../../features/order/reviewEdits.js";
import { reviewTableHeaders } from "../../../features/report/rowPresentation.js";
import { ReportRow } from "./ReportRow.jsx";

export function ReviewTable({
  rows,
  edits,
  expanded,
  invalidKeys,
  acknowledgedDuplicates,
  boxLabel,
  onToggle,
  onEdit,
  onAcknowledge,
}) {
  return (
    <div className="flex-1 overflow-auto px-6 py-4">
      <table className="w-full min-w-[1280px] table-fixed border-separate border-spacing-0">
        <colgroup>
          <col className="w-12" />
          <col className="w-[11%]" />
          <col />
          <col className="w-[8%]" />
          <col className="w-[8%]" />
          <col className="w-[8%]" />
          <col className="w-[9%]" />
          <col className="w-[13%]" />
          <col className="w-[8%]" />
          <col className="w-[18%]" />
        </colgroup>
        <thead>
          <tr className="text-left">
            {reviewTableHeaders().map((header) => (
              <th
                key={header.key}
                className={`sticky top-0 z-10 whitespace-nowrap bg-[var(--color-ground)] px-4 pb-3.5 pt-2 text-[13px] font-medium tracking-wide text-[var(--color-ink-faint)] ${
                  header.align === "right" ? "text-right" : ""
                }`}
              >
                {header.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr>
              <td colSpan={10} className="py-16 text-center text-[15px] text-[var(--color-ink-faint)]">
                Нет позиций в этой категории
              </td>
            </tr>
          )}
          {rows.map((row) => (
            <ReportRow
              key={rowKey(row)}
              row={row}
              edit={edits.get(rowKey(row))}
              expanded={expanded === rowKey(row)}
              invalid={invalidKeys.has(rowKey(row))}
              acknowledged={acknowledgedDuplicates.has(rowKey(row))}
              boxLabel={boxLabel}
              onToggle={() => onToggle(rowKey(row))}
              onEdit={onEdit}
              onAcknowledge={(next) => onAcknowledge(rowKey(row), next)}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
