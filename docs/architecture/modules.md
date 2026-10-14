# Модули и владельцы

Статус: действующий. Область: браузерная версия после реорганизации. Проверено: 13 октября 2026 года.

`src/app.js` запускает приложение. `src/workbookProcessor.js` и прежние бюджетные точки входа только переэкспортируют совместимый API. Внутренний код импортирует владельца напрямую; тесты прежнего API намеренно используют фасады.

| Ответственность | Владелец в src | Проверка |
|---|---|---|
| Бренды, партии, кратности, единицы | `domain/brands/rules.js`, `domain/order/quantities.js`, `budget-rules.js` | `test:budget`, parity |
| Потребность, категории, новинки, периоды | `domain/calculation/` | `test:tyumen`, `test:budget`, parity |
| Артикулы, объёмы, ЧЗ, соответствия | `domain/matching/` | `test:tyumen`, parity |
| Остатки и перемещения | `domain/warehouse/stock.js` | `test:tyumen` |
| Подбор суммы, комплекты, цены | `domain/budget/` | `test:budget`, `test:christina` |
| ZIP/XML, ячейки, стили, формулы | `infrastructure/excel/archive.js`, `xml.js` | побайтовый parity |
| Колонки и исходники | `infrastructure/excel/columns.js`, `source.js`, `cities.js` | parity, `test:tyumen` |
| Форматы бланков | `infrastructure/excel/blanks.js` | parity по брендам |
| Цены и остатки в Excel | `infrastructure/excel/pricing.js`, `stocks.js` | `test:budget`, parity |
| 1С-перемещения и таблицы Севера | `infrastructure/excel/transfer-tables.js` | E2E скачиваний |
| Заказ, ручные правки | `workflows/order/` | parity, E2E |
| Сведение складов Тюмени | `workflows/tyumen/merge.js` | `test:tyumen`, parity |
| Пересчёт | `workflows/recalculation/recalculate.js` | parity, E2E |
| Север: наличие, план, экспорт | `workflows/north/` | `test:tyumen`, parity, E2E |
| Адаптация отчёта к бюджету | `workflows/budget/adapters.js` | `test:budget` |
| Запуск, режимы, сбросы | `app/bootstrap.js`, `lifecycle.js` | E2E |
| UI, события, ввод | `ui/order/`, `north/`, `tyumen/` | UI baseline |
| Бюджетный диалог и цены | `ui/budget/`, `pricing/` | E2E применения и отмены |
| Загрузка, скачивание, пересчёт UI | `ui/files/`, `downloads/`, `recalculation/` | E2E |

Зависимости: `domain -> domain/shared`; `infrastructure/excel -> domain/shared/Excel`; `workflows -> infrastructure/domain/shared`; `ui/app -> workflows/domain/infrastructure`. Циклы, domain-зависимости на DOM/File/Blob/XML/сеть и внутренние импорты фасадов запрещены автоматически.

Временная граница: `infrastructure/excel/source.js` применяет чистые расчёты к изменяемому workbook. Контракт этого адаптера сохранён. `blanks.js` (около 450 строк) имеет одну обязанность: распознавать поддерживаемые форматы бланков. Разделение ради количества файлов не требуется.

`domain/north/cities.js` хранит предметные города; `ui/north/display-cities.js` — прежние экранные подписи и раскладку. Это разные контракты, не дубли формулы. Все прежние workbook-функции и их владельцы перечислены в `docs/plans/export-owners.json`; SHA тел дополняет parity результатов.
