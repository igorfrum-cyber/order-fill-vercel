export const loginFailedMessage =
  "Не получилось войти. Проверьте логин и пароль или запросите новую ссылку.";

export const loginAccessHint =
  "Нет доступа? Попросите владельца или администратора компании прислать приглашение.";

const ROLE_LABELS = {
  purchaser: "Закупщик",
  company_admin: "Администратор компании",
  company_owner: "Владелец компании",
  platform_admin: "Администратор сервиса",
};

export function roleLabel(role) {
  return ROLE_LABELS[role] || "Пользователь";
}

export function accessSummaryForRole(role) {
  if (role === "platform_admin") {
    return "Вы можете создавать компании, помогать с доступом и смотреть историю по всем компаниям.";
  }
  if (role === "company_owner") {
    return "Вы управляете доступом сотрудников и видите выгрузки своей компании.";
  }
  if (role === "company_admin") {
    return "Вы приглашаете сотрудников, сбрасываете доступ и видите выгрузки своей компании.";
  }
  return "Вы создаёте выгрузки, проверяете строки и скачиваете готовые файлы.";
}

export function profileCompanyLabel(me) {
  if (!me?.company_id || me.role === "platform_admin") return "Сервис";
  return me.company_name || "Ваша компания";
}

export function profileFields(me) {
  return [
    { label: "Логин", value: me.login, editable: false },
    { label: "Компания", value: profileCompanyLabel(me), editable: false },
    { label: "Доступ", value: roleLabel(me.role), editable: false },
  ];
}

export const accountPasswordHint =
  "Ваш пароль знаете только вы. Если доступ нужен другому человеку, создайте отдельного пользователя.";

export const logoutEverywhereLabel = "Выйти со всех устройств";

export const logoutEverywhereConfirm =
  "Вы выйдете здесь и на других устройствах. Войти снова можно будет по логину и паролю.";

export function tourForRole(role) {
  if (role === "platform_admin") {
    return [
      {
        target: "companies",
        placement: "bottom",
        title: "Компании",
        body: "Сначала создайте компанию и выберите её в списке.",
      },
      {
        target: "company-select",
        placement: "bottom",
        title: "Фильтр по компании",
        body: "Историю выгрузок можно сузить до одной компании.",
      },
      {
        target: "users",
        placement: "bottom",
        title: "Пользователи",
        body: "Выберите компанию и пригласите администратора по ссылке.",
      },
      {
        target: "jobs",
        placement: "top",
        title: "Выгрузки",
        body: "Здесь только просмотр. Новую выгрузку создаёт закупщик или администратор компании.",
      },
      {
        target: "help",
        placement: "bottom",
        title: "Справка",
        body: "Если что-то непонятно, откройте знак вопроса — подсказка всегда рядом.",
      },
    ];
  }
  if (role === "company_owner" || role === "company_admin") {
    return [
      {
        target: "users",
        placement: "bottom",
        title: "Сотрудники",
        body: "Пригласите людей одноразовой ссылкой. Пароль они поставят сами.",
      },
      {
        target: "order",
        placement: "bottom",
        title: "Бланк закупки",
        body: "Отсюда загружают таблицу продаж из 1С и бланк поставщика.",
      },
      {
        target: "jobs",
        placement: "top",
        title: "История",
        body: "Готовые и незавершённые выгрузки открываются из этой таблицы.",
      },
      {
        target: "help",
        placement: "bottom",
        title: "Справка",
        body: "Знак вопроса открывает короткие ответы, если запутались.",
      },
    ];
  }
  return [
    {
      target: "order",
      placement: "bottom",
      title: "Бланк закупки",
      body: "Начните отсюда: загрузите таблицу продаж из 1С и бланк поставщика.",
    },
    {
      target: "north",
      placement: "bottom",
      title: "Север",
      body: "Если нужно соединить бланки городов с таблицей Тюмени — этот сценарий.",
    },
    {
      target: "jobs",
      placement: "top",
      title: "История выгрузок",
      body: "Готовые файлы и то, что ещё нужно проверить, лежат в этой таблице.",
    },
    {
      target: "help",
      placement: "bottom",
      title: "Справка",
      body: "Если запутались, нажмите знак вопроса. Подсказка никуда не денется.",
    },
  ];
}

export function quickStartForRole(role) {
  return tourForRole(role).map((step) => step.body);
}

export const helpSections = [
  {
    title: "Как сделать выгрузку",
    body: "На экране «Выгрузки» выберите бланк закупки или Север. Загрузите файлы, дождитесь обработки и скачайте результат.",
  },
  {
    title: "Какие файлы нужны",
    body: "Для бланка закупки нужна таблица продаж из 1С и бланк поставщика. Для Севера загрузите бланки городов и таблицу Тюмени.",
  },
  {
    title: 'Что значит "Нужно проверить"',
    body: "Это строки, где сервис не уверен в количестве. Откройте их, поправьте при необходимости и продолжите.",
  },
  {
    title: "Пользователи и доступ",
    body: "Новый сотрудник входит по одноразовой ссылке. Если человек потерял пароль, сбросьте доступ и отправьте новую ссылку.",
  },
  {
    title: "Если не получается войти",
    body: "Проверьте логин и пароль. Если ссылка устарела или доступ закрыли, попросите владельца или администратора компании прислать новое приглашение.",
  },
];
