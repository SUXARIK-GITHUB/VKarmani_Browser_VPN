const NATIVE_HOST = 'com.vkarmani.browser';
const RECONNECT_ALARM = 'vkarmani-reconnect';
const REFRESH_ALARM = 'vkarmani-refresh';

let nativePort = null;
let requestSeq = 0;
const pending = new Map();
let lastLocalPort = 0;

function nextId() {
  requestSeq += 1;
  return `${Date.now()}-${requestSeq}`;
}

function chromeCall(target, methodName, ...args) {
  return new Promise((resolve, reject) => {
    const method = target?.[methodName];
    if (typeof method !== 'function') {
      reject(new Error(`Chrome API method is not available: ${methodName}`));
      return;
    }
    method.call(target, ...args, (value) => {
      const err = chrome.runtime.lastError;
      if (err) reject(new Error(err.message));
      else resolve(value);
    });
  });
}

async function oneShot(action, payload = {}) {
  const request = { id: nextId(), action, ...payload };
  const response = await chromeCall(chrome.runtime, 'sendNativeMessage', NATIVE_HOST, request);
  if (!response?.ok) throw new Error(response?.error || 'Native helper error');
  return response.status;
}

function closeNativePort() {
  if (nativePort) {
    try { nativePort.disconnect(); } catch (_) {}
  }
  nativePort = null;
}

function ensureNativePort() {
  if (nativePort) return nativePort;
  const port = chrome.runtime.connectNative(NATIVE_HOST);
  nativePort = port;

  port.onMessage.addListener((response) => {
    const waiter = pending.get(response?.id);
    if (!waiter) return;
    pending.delete(response.id);
    if (response.ok) waiter.resolve(response.status);
    else waiter.reject(new Error(response.error || 'Native helper error'));
  });

  port.onDisconnect.addListener(async () => {
    const message = chrome.runtime.lastError?.message || 'Native helper disconnected';
    nativePort = null;
    for (const [, waiter] of pending) waiter.reject(new Error(message));
    pending.clear();
    const { desiredConnected } = await chrome.storage.local.get({ desiredConnected: false });
    if (desiredConnected) {
      await setBadge('!', '#b91c1c');
      // Fail-closed: browser proxy is intentionally NOT cleared here.
      chrome.alarms.create(RECONNECT_ALARM, { delayInMinutes: 0.5 });
    } else {
      await setBadge('', '#000000');
    }
  });

  return port;
}

function portRequest(action, payload = {}) {
  return new Promise((resolve, reject) => {
    const port = ensureNativePort();
    const id = nextId();
    pending.set(id, { resolve, reject });
    try {
      port.postMessage({ id, action, ...payload });
    } catch (err) {
      pending.delete(id);
      reject(err);
    }
  });
}

async function setBadge(text, color) {
  await chrome.action.setBadgeBackgroundColor({ color });
  await chrome.action.setBadgeText({ text });
}

async function setPrivacyProtection() {
  try {
    await chromeCall(chrome.privacy.network.webRTCIPHandlingPolicy, 'set', { value: 'disable_non_proxied_udp', scope: 'regular' });
  } catch (_) {}
  try {
    await chromeCall(chrome.privacy.network.networkPredictionEnabled, 'set', { value: false, scope: 'regular' });
  } catch (_) {}
}

async function clearPrivacyProtection() {
  try { await chromeCall(chrome.privacy.network.webRTCIPHandlingPolicy, 'clear', { scope: 'regular' }); } catch (_) {}
  try { await chromeCall(chrome.privacy.network.networkPredictionEnabled, 'clear', { scope: 'regular' }); } catch (_) {}
}

async function setBrowserProxy(localPort) {
  const current = await chromeCall(chrome.proxy.settings, 'get', { incognito: false });
  if (!['controllable_by_this_extension', 'controlled_by_this_extension'].includes(current.levelOfControl)) {
    throw new Error(`Proxy setting is ${current.levelOfControl}; another policy/extension controls it`);
  }
  const config = {
    mode: 'fixed_servers',
    rules: {
      singleProxy: { scheme: 'http', host: '127.0.0.1', port: localPort },
      bypassList: ['localhost', '127.0.0.1', '[::1]']
    }
  };
  await chromeCall(chrome.proxy.settings, 'set', { value: config, scope: 'regular' });
  await setPrivacyProtection();
  lastLocalPort = localPort;
}

async function clearBrowserProxy() {
  await chromeCall(chrome.proxy.settings, 'clear', { scope: 'regular' });
  await clearPrivacyProtection();
  lastLocalPort = 0;
}

async function connectServer(serverId) {
  const status = await portRequest('connect', { server_id: serverId });
  if (!status?.connected || !status.local_proxy_port) throw new Error('Helper did not start local proxy');
  try {
    await setBrowserProxy(status.local_proxy_port);
  } catch (err) {
    try { await portRequest('disconnect'); } catch (_) {}
    throw err;
  }
  await chrome.storage.local.set({ desiredConnected: true, selectedId: status.selected_id || serverId });
  await setBadge('ON', '#15803d');
  return status;
}

async function connectAuto() {
  const status = await portRequest('connect_auto');
  if (!status?.connected || !status.local_proxy_port) throw new Error('Helper did not start local proxy');
  try {
    await setBrowserProxy(status.local_proxy_port);
  } catch (err) {
    try { await portRequest('disconnect'); } catch (_) {}
    throw err;
  }
  await chrome.storage.local.set({ desiredConnected: true, selectedId: status.selected_id || '' });
  await setBadge('ON', '#15803d');
  return status;
}

async function disconnect() {
  // User explicitly requested disconnect: restore browser first, then stop tunnel.
  await clearBrowserProxy();
  await chrome.storage.local.set({ desiredConnected: false });
  if (nativePort) {
    try { await portRequest('disconnect'); } catch (_) {}
    closeNativePort();
  } else {
    try { await oneShot('disconnect'); } catch (_) {}
  }
  await setBadge('', '#000000');
}

async function getStatus() {
  let status;
  if (nativePort) status = await portRequest('status');
  else status = await oneShot('status');
  const browser = await chrome.storage.local.get({ desiredConnected: false, selectedId: '' });
  return { ...status, desired_connected: browser.desiredConnected, browser_selected_id: browser.selectedId, browser_proxy_port: lastLocalPort };
}

async function restoreConnection() {
  const { desiredConnected, selectedId } = await chrome.storage.local.get({ desiredConnected: false, selectedId: '' });
  if (!desiredConnected) return;
  try {
    let status;
    if (selectedId) {
      try { status = await connectServer(selectedId); }
      catch (_) { status = await connectAuto(); }
    } else {
      status = await connectAuto();
    }
    if (status?.connected) await setBadge('ON', '#15803d');
  } catch (_) {
    // Fail closed. If Chrome persisted our proxy, keep it. If this is a fresh browser start
    // and no proxy is currently set, point to a dead loopback endpoint until recovery/user disconnect.
    try {
      const current = await chromeCall(chrome.proxy.settings, 'get', { incognito: false });
      if (current.levelOfControl === 'controllable_by_this_extension') {
        await setBrowserProxy(lastLocalPort || 9);
      }
    } catch (_) {}
    await setBadge('!', '#b91c1c');
  }
}

chrome.runtime.onInstalled.addListener(async () => {
  const current = await chrome.storage.local.get({ desiredConnected: false });
  await chrome.storage.local.set({ desiredConnected: current.desiredConnected || false });
  chrome.alarms.create(REFRESH_ALARM, { periodInMinutes: 60 });
});

chrome.runtime.onStartup.addListener(() => {
  chrome.alarms.create(REFRESH_ALARM, { periodInMinutes: 60 });
  restoreConnection();
});

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === RECONNECT_ALARM) {
    await restoreConnection();
    return;
  }
  if (alarm.name === REFRESH_ALARM) {
    try {
      const status = nativePort ? await portRequest('status') : await oneShot('status');
      if (status?.subscription_set) {
        if (nativePort) await portRequest('refresh');
        else await oneShot('refresh');
      }
    } catch (_) {}
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  // Only extension pages use this API; reject messages originating from ordinary web pages.
  if (sender.id && sender.id !== chrome.runtime.id) {
    sendResponse({ ok: false, error: 'Invalid sender' });
    return false;
  }
  (async () => {
    switch (message?.action) {
      case 'get_state': return { ok: true, status: await getStatus() };
      case 'set_subscription': return { ok: true, status: await oneShot('set_subscription', { subscription_url: message.subscription_url }) };
      case 'refresh': return { ok: true, status: nativePort ? await portRequest('refresh') : await oneShot('refresh') };
      case 'probe_servers': return { ok: true, status: nativePort ? await portRequest('probe_servers') : await oneShot('probe_servers') };
      case 'connect': return { ok: true, status: await connectServer(message.server_id) };
      case 'connect_auto': return { ok: true, status: await connectAuto() };
      case 'disconnect': await disconnect(); return { ok: true, status: await getStatus() };
      case 'clear_subscription':
        await disconnect();
        return { ok: true, status: await oneShot('clear_subscription') };
      default: return { ok: false, error: 'Unknown action' };
    }
  })().then(sendResponse).catch((err) => sendResponse({ ok: false, error: err?.message || String(err) }));
  return true;
});

restoreConnection();
