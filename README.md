# Заполнение бланка заказа — Vercel версия

Это frontend-only версия инструмента. Excel-файлы обрабатываются прямо в браузере:

- файлы не отправляются на сервер;
- backend не нужен;
- подходит для бесплатного деплоя на Vercel;
- сохраняет `.xlsx`, меняя только нужные ячейки количества.

## Локальный запуск

```bash
npm install
npm run dev
```

Открыть:

```text
http://127.0.0.1:3200
```

## Проверка

```bash
npm run test:workbook
npm run build
```

## Деплой на Vercel

1. Загрузите папку `order-fill-vercel` в GitHub-репозиторий.
2. В Vercel создайте новый проект из этого репозитория.
3. Vercel сам подхватит настройки из `vercel.json`:

```text
Build Command: npm run build
Output Directory: dist
Install Command: npm ci
```

После деплоя Vercel выдаст прямую ссылку для закупщика.
