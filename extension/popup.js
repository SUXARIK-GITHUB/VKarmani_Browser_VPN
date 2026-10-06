const $ = (id) => document.getElementById(id);
let current = null;
let busy = false;

const FLAG_BASE = 'flags';
const ISO_CODES = `AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW`.split(' ');
const EXTRA_ALIASES = new Map(Object.entries({
  holland: 'NL', nederland: 'NL',
  england: 'GB', uk: 'GB', britain: 'GB', 'great britain': 'GB',
  usa: 'US', america: 'US',
  uae: 'AE',
  korea: 'KR', 'south korea': 'KR',
  czechia: 'CZ',
  russia: 'RU',
  россия: 'RU', германия: 'DE', нидерланды: 'NL', голландия: 'NL', эстония: 'EE', финляндия: 'FI',
  швеция: 'SE', польша: 'PL', франция: 'FR', великобритания: 'GB', англия: 'GB', сша: 'US',
  канада: 'CA', япония: 'JP', сингапур: 'SG', гонконг: 'HK', турция: 'TR', испания: 'ES', италия: 'IT',
  норвегия: 'NO', дания: 'DK', латвия: 'LV', литва: 'LT', чехия: 'CZ', австрия: 'AT', швейцария: 'CH',
  румыния: 'RO', болгария: 'BG', сербия: 'RS', молдова: 'MD', украина: 'UA', казахстан: 'KZ',
  китай: 'CN', тайвань: 'TW', индия: 'IN', австралия: 'AU', бразилия: 'BR', мексика: 'MX'
}));
const CITY_HINTS = new Map(Object.entries({
  tallinn: 'EE', таллин: 'EE', helsinki: 'FI', хельсинки: 'FI', stockholm: 'SE', стокгольм: 'SE',
  amsterdam: 'NL', амстердам: 'NL', frankfurt: 'DE', франкфурт: 'DE', berlin: 'DE', берлин: 'DE',
  warsaw: 'PL', варшава: 'PL', paris: 'FR', париж: 'FR', london: 'GB', лондон: 'GB',
  riga: 'LV', рига: 'LV', vilnius: 'LT', вильнюс: 'LT', prague: 'CZ', прага: 'CZ',
  vienna: 'AT', вена: 'AT', zurich: 'CH', цюрих: 'CH', bucharest: 'RO', бухарест: 'RO',
  sofia: 'BG', софия: 'BG', belgrade: 'RS', белград: 'RS', istanbul: 'TR', стамбул: 'TR',
  madrid: 'ES', мадрид: 'ES', milan: 'IT', милан: 'IT', oslo: 'NO', осло: 'NO',
  copenhagen: 'DK', копенгаген: 'DK', tokyo: 'JP', токио: 'JP', singapore: 'SG', сингапур: 'SG',
  'hong kong': 'HK', гонконг: 'HK', dubai: 'AE', дубай: 'AE', toronto: 'CA', торонто: 'CA',
  'new york': 'US', 'los angeles': 'US', miami: 'US', chicago: 'US', sydney: 'AU', сидней: 'AU'
}));

const regionNamesRu = typeof Intl.DisplayNames === 'function' ? new Intl.DisplayNames(['ru'], { type: 'region' }) : null;
const regionNamesEn = typeof Intl.DisplayNames === 'function' ? new Intl.DisplayNames(['en'], { type: 'region' }) : null;
const COUNTRY_LOOKUP = buildCountryLookup();

function normalize(value) {
  return String(value || '')
    .normalize('NFKD')
    .replace(/\p{M}+/gu, '')
    .toLocaleLowerCase('ru-RU')
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim();
}

function buildCountryLookup() {
  const rows = [];
  for (const code of ISO_CODES) {
    const names = new Set();
    try { names.add(regionNamesEn?.of(code)); } catch (_) {}
    try { names.add(regionNamesRu?.of(code)); } catch (_) {}
    for (const name of names) {
      const key = normalize(name);
      if (key && key.length > 1) rows.push([key, code]);
    }
  }
  for (const [alias, code] of EXTRA_ALIASES) rows.push([normalize(alias), code]);
  for (const [alias, code] of CITY_HINTS) rows.push([normalize(alias), code]);
  rows.sort((a, b) => b[0].length - a[0].length);
  return rows;
}

function countryCodeFromFlagEmoji(value) {
  const chars = Array.from(String(value || ''));
  for (let i = 0; i < chars.length - 1; i += 1) {
    const a = chars[i].codePointAt(0);
    const b = chars[i + 1].codePointAt(0);
    if (a >= 0x1F1E6 && a <= 0x1F1FF && b >= 0x1F1E6 && b <= 0x1F1FF) {
      return String.fromCharCode(65 + a - 0x1F1E6, 65 + b - 0x1F1E6);
    }
  }
  return '';
}

function countryCodeFromIsoToken(value) {
  const raw = String(value || '');
  const matches = raw.match(/(?:^|[^A-Za-z])([A-Z]{2})(?=$|[^A-Za-z])/g) || [];
  for (const match of matches) {
    const codeMatch = match.match(/([A-Z]{2})/);
    const code = codeMatch?.[1] || '';
    if (ISO_CODES.includes(code)) return code;
  }
  return '';
}

function countryMeta(server) {
  const raw = `${server?.name || ''}`;
  let code = countryCodeFromFlagEmoji(raw) || countryCodeFromIsoToken(raw);
  const normalized = ` ${normalize(raw)} `;

  if (!code) {
    for (const [alias, aliasCode] of COUNTRY_LOOKUP) {
      if (normalized.includes(` ${alias} `)) {
        code = aliasCode;
        break;
      }
    }
  }

  if (!code) return { code: '', label: 'Локация VKarmani', flagUrl: '' };
  let label = code;
  try { label = regionNamesRu?.of(code) || regionNamesEn?.of(code) || code; } catch (_) {}
  return {
    code,
    label,
    flagUrl: `${FLAG_BASE}/${code.toLowerCase()}.png`
  };
}

function cleanServerName(value) {
  return String(value || 'VKarmani server')
    .replace(/[\u{1F1E6}-\u{1F1FF}]{2}/gu, '')
    .replace(/^[\s·•|—–\-:]+/u, '')
    .trim() || 'VKarmani server';
}

function makeFlag(meta, className = 'server-flag') {
  const shell = document.createElement('div');
  shell.className = `${className} flag-shell flag-placeholder`;
  const fallback = document.createElement('span');
  fallback.textContent = meta.code || 'VK';
  shell.appendChild(fallback);

  if (meta.flagUrl) {
    const img = document.createElement('img');
    img.alt = '';
    img.loading = 'lazy';
    img.referrerPolicy = 'no-referrer';
    img.addEventListener('load', () => shell.classList.remove('flag-placeholder'));
    img.addEventListener('error', () => img.remove(), { once: true });
    img.src = meta.flagUrl;
    shell.prepend(img);
  }
  return shell;
}

function setHeroFlag(server) {
  const target = $('heroFlag');
  target.textContent = '';
  const meta = server ? countryMeta(server) : { code: '', flagUrl: '' };
  const flag = makeFlag(meta, 'hero-flag');
  for (const child of Array.from(flag.childNodes)) target.appendChild(child);
  target.className = flag.className;
}

function send(action, payload = {}) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage({ action, ...payload }, (response) => {
      const err = chrome.runtime.lastError;
      if (err) return reject(new Error(err.message));
      if (!response?.ok) return reject(new Error(response?.error || 'Unknown extension error'));
      resolve(response.status);
    });
  });
}

function setBusy(value) {
  busy = value;
  document.querySelectorAll('button').forEach((button) => { button.disabled = value; });
}

function showError(message) {
  $('errorBox').textContent = message || '';
  $('errorBox').classList.toggle('hidden', !message);
}

function showWarning(message) {
  $('warningBox').textContent = message || '';
  $('warningBox').classList.toggle('hidden', !message);
}

function sourceLabel(value) {
  const source = String(value || '');
  if (!source) return '';
  if (source === 'direct') return 'прямое обновление';
  if (source.startsWith('doh-')) return 'резервный DNS';
  if (source.startsWith('direct-ip-')) return 'резервный маршрут';
  if (source === 'active-vpn') return 'через VPN';
  if (source.startsWith('public-relay:')) return 'аварийный relay';
  return 'обновлено';
}

function fmtTime(value) {
  if (!value) return 'Ещё не обновлялась';
  try {
    return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value));
  } catch (_) {
    return value;
  }
}

function latencyClass(server) {
  if (!server?.reachable) return 'bad';
  const ms = Number(server.latency_ms || 0);
  if (!ms || ms < 80) return 'good';
  if (ms < 180) return 'mid';
  return 'bad';
}

function latencyText(server) {
  if (!server?.reachable) return 'Недоступен';
  const ms = Number(server.latency_ms || 0);
  return ms > 0 ? `${ms} мс` : 'Доступен';
}

function setStatusChip(mode, text) {
  $('statePill').className = `status-chip status-${mode}`;
  $('statePill').innerHTML = '';
  const dot = document.createElement('span');
  dot.className = 'status-dot';
  $('statePill').append(dot, document.createTextNode(text));
}

function renderConnection(status, selectedServer) {
  const connectingProblem = status?.desired_connected && !status?.connected;
  if (connectingProblem) {
    setStatusChip('error', 'Защита');
    $('connectionTitle').textContent = 'Требуется восстановление';
    $('connectionSubtitle').textContent = 'Прямой трафик не включён. Расширение пытается восстановить защищённое соединение.';
    setHeroFlag(selectedServer);
  } else if (status?.connected) {
    setStatusChip('on', 'Подключено');
    $('connectionTitle').textContent = cleanServerName(selectedServer?.name || 'VKarmani server');
    const meta = selectedServer ? countryMeta(selectedServer) : null;
    $('connectionSubtitle').textContent = meta?.code ? `${meta.label} · браузер защищён` : 'Браузер защищён';
    setHeroFlag(selectedServer);
  } else {
    setStatusChip('off', 'Отключено');
    $('connectionTitle').textContent = 'Не подключено';
    $('connectionSubtitle').textContent = 'Выберите сервер или подключитесь автоматически.';
    setHeroFlag(null);
  }
  $('autoBtn').classList.toggle('hidden', !!status?.connected);
  $('disconnectBtn').classList.toggle('hidden', !status?.connected);
}

function renderServerList(status, selected) {
  const list = $('serverList');
  list.textContent = '';
  const servers = status.servers || [];
  if (!servers.length) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    empty.textContent = 'Список серверов пока пуст. Обновите подписку.';
    list.appendChild(empty);
    return;
  }

  for (const server of servers) {
    const active = !!status.connected && server.id === selected;
    const meta = countryMeta(server);
    const row = document.createElement('button');
    row.type = 'button';
    row.className = `server-row${active ? ' active' : ''}`;
    row.disabled = busy || active;
    row.setAttribute('aria-label', active ? `${server.name}, активен` : `Подключиться к ${server.name}`);

    row.appendChild(makeFlag(meta));

    const copy = document.createElement('div');
    copy.className = 'server-copy';
    const name = document.createElement('div');
    name.className = 'server-name';
    name.textContent = cleanServerName(server.name);
    const country = document.createElement('div');
    country.className = 'server-country';
    country.textContent = meta.label;
    copy.append(name, country);

    const trailing = document.createElement('div');
    trailing.className = 'server-trailing';
    if (active) {
      const activeLabel = document.createElement('span');
      activeLabel.className = 'active-label';
      activeLabel.textContent = 'Активен';
      trailing.appendChild(activeLabel);
    } else {
      const latency = document.createElement('span');
      latency.className = `latency ${latencyClass(server)}`;
      latency.textContent = latencyText(server);
      const arrow = document.createElement('span');
      arrow.className = 'server-arrow';
      arrow.textContent = '›';
      trailing.append(latency, arrow);
    }

    row.append(copy, trailing);
    row.addEventListener('click', () => act(() => send('connect', { server_id: server.id })));
    list.appendChild(row);
  }
}

function render(status) {
  current = status;
  showError('');
  const hasSub = !!status?.subscription_set;
  $('setupCard').classList.toggle('hidden', hasSub);
  $('mainCard').classList.toggle('hidden', !hasSub);

  if (status?.desired_connected && !status?.connected) {
    showWarning('VPN должен быть подключён, но helper сейчас недоступен. Прокси оставлен в fail-closed режиме, прямой трафик не включён.');
  } else if (status?.last_error && status?.cache_available) {
    showWarning(`Не удалось обновить подписку. Используется последний рабочий список. ${status.last_error}`);
  } else if (status?.last_error) {
    showWarning(status.last_error);
  } else {
    showWarning('');
  }

  if (!hasSub) {
    setStatusChip('off', 'Не настроено');
    return;
  }

  $('maskedSubscription').textContent = status.subscription_masked || 'Сохранена';
  $('lastUpdate').textContent = `Обновлено: ${fmtTime(status.last_success)}${status.last_source ? ` · ${sourceLabel(status.last_source)}` : ''}`;
  $('helperInfo').textContent = `Helper ${status.helper_version || '?'} · sing-box: ${status.sing_box_found ? (status.sing_box_version || 'найден') : 'не найден'}`;

  const selected = status.selected_id || status.browser_selected_id;
  const selectedServer = (status.servers || []).find((server) => server.id === selected);
  renderConnection(status, selectedServer);
  renderServerList(status, selected);
}

async function load() {
  try {
    render(await send('get_state'));
  } catch (err) {
    showError(`Локальный модуль VKarmani не установлен или недоступен: ${err.message}. Запустите 00_INSTALL_VKARMANI.cmd из этой же папки один раз, затем перезапустите браузер.`);
  }
}

async function act(fn) {
  if (busy) return false;
  setBusy(true);
  showError('');
  try {
    const status = await fn();
    setBusy(false);
    render(status);
    return true;
  } catch (err) {
    setBusy(false);
    showError(err.message || String(err));
    return false;
  }
}

$('saveSubscription').addEventListener('click', async () => {
  const url = $('subscriptionInput').value.trim();
  if (!url) return showError('Вставьте ссылку подписки.');
  const ok = await act(() => send('set_subscription', { subscription_url: url }));
  if (ok) $('subscriptionInput').value = '';
});
$('subscriptionInput').addEventListener('keydown', (event) => {
  if (event.key === 'Enter') $('saveSubscription').click();
});
$('refreshBtn').addEventListener('click', () => act(() => send('refresh')));
$('probeBtn').addEventListener('click', () => act(() => send('probe_servers')));
$('autoBtn').addEventListener('click', () => act(() => send('connect_auto')));
$('disconnectBtn').addEventListener('click', () => act(() => send('disconnect')));
$('changeSubscriptionBtn').addEventListener('click', () => {
  $('setupCard').classList.remove('hidden');
  $('mainCard').classList.add('hidden');
  $('subscriptionInput').focus();
});
$('clearSubscriptionBtn').addEventListener('click', () => {
  if (confirm('Удалить сохранённую подписку и список серверов?')) act(() => send('clear_subscription'));
});

load();
