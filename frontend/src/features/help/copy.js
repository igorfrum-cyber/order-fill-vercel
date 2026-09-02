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
    return "Вы можете создавать компании, помогать с доступом и смотреть обзор сервиса.";
  }
  if (role === "company_owner") {
    return "Вы управляете доступом сотрудников и видите выгрузки своей компании.";
  }
  if (role === "company_admin") {
    return "Вы приглашаете сотрудников, правите данные компании, сбрасываете доступ и видите выгрузки своей компании.";
  }
  return "Вы создаёте выгрузки, проверяете строки и скачиваете готовые файлы.";
}

export function profileCompanyLabel(me) {
  if (!me?.company_id || me.role === "platform_admin") return "Сервис";
  return me.company_name || "Ваша компания";
}

export function headerContext(me) {
  if (me?.role === "platform_admin") {
    return { companyLine: "Сервис", roleLine: roleLabel(me.role) };
  }
  return {
    companyLine: me?.company_name ? `Компания: ${me.company_name}` : "Компания",
    roleLine: roleLabel(me?.role),
  };
}

export function profileFields(me) {
  return [
    { label: "Логин", value: me.login, editable: false },
    { label: "Компания", value: profileCompanyLabel(me), editable: false },
    { label: "Доступ", value: roleLabel(me.role), editable: false },
  ];
}

export const accountPasswordHint =
  "Пароль только ваш. На работе удобнее входить по Face ID, Touch ID или Windows Hello.";

export const logoutEverywhereLabel = "Выйти со всех устройств";

export const logoutEverywhereConfirm =
  "Вы выйдете здесь и на других устройствах. Войти снова можно будет по логину и паролю.";

export const twoFactorEnableLabel = "Включить вход по коду";

export const twoFactorDisableLabel = "Отключить вход по коду";

export const twoFactorSetupHint =
  "На работе хватит Face ID или Touch ID. Код нужен, только если входите с чужого компьютера.";

export const twoFactorRecoveryHint = "Сохраните запасные коды: каждый работает один раз.";

export const twoFactorSetupSteps = [
  "Откройте Яндекс Ключ, Google Authenticator или 1Password.",
  "На телефоне нажмите «Добавить в приложение» — секрет подставится сам. На компьютере наведите камеру на квадрат.",
  "Введите шесть цифр из приложения.",
];

export const twoFactorOpenAppLabel = "Добавить в приложение";

export const twoFactorLoginTitle = "Подтвердите вход";

export const twoFactorCodeLabel = "Код из приложения";

export const twoFactorRecoveryLabel = "Использовать запасной код";

export const twoFactorRecoveryCodeLabel = "Запасной код";

export const twoFactorManualKeyLabel = "Ключ для приложения";

export const twoFactorRequiredHint =
  "Добавьте Face ID, Touch ID или Windows Hello. На работе код из приложения не понадобится.";

export const securitySetupLabel = "Настроить";

export const passkeyLoginTitle = "Быстрый вход";

export const passkeyLoginHint = "Face ID, Touch ID, Windows Hello или 1Password";

export const passkeyLoginButton = "Войти по Face ID или Touch ID";

export const passkeySettingsTitle = "Быстрый вход";

export const passkeySettingsHint =
  "На работе удобнее входить по Face ID, Touch ID или Windows Hello. Один раз добавьте это устройство — код из приложения не понадобится.";

export const passkeyAddButton = "Добавить Face ID или Touch ID";

export const passkeyDeleteButton = "Удалить";

export const passkeyInsecureOriginHint =
  "Face ID на этом адресе недоступен. Откройте сайт по обычному домену с https — или войдите паролем.";

export const sessionsTitle = "Где вы вошли";

export const sessionsHint = "Этот компьютер отмечен. Если видите чужой вход — закройте его.";

export const sessionCurrentLabel = "Этот компьютер";

export const sessionRevokeLabel = "Выйти";

export const inviteRoleHint =
  "Выберите, что человек сможет делать в компании. Доступ можно отключить позже.";

export function tourForRole(role) {
  if (role === "platform_admin") {
    return [
      {
        target: "overview",
        placement: "bottom",
        title: "Обзор",
        body: "Сначала смотрите, отвечают ли сервисы и кто менял доступ.",
      },
      {
        target: "companies",
        placement: "bottom",
        title: "Компании",
        body: "Сначала создайте компанию и выберите её в списке.",
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
        body: "На обзоре видно, что сейчас считается. Новую выгрузку создаёт закупщик или администратор компании.",
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
        target: "company",
        placement: "bottom",
        title: "Компания",
        body: "Здесь название и латинский адрес входа. По этой ссылке входят сотрудники.",
      },
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
    title: "Статусы выгрузок",
    body: "Пока файлы считаются, статус будет «В очереди» или «Обработка». «На проверке» — откройте выгрузку, «Готово» — скачайте файлы.",
  },
  {
    title: "Пользователи и доступ",
    body: "Новый сотрудник входит по одноразовой ссылке. Если человек потерял пароль, сбросьте доступ и отправьте новую ссылку.",
  },
  {
    title: "Обзор сервиса",
    body: "Администратор сервиса на «Обзоре» видит, отвечают ли сервисы, какие выгрузки в работе и кто менял доступ.",
  },
  {
    title: "Если не получается войти",
    body: "Проверьте логин и пароль или войдите по Face ID, Touch ID или Windows Hello. Если ссылка устарела или доступ закрыли, попросите владельца или администратора компании прислать новое приглашение.",
  },
  {
    title: "Как быстрее входить",
    body: "В профиле добавьте Face ID, Touch ID или Windows Hello — на работе код не понадобится. Если нужен запасной вход, откройте код из приложения и нажмите «Добавить в приложение».",
  },
];
