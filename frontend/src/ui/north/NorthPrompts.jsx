import { formatNorthQuantity } from "../../features/north/northPlan.js";
import { Modal } from "../widgets.jsx";

export function NorthPrompts({ mergePrompt, shortPrompt, onCancelMerge, onConfirmMerge, onCancelShort, onConfirmShort }) {
  return (
    <>
      {mergePrompt && (
        <Modal
          title="Соединить бланки городов"
          cancelLabel="Назад"
          confirmLabel="Соединить"
          onCancel={onCancelMerge}
          onConfirm={onConfirmMerge}
        >
          <p className="mb-2">Будут объединены эти бланки:</p>
          {(mergePrompt.result.plan.confirmationGroups?.length
            ? mergePrompt.result.plan.confirmationGroups.map((group) => `${group.city.label}: ${group.variants.join(", ")}`)
            : mergePrompt.entries.map((entry) => entry.file.name)
          ).map((text) => (
            <div key={text} className="mt-1 rounded-lg border border-[var(--color-line)] bg-[var(--color-ground)] px-3 py-2 text-[13px] text-[var(--color-ink)]">
              {text}
            </div>
          ))}
        </Modal>
      )}

      {shortPrompt && (
        <Modal
          title="Есть позиции ниже потребности"
          cancelLabel="Назад"
          confirmLabel="Все равно продолжить"
          onCancel={onCancelShort}
          onConfirm={onConfirmShort}
        >
          {`По этим позициям факт у поставщика меньше общей нехватки Тюмени и северных городов.\n\n${shortPrompt.slice(0, 8).map((item) => `• ${item.name}: факт ${formatNorthQuantity(item.actual) || "0"}, нужно ${formatNorthQuantity(item.need)}`).join("\n")}${shortPrompt.length > 8 ? `\nЕще позиций: ${shortPrompt.length - 8}` : ""}\n\nМожно вернуться и поправить цифры или продолжить скачивание как есть.`}
        </Modal>
      )}
    </>
  );
}
