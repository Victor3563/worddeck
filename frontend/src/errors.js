
const FIELD_LABELS = {
  email: 'Email',
  display_name: 'Имя',
  password: 'Пароль',
  current_password: 'Текущий пароль',
  new_password: 'Новый пароль',
  name: 'Название',
  title: 'Название',
  description: 'Описание',
  ru_word: 'Слово на русском',
  en_word: 'Слово на английском',
  ru_example: 'Пример на русском',
  en_example: 'Пример на английском',
}

const label = (field) => FIELD_LABELS[field] || field

const BY_FIELD = {
  'invalid:email': () => 'Некорректный email',
  'taken:email': () => 'Пользователь с таким email уже зарегистрирован',
  'too_short:password': (p) => `Пароль должен быть не короче ${p.min} символов`,
  'too_short:new_password': (p) => `Пароль должен быть не короче ${p.min} символов`,
  'too_long:password': () => 'Пароль слишком длинный',
  'too_long:new_password': () => 'Пароль слишком длинный',
  'wrong_password:current_password': () => 'Текущий пароль указан неверно',
  'wrong_password:password': () => 'Пароль указан неверно',
  'invalid:color': () => 'Неизвестный цвет',
  'invalid:mode': () => 'Неизвестный режим обучения',
  'invalid:resource_type': () => 'Можно поделиться областью, папкой или карточкой',
  'invalid:total_cards': () => 'Некорректные значения статистики',
  'required:target_id': () => 'Выберите, куда скопировать',
  'required:ids': () => 'Не выбраны карточки',
  'required:cards': (p) => (p.number ? null : 'Добавьте хотя бы одну карточку'),
  'too_large:file': (p) => `Файл больше ${Math.round(p.max_bytes / 1024)} КБ`,
  'bad_request:file': () => 'Ожидается файл или текст с карточками',
}

const BY_CODE = {
  network: () => 'Нет связи с сервером. Проверьте подключение.',
  bad_request: () => 'Некорректный запрос',
  too_large: () => 'Слишком большой запрос',
  not_found: () => 'Не найдено',
  unauthenticated: () => 'Требуется вход в аккаунт',
  invalid_credentials: () => 'Неверный email или пароль',
  rate_limited: () => 'Слишком много попыток. Подождите минуту и попробуйте снова.',
  internal: () => 'Внутренняя ошибка сервера',
  required: (p, f) => `Поле «${label(f)}» обязательно`,
  too_short: (p, f) => `Поле «${label(f)}» короче ${p.min} символов`,
  too_long: (p, f) => `Поле «${label(f)}» длиннее ${p.max} символов`,
  invalid: (p, f) => `Некорректное значение поля «${label(f)}»`,
  taken: (p, f) => `Значение поля «${label(f)}» уже занято`,
  wrong_password: () => 'Пароль указан неверно',
  too_many: (p) => `Слишком много карточек: максимум ${p.max} за раз`,
  empty_file: () => 'Файл пуст или не содержит карточек',
  not_utf8: () => 'Файл должен быть в кодировке UTF-8',
  invalid_json: () => 'Некорректный JSON',
  no_valid_cards: () => 'В файле нет корректных карточек',
  bad_columns: (p) => `Ожидалось 2 колонки (слово; перевод) или 4 (слово; пример; перевод; пример), найдено ${p.found}`,
  bad_line: () => 'Не удалось разобрать строку',
}

export function errorText({ code, field, params } = {}, status = 0) {
  const p = params || {}
  if (field === 'cards' && p.number) {
    return `Карточка №${p.number}: ${errorText({ code, field: p.item_field, params: p })}`
  }
  const specific = BY_FIELD[`${code}:${field}`]
  const text = (specific && specific(p, field)) || (BY_CODE[code] && BY_CODE[code](p, field))
  return text || (status ? `Ошибка ${status}` : 'Неизвестная ошибка')
}
