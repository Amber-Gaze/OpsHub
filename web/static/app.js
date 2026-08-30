/* OpsHub 运控配置台 前端逻辑 */
'use strict';

// ---------- 全局状态 ----------
let token = localStorage.getItem('opshub_token') || '';
let username = localStorage.getItem('opshub_user') || '';
let treeData = [];                 // /configs/tree 的缓存
let currentBusiness = '';
let currentModule = '';

// ---------- DOM ----------
const $ = (id) => document.getElementById(id);
const loginView = $('login-view'), appView = $('app-view');
const loginForm = $('login-form'), loginError = $('login-error');
const treeEl = $('tree'), treeStatus = $('tree-status');
const moduleView = $('module-view'), welcome = $('welcome');
const moduleTitle = $('module-title'), itemsBody = $('items-body'), moduleEmpty = $('module-empty');
const scopeBadge = $('scope-badge'), userName = $('user-name');
const editModal = $('edit-modal'), addModal = $('add-modal'), historyModal = $('history-modal');
const toast = $('toast');

// ---------- 通用 ----------
async function api(method, path, body) {
  const headers = {};
  if (token) headers['Authorization'] = 'Bearer ' + token;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (res.status === 401) {
    logout();
    throw new Error('登录已过期，请重新登录');
  }
  let data = null;
  if (res.status !== 204) {
    const text = await res.text();
    try { data = text ? JSON.parse(text) : null; } catch { data = null; }
  }
  if (!res.ok) {
    const msg = (data && (data.error || data.message)) || ('HTTP ' + res.status);
    throw new Error(msg);
  }
  return data;
}

let toastTimer = null;
function showToast(msg, type = 'ok') {
  toast.textContent = msg;
  toast.className = 'toast ' + type;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.add('hidden'), 2600);
}

function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
}
function fmtTime(t) {
  return t ? String(t).replace('T', ' ').slice(0, 19) + ' UTC' : '-';
}

// ---------- 登录 ----------
loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const user = $('login-user').value.trim();
  const pass = $('login-pass').value;
  loginError.textContent = '';
  try {
    const data = await fetch('/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user, password: pass }),
    }).then(async (r) => {
      const d = await r.json().catch(() => ({}));
      if (!r.ok) throw new Error(d.error || 'HTTP ' + r.status);
      return d;
    });
    token = data.token;
    username = user;
    localStorage.setItem('opshub_token', token);
    localStorage.setItem('opshub_user', username);
    enterApp();
  } catch (err) {
    loginError.textContent = '登录失败：' + err.message;
  }
});

// ---------- 注册 ----------
$('goto-signup').addEventListener('click', () => {
  $('login-form').classList.add('hidden');
  $('signup-form').classList.remove('hidden');
  $('signup-error').textContent = '';
});
$('goto-login').addEventListener('click', () => {
  $('signup-form').classList.add('hidden');
  $('login-form').classList.remove('hidden');
  $('login-error').textContent = '';
});

$('signup-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const body = {
    user: $('signup-user').value.trim(),
    password: $('signup-pass').value,
    email: $('signup-email').value.trim(),
    phone: $('signup-phone').value.trim(),
  };
  $('signup-error').textContent = '';
  try {
    const res = await fetch('/signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const d = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(d.error || 'HTTP ' + res.status);
    showToast('注册成功，请登录');
    $('login-user').value = body.user;
    $('login-pass').value = '';
    $('goto-login').click();
  } catch (err) {
    $('signup-error').textContent = '注册失败：' + err.message;
  }
});

function logout() {
  token = '';
  username = '';
  localStorage.removeItem('opshub_token');
  localStorage.removeItem('opshub_user');
  appView.classList.add('hidden');
  loginView.classList.remove('hidden');
}

$('logout-btn').addEventListener('click', logout);

// ---------- 进入应用 ----------
async function enterApp() {
  loginView.classList.add('hidden');
  appView.classList.remove('hidden');
  userName.textContent = username || '';
  await Promise.all([loadScope(), loadTree()]);
}

// 我的权限（scope）
async function loadScope() {
  scopeBadge.textContent = '';
  try {
    const data = await api('POST', '/scope', { token });
    const grants = data.scope || [];
    const text = grants.map((g) => g.obj + ':' + g.act).join('，');
    scopeBadge.textContent = text ? '权限 ' + text : '无配置权限';
    scopeBadge.title = text;
  } catch { /* 忽略 */ }
}

// ---------- 配置树 ----------
async function loadTree() {
  treeStatus.textContent = '加载中…';
  try {
    treeData = await api('GET', '/configs/tree');
    renderTree();
    treeStatus.textContent = '';
  } catch (err) {
    treeStatus.textContent = '加载失败：' + err.message;
  }
}

function renderTree() {
  treeEl.innerHTML = '';
  if (!Array.isArray(treeData) || treeData.length === 0) {
    treeEl.innerHTML = '<div class="status">暂无可见配置（或没有权限）</div>';
    return;
  }
  for (const biz of treeData) {
    const bnode = document.createElement('div');
    bnode.className = 'tree-node';

    const head = document.createElement('div');
    head.className = 'tree-business';
    head.innerHTML = `<span class="arrow">▶</span><span>${esc(biz.business)}</span>`;
    head.addEventListener('click', () => {
      head.classList.toggle('open');
    });

    const mods = document.createElement('div');
    mods.className = 'tree-modules';
    for (const mod of biz.modules || []) {
      const m = document.createElement('div');
      m.className = 'tree-module' + (currentBusiness === biz.business && currentModule === mod.module ? ' active' : '');
      m.textContent = mod.module === '' ? '(业务级配置)' : mod.module;
      m.title = mod.items.map((i) => i.key).join('\n');
      m.addEventListener('click', () => selectModule(biz.business, mod.module, mod.items));
      mods.appendChild(m);
    }

    bnode.appendChild(head);
    bnode.appendChild(mods);
    treeEl.appendChild(bnode);
    if (currentBusiness === biz.business) head.classList.add('open');
  }
}

function selectModule(business, module, items) {
  currentBusiness = business;
  currentModule = module;
  welcome.classList.add('hidden');
  moduleView.classList.remove('hidden');
  moduleTitle.textContent = `${business} / ${module === '' ? '(业务级配置)' : module}`;
  itemsBody.innerHTML = '';
  if (!items || items.length === 0) {
    moduleEmpty.classList.remove('hidden');
  } else {
    moduleEmpty.classList.add('hidden');
    for (const it of items) {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td class="mono">${esc(it.key)}</td>
        <td class="val">${esc(it.value)}</td>
        <td>v${it.version}</td>
        <td>${esc(it.updated_by)}</td>
        <td>${fmtTime(it.updated_at)}</td>
        <td class="actions">
          <button class="btn ghost small" data-act="edit">编辑</button>
          <button class="btn ghost small" data-act="history">历史</button>
          <button class="btn danger small" data-act="delete">删除</button>
        </td>`;
      const onEdit = () => openEdit(it);
      const onHistory = () => openHistory(it);
      const onDelete = async () => {
        if (!confirm(`确认删除配置 ${it.key} ？`)) return;
        try {
          await api('DELETE', `/configs/tree/${business}/${module}/${lastSeg(it.key)}`);
          showToast('已删除');
          await loadTree();
          // 重新选中当前模块
          selectModuleByName(business, module);
        } catch (err) { showToast(err.message, 'err'); }
      };
      tr.querySelector('[data-act="edit"]').addEventListener('click', onEdit);
      tr.querySelector('[data-act="history"]').addEventListener('click', onHistory);
      tr.querySelector('[data-act="delete"]').addEventListener('click', onDelete);
      itemsBody.appendChild(tr);
    }
  }
  renderTree(); // 刷新高亮
}

function selectModuleByName(business, module) {
  const biz = treeData.find((b) => b.business === business);
  if (!biz) return;
  const mod = (biz.modules || []).find((m) => m.module === module);
  if (mod) selectModule(business, module, mod.items);
}

function lastSeg(key) {
  const p = String(key).split('/');
  return p[p.length - 1];
}

// ---------- 编辑 ----------
let editingItem = null;
function openEdit(it) {
  editingItem = it;
  $('edit-key').textContent = it.key;
  $('edit-value').value = it.value;
  $('edit-error').textContent = '';
  editModal.classList.remove('hidden');
}
$('edit-save').addEventListener('click', async () => {
  const value = $('edit-value').value;
  try {
    await api('PUT', `/configs/tree/${currentBusiness}/${currentModule}/${lastSeg(editingItem.key)}`,
      { value, operator: username });
    showToast('已保存');
    closeModal(editModal);
    await loadTree();
    selectModuleByName(currentBusiness, currentModule);
  } catch (err) { $('edit-error').textContent = '保存失败：' + err.message; }
});

// ---------- 新增 ----------
$('add-btn').addEventListener('click', () => {
  $('add-scope').textContent = `${currentBusiness} / ${currentModule}`;
  $('add-name').value = '';
  $('add-value').value = '';
  $('add-error').textContent = '';
  addModal.classList.remove('hidden');
});
$('add-save').addEventListener('click', async () => {
  const name = $('add-name').value.trim();
  const value = $('add-value').value;
  if (!name) { $('add-error').textContent = '请填写配置项名称'; return; }
  const key = currentModule === '' ? `${currentBusiness}/${name}` : `${currentBusiness}/${currentModule}/${name}`;
  try {
    await api('POST', '/configs', { key, value, operator: username });
    showToast('已创建 ' + key);
    closeModal(addModal);
    await loadTree();
    selectModuleByName(currentBusiness, currentModule);
  } catch (err) { $('add-error').textContent = '创建失败：' + err.message; }
});

// ---------- 历史 ----------
async function openHistory(it) {
  $('history-key').textContent = it.key;
  $('history-body').innerHTML = '';
  $('history-current').textContent = '加载中…';
  $('history-diff').classList.add('hidden');
  historyModal.classList.remove('hidden');
  try {
    const data = await api('GET', '/configs/history/' + it.key);
    const cur = data.current;
    $('history-current').textContent = cur
      ? `key:      ${cur.key}\nvalue:    ${cur.value}\nversion:  v${cur.version}\nupdated_by: ${cur.updated_by}\nupdated_at: ${fmtTime(cur.updated_at)}`
      : '(该配置已删除)';
    const hist = data.history || [];
    for (let i = 0; i < hist.length; i++) {
      const h = hist[i];
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td>${i + 1}</td>
        <td><span class="badge" style="color:${h.action === 'delete' ? 'var(--danger)' : h.action === 'create' ? 'var(--ok)' : 'var(--warn)'}">${esc(h.action)}</span></td>
        <td>v${h.version}</td>
        <td>${esc(h.operator)}</td>
        <td>${fmtTime(h.created_at)}</td>
        <td class="actions"><button class="btn ghost small" data-diff="1">差异</button></td>`;
      tr.querySelector('[data-diff]').addEventListener('click', () => showHistoryDiff(h));
      $('history-body').appendChild(tr);
    }
    if (hist.length === 0) {
      $('history-body').innerHTML = '<tr><td colspan="6" class="empty">暂无变更记录</td></tr>';
    }
  } catch (err) {
    $('history-current').textContent = '加载失败：' + err.message;
  }
}

// 差异对比：后端只提供 before/after，diff 由前端公共逻辑完成。
// 若后续希望更强能力，可换成 jsdiff 等前端 diff 库（改 renderDiff 一处即可）。
function showHistoryDiff(h) {
  $('history-diff-title').textContent = `差异对比（${h.action} · v${h.version}）`;
  $('history-diff-view').innerHTML = renderDiff(h.before, h.after);
  $('history-diff').classList.remove('hidden');
}

// 简单的行级 LCS diff：返回带标记的行（' ' 相同 / '-' 删除 / '+' 新增）
function diffLines(a, b) {
  const A = String(a == null ? '' : a).split('\n');
  const B = String(b == null ? '' : b).split('\n');
  const n = A.length, m = B.length;
  const dp = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = A[i] === B[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const out = [];
  let i = 0, j = 0;
  while (i < n && j < m) {
    if (A[i] === B[j]) { out.push({ t: ' ', text: A[i] }); i++; j++; }
    else if (dp[i + 1][j] >= dp[i][j + 1]) { out.push({ t: '-', text: A[i] }); i++; }
    else { out.push({ t: '+', text: B[j] }); j++; }
  }
  while (i < n) { out.push({ t: '-', text: A[i] }); i++; }
  while (j < m) { out.push({ t: '+', text: B[j] }); j++; }
  return out;
}

function renderDiff(before, after) {
  return diffLines(before, after).map((l) => {
    const cls = l.t === '+' ? 'diff-add' : l.t === '-' ? 'diff-del' : '';
    const sign = l.t === '+' ? '+' : l.t === '-' ? '-' : ' ';
    return `<div class="diff-line ${cls}"><span class="diff-sign">${sign}</span><span>${esc(l.text) || ' '}</span></div>`;
  }).join('');
}

// ---------- Tab 切换 ----------
const tabViews = {
  config: $('tab-config'),
  users: $('tab-users'),
  policies: $('tab-policies'),
};
document.querySelectorAll('#main-tabs .tab').forEach((btn) => {
  btn.addEventListener('click', async () => {
    document.querySelectorAll('#main-tabs .tab').forEach((b) => b.classList.remove('active'));
    btn.classList.add('active');
    Object.entries(tabViews).forEach(([k, el]) => el.classList.toggle('hidden', k !== btn.dataset.tab));
    if (btn.dataset.tab === 'users') await loadUsers();
    if (btn.dataset.tab === 'policies') await loadPolicies();
  });
});

// ---------- 用户管理 ----------
async function loadUsers() {
  const status = $('users-status');
  status.textContent = '加载中…';
  try {
    const list = await api('GET', '/users');
    const body = $('users-body');
    body.innerHTML = '';
    $('users-empty').classList.add('hidden');
    if (!Array.isArray(list) || list.length === 0) {
      $('users-empty').classList.remove('hidden');
      status.textContent = '';
      return;
    }
    for (const u of list) {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td class="mono">${esc(u.username)}</td>
        <td>${esc(u.email)}</td>
        <td>${esc(u.phone)}</td>
        <td>${u.is_admin ? '✓' : ''}</td>
        <td>${u.status === 1 ? '启用' : '禁用'}</td>
        <td>${u.logined_at ? new Date(u.logined_at * 1000).toLocaleString() : '-'}</td>
        <td class="actions">
          <button class="btn ghost small" data-u="grant">权限</button>
          <button class="btn ghost small" data-u="edit">编辑</button>
          <button class="btn ghost small" data-u="passwd">改密</button>
          <button class="btn danger small" data-u="delete">删除</button>
        </td>`;
      tr.querySelector('[data-u="grant"]').addEventListener('click', () => openUserGrants(u));
      tr.querySelector('[data-u="edit"]').addEventListener('click', () => openUserEdit(u));
      tr.querySelector('[data-u="passwd"]').addEventListener('click', () => openUserPasswd(u));
      tr.querySelector('[data-u="delete"]').addEventListener('click', () => deleteUser(u));
      body.appendChild(tr);
    }
    status.textContent = '';
  } catch (err) {
    status.textContent = '加载失败：' + err.message;
  }
}

$('user-add-btn').addEventListener('click', () => {
  ['uadd-name', 'uadd-pass', 'uadd-email', 'uadd-phone'].forEach((id) => { $(id).value = ''; });
  $('uadd-error').textContent = '';
  $('user-add-modal').classList.remove('hidden');
});
$('uadd-save').addEventListener('click', async () => {
  const body = {
    user: $('uadd-name').value.trim(),
    password: $('uadd-pass').value,
    email: $('uadd-email').value.trim(),
    phone: $('uadd-phone').value.trim(),
  };
  try {
    await api('POST', '/signup', body);
    showToast('已创建用户 ' + body.user);
    closeModal($('user-add-modal'));
    await loadUsers();
  } catch (err) { $('uadd-error').textContent = '创建失败：' + err.message; }
});

let editingUser = null;
function openUserEdit(u) {
  editingUser = u;
  $('uedit-name').textContent = u.username;
  $('uedit-email').value = u.email || '';
  $('uedit-phone').value = u.phone || '';
  $('uedit-admin').checked = !!u.is_admin;
  $('uedit-status').value = u.status;
  $('uedit-error').textContent = '';
  $('user-edit-modal').classList.remove('hidden');
}
$('uedit-save').addEventListener('click', async () => {
  const body = {
    email: $('uedit-email').value.trim(),
    phone: $('uedit-phone').value.trim(),
    is_admin: $('uedit-admin').checked,
    status: parseInt($('uedit-status').value, 10),
  };
  try {
    await api('PUT', '/users/' + encodeURIComponent(editingUser.username), body);
    showToast('已保存');
    closeModal($('user-edit-modal'));
    await loadUsers();
  } catch (err) { $('uedit-error').textContent = '保存失败：' + err.message; }
});

function openUserPasswd(u) {
  editingUser = u;
  $('upasswd-name').textContent = u.username;
  $('upasswd-old').value = '';
  $('upasswd-new').value = '';
  $('upasswd-error').textContent = '';
  $('user-passwd-modal').classList.remove('hidden');
}
$('upasswd-save').addEventListener('click', async () => {
  const body = { old_password: $('upasswd-old').value, new_password: $('upasswd-new').value };
  try {
    await api('PUT', '/users/' + encodeURIComponent(editingUser.username) + '/change-passwd', body);
    showToast('密码已更新');
    closeModal($('user-passwd-modal'));
  } catch (err) { $('upasswd-error').textContent = '修改失败：' + err.message; }
});

async function deleteUser(u) {
  if (!confirm('确认删除用户 ' + u.username + ' ？')) return;
  try {
    await api('DELETE', '/users/' + encodeURIComponent(u.username));
    showToast('已删除');
    await loadUsers();
  } catch (err) { showToast(err.message, 'err'); }
}

async function openUserGrants(u) {
  $('ugrants-name').textContent = u.username;
  $('ugrants-body').innerHTML = '<tr><td colspan="2" class="empty">加载中…</td></tr>';
  $('user-grants-modal').classList.remove('hidden');
  try {
    const data = await api('GET', '/users/' + encodeURIComponent(u.username) + '/grants');
    const grants = data.grants || [];
    const body = $('ugrants-body');
    body.innerHTML = '';
    if (grants.length === 0) {
      body.innerHTML = '<tr><td colspan="2" class="empty">无配置授权（管理员默认可访问全部）</td></tr>';
    }
    for (const g of grants) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td class="mono">${esc(g.obj)}</td><td>${esc(g.act)}</td>`;
      body.appendChild(tr);
    }
  } catch (err) {
    $('ugrants-body').innerHTML = `<tr><td colspan="2" class="empty">加载失败：${esc(err.message)}</td></tr>`;
  }
}

// ---------- 权限管理 ----------
async function loadPolicies() {
  const status = $('policies-status');
  status.textContent = '加载中…';
  try {
    const data = await api('GET', '/policies');
    const pol = data.policies || [];
    const grp = data.groupings || [];
    const pb = $('policies-body');
    pb.innerHTML = '';
    if (pol.length === 0) {
      pb.innerHTML = '<tr><td colspan="3" class="empty">暂无策略</td></tr>';
    }
    for (const p of pol) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td class="mono">${esc(p[0])}</td><td class="mono">${esc(p[1])}</td><td>${esc(p[2])}</td>`;
      pb.appendChild(tr);
    }
    const gb = $('groupings-body');
    gb.innerHTML = '';
    if (grp.length === 0) {
      gb.innerHTML = '<tr><td colspan="2" class="empty">暂无角色绑定</td></tr>';
    }
    for (const g of grp) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td class="mono">${esc(g[0])}</td><td class="mono">${esc(g[1])}</td>`;
      gb.appendChild(tr);
    }
    status.textContent = '';
  } catch (err) {
    status.textContent = '加载失败：' + err.message;
  }
}

function grantPayload() {
  return {
    sub: $('grant-sub').value.trim(),
    business: $('grant-business').value.trim(),
    module: $('grant-module').value.trim(),
    item: $('grant-item').value.trim(),
    act: $('grant-act').value,
  };
}
function clearGrantForm() { $('grant-module').value = ''; $('grant-item').value = ''; }

$('grant-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  try {
    const r = await api('POST', '/policies/config-grant', grantPayload());
    showToast(`已授权 ${r.obj} : ${r.act}`);
    clearGrantForm();
    await loadPolicies();
  } catch (err) { showToast('授权失败：' + err.message, 'err'); }
});
$('grant-revoke-btn').addEventListener('click', async () => {
  try {
    await api('POST', '/policies/config-revoke', grantPayload());
    showToast('已撤销');
    clearGrantForm();
    await loadPolicies();
  } catch (err) { showToast('撤销失败：' + err.message, 'err'); }
});

$('role-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const body = { user: $('role-user').value.trim(), role: $('role-name').value.trim() };
  try {
    await api('POST', '/policies/roles', body);
    showToast('已绑定');
    $('role-user').value = '';
    $('role-name').value = '';
    await loadPolicies();
  } catch (err) { showToast('绑定失败：' + err.message, 'err'); }
});
$('role-unbind-btn').addEventListener('click', async () => {
  const body = { user: $('role-user').value.trim(), role: $('role-name').value.trim() };
  try {
    await api('POST', '/policies/roles/delete', body);
    showToast('已解绑');
    $('role-user').value = '';
    $('role-name').value = '';
    await loadPolicies();
  } catch (err) { showToast('解绑失败：' + err.message, 'err'); }
});

// ---------- 弹窗关闭 ----------
function closeModal(modal) { modal.classList.add('hidden'); }
document.querySelectorAll('[data-close]').forEach((btn) => {
  btn.addEventListener('click', () => closeModal($(btn.dataset.close)));
});
document.querySelectorAll('.modal-mask').forEach((mask) => {
  mask.addEventListener('click', (e) => { if (e.target === mask) mask.classList.add('hidden'); });
});

// ---------- 初始化 ----------
if (token) {
  enterApp();
} else {
  loginView.classList.remove('hidden');
}
