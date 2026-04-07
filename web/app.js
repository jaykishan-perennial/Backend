const API = '';
let token = localStorage.getItem('admin_token') || '';

// ── HTTP helper ──

async function api(method, path, body) {
    const opts = { method, headers: { 'Content-Type': 'application/json' } };
    if (token) opts.headers['Authorization'] = `Bearer ${token}`;
    if (body) opts.body = JSON.stringify(body);
    const res = await fetch(API + path, opts);
    const data = await res.json();
    if (!res.ok) throw new Error(data.message || `HTTP ${res.status}`);
    return data;
}

// ── Toast ──

function toast(msg, type = 'info') {
    const el = document.getElementById('toast');
    el.textContent = msg;
    el.className = `toast show ${type}`;
    setTimeout(() => { el.className = 'toast hidden'; }, 3000);
}

// ── Modal ──

function openModal(title, bodyHTML, footerHTML) {
    document.getElementById('modal-title').textContent = title;
    document.getElementById('modal-body').innerHTML = bodyHTML;
    document.getElementById('modal-footer').innerHTML = footerHTML || '';
    document.getElementById('modal-overlay').classList.remove('hidden');
}

function closeModal() {
    document.getElementById('modal-overlay').classList.add('hidden');
}

document.getElementById('modal-close').addEventListener('click', closeModal);
document.getElementById('modal-overlay').addEventListener('click', e => {
    if (e.target === e.currentTarget) closeModal();
});

// ── Auth ──

function checkAuth() {
    if (token) {
        document.getElementById('login-screen').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');
        loadDashboard();
    } else {
        document.getElementById('login-screen').classList.remove('hidden');
        document.getElementById('app').classList.add('hidden');
    }
}

document.getElementById('login-form').addEventListener('submit', async e => {
    e.preventDefault();
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;
    const errEl = document.getElementById('login-error');
    errEl.classList.add('hidden');

    try {
        const data = await api('POST', '/api/admin/login', { email, password });
        token = data.data.token;
        localStorage.setItem('admin_token', token);
        checkAuth();
    } catch (err) {
        errEl.textContent = err.message;
        errEl.classList.remove('hidden');
    }
});

document.getElementById('logout-btn').addEventListener('click', () => {
    token = '';
    localStorage.removeItem('admin_token');
    checkAuth();
});

// ── Navigation ──

document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', e => {
        e.preventDefault();
        const page = item.dataset.page;
        document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
        item.classList.add('active');
        document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
        document.getElementById(`page-${page}`).classList.add('active');

        if (page === 'dashboard') loadDashboard();
        else if (page === 'customers') loadCustomers(1);
        else if (page === 'packs') loadPacks(1);
        else if (page === 'subscriptions') loadSubscriptions(1);
        else if (page === 'audit') loadAuditLogs(1);
    });
});

// ── Helpers ──

function formatDate(d) {
    if (!d || d === '0001-01-01T00:00:00Z') return '—';
    return new Date(d).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
}

function statusBadge(s) {
    return `<span class="badge badge-${s}">${s}</span>`;
}

function renderPagination(containerId, page, limit, total, loadFn) {
    const pages = Math.ceil(total / limit) || 1;
    const el = document.getElementById(containerId);
    let html = `<button ${page <= 1 ? 'disabled' : ''} onclick="${loadFn}(${page - 1})">&laquo;</button>`;
    for (let i = 1; i <= pages; i++) {
        html += `<button class="${i === page ? 'active' : ''}" onclick="${loadFn}(${i})">${i}</button>`;
    }
    html += `<button ${page >= pages ? 'disabled' : ''} onclick="${loadFn}(${page + 1})">&raquo;</button>`;
    html += `<span class="page-info">${total} total</span>`;
    el.innerHTML = html;
}

// ── Dashboard ──

async function loadDashboard() {
    try {
        const data = await api('GET', '/api/v1/admin/dashboard');
        const d = data.data;
        document.getElementById('stat-customers').textContent = d.total_customers;
        document.getElementById('stat-active').textContent = d.active_subscriptions;
        document.getElementById('stat-pending').textContent = d.pending_requests;
        document.getElementById('stat-revenue').textContent = `$${Number(d.total_revenue).toFixed(2)}`;

        const tbody = document.getElementById('activities-table');
        if (!d.recent_activities || d.recent_activities.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No recent activities</td></tr>';
            return;
        }
        tbody.innerHTML = d.recent_activities.map(a => `
            <tr>
                <td>${a.id}</td>
                <td>${a.customer_name || '—'}</td>
                <td>${a.pack_name || '—'}</td>
                <td>${statusBadge(a.status)}</td>
                <td>${formatDate(a.date)}</td>
            </tr>`).join('');
    } catch (err) {
        if (err.message.includes('401') || err.message.includes('expired')) {
            token = '';
            localStorage.removeItem('admin_token');
            checkAuth();
        }
        toast(err.message, 'error');
    }
}

// ── Customers ──

async function loadCustomers(page = 1) {
    const search = document.getElementById('customer-search').value;
    try {
        const data = await api('GET', `/api/v1/admin/customers?page=${page}&limit=10&search=${encodeURIComponent(search)}`);
        const customers = data.data || [];
        const tbody = document.getElementById('customers-table');

        if (customers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No customers found</td></tr>';
            document.getElementById('customers-pagination').innerHTML = '';
            return;
        }

        tbody.innerHTML = customers.map(c => `
            <tr>
                <td>${c.id}</td>
                <td>${c.name}</td>
                <td>${c.email}</td>
                <td>${c.phone || '—'}</td>
                <td>${formatDate(c.created_at)}</td>
                <td class="actions">
                    <button class="btn btn-sm btn-primary" onclick="editCustomer(${c.id})">Edit</button>
                    <button class="btn btn-sm btn-success" onclick="assignSubscriptionModal(${c.id}, '${c.name}')">Assign Sub</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteCustomer(${c.id}, '${c.name}')">Delete</button>
                </td>
            </tr>`).join('');

        const pg = data.pagination;
        renderPagination('customers-pagination', pg.page, pg.limit, pg.total, 'loadCustomers');
    } catch (err) { toast(err.message, 'error'); }
}
window.loadCustomers = loadCustomers;

let customerSearchTimer;
document.getElementById('customer-search').addEventListener('input', () => {
    clearTimeout(customerSearchTimer);
    customerSearchTimer = setTimeout(() => loadCustomers(1), 400);
});

window.editCustomer = async function (id) {
    try {
        const data = await api('GET', `/api/v1/admin/customers/${id}`);
        const c = data.data;
        openModal('Edit Customer', `
            <div class="form-group"><label>Name</label><input id="m-cust-name" value="${c.name}"></div>
            <div class="form-group"><label>Phone</label><input id="m-cust-phone" value="${c.phone || ''}"></div>
        `, `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
            <button class="btn btn-primary" onclick="updateCustomer(${id})">Save</button>`);
    } catch (err) { toast(err.message, 'error'); }
};

window.updateCustomer = async function (id) {
    const name = document.getElementById('m-cust-name').value;
    const phone = document.getElementById('m-cust-phone').value;
    try {
        await api('PUT', `/api/v1/admin/customers/${id}`, { name, phone });
        closeModal();
        toast('Customer updated', 'success');
        loadCustomers(1);
    } catch (err) { toast(err.message, 'error'); }
};

window.deleteCustomer = function (id, name) {
    openModal('Delete Customer', `<p>Are you sure you want to delete <strong>${name}</strong>?</p>`,
        `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
         <button class="btn btn-danger" onclick="confirmDeleteCustomer(${id})">Delete</button>`);
};

window.confirmDeleteCustomer = async function (id) {
    try {
        await api('DELETE', `/api/v1/admin/customers/${id}`);
        closeModal();
        toast('Customer deleted', 'success');
        loadCustomers(1);
    } catch (err) { toast(err.message, 'error'); }
};

// ── Subscription Packs ──

async function loadPacks(page = 1) {
    try {
        const data = await api('GET', `/api/v1/admin/subscription-packs?page=${page}&limit=10`);
        const packs = data.data || [];
        const tbody = document.getElementById('packs-table');

        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No packs found</td></tr>';
            document.getElementById('packs-pagination').innerHTML = '';
            return;
        }

        tbody.innerHTML = packs.map(p => {
            const desc = (p.description || '').replace(/'/g, "\\'");
            return `<tr>
                <td>${p.id}</td>
                <td>${p.name}</td>
                <td>${p.description || '—'}</td>
                <td><code>${p.sku}</code></td>
                <td>$${Number(p.price).toFixed(2)}</td>
                <td>${p.validity_months} month${p.validity_months > 1 ? 's' : ''}</td>
                <td class="actions">
                    <button class="btn btn-sm btn-primary" onclick="editPack(${p.id}, '${p.name}', '${desc}', '${p.sku}', ${p.price}, ${p.validity_months})">Edit</button>
                    <button class="btn btn-sm btn-danger" onclick="deletePack(${p.id}, '${p.name}')">Delete</button>
                </td>
            </tr>`;
        }).join('');

        const pg = data.pagination;
        renderPagination('packs-pagination', pg.page, pg.limit, pg.total, 'loadPacks');
    } catch (err) { toast(err.message, 'error'); }
}
window.loadPacks = loadPacks;

document.getElementById('add-pack-btn').addEventListener('click', () => {
    openModal('Add Subscription Pack', `
        <div class="form-group"><label>Name</label><input id="m-pack-name" placeholder="Premium Plan"></div>
        <div class="form-group"><label>Description</label><textarea id="m-pack-desc" rows="2" placeholder="Full access to all features"></textarea></div>
        <div class="form-group"><label>SKU</label><input id="m-pack-sku" placeholder="premium-plan"></div>
        <div class="form-group"><label>Price ($)</label><input id="m-pack-price" type="number" step="0.01" min="0" placeholder="29.99"></div>
        <div class="form-group"><label>Validity (months)</label><input id="m-pack-validity" type="number" min="1" max="12" placeholder="12"></div>
    `, `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
        <button class="btn btn-primary" onclick="createPack()">Create</button>`);
});

window.createPack = async function () {
    const name = document.getElementById('m-pack-name').value.trim();
    const description = document.getElementById('m-pack-desc').value.trim();
    const sku = document.getElementById('m-pack-sku').value.trim();
    const price = parseFloat(document.getElementById('m-pack-price').value);
    const validity_months = parseInt(document.getElementById('m-pack-validity').value);

    if (!name || name.length < 2) return toast('Pack name must be at least 2 characters', 'error');
    if (!sku || sku.length < 2) return toast('SKU must be at least 2 characters', 'error');
    if (!price || price <= 0) return toast('Price must be greater than 0', 'error');
    if (!validity_months || validity_months < 1 || validity_months > 12) return toast('Validity must be 1-12 months', 'error');

    try {
        await api('POST', '/api/v1/admin/subscription-packs', { name, description, sku, price, validity_months });
        closeModal();
        toast('Pack created', 'success');
        loadPacks(1);
    } catch (err) { toast(err.message, 'error'); }
};

window.editPack = function (id, name, desc, sku, price, validity) {
    openModal('Edit Subscription Pack', `
        <div class="form-group"><label>Name</label><input id="m-pack-name" value="${name}"></div>
        <div class="form-group"><label>Description</label><textarea id="m-pack-desc" rows="2">${desc}</textarea></div>
        <div class="form-group"><label>SKU</label><input id="m-pack-sku" value="${sku}"></div>
        <div class="form-group"><label>Price ($)</label><input id="m-pack-price" type="number" step="0.01" value="${price}"></div>
        <div class="form-group"><label>Validity (months)</label><input id="m-pack-validity" type="number" min="1" max="12" value="${validity}"></div>
    `, `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
        <button class="btn btn-primary" onclick="updatePack(${id})">Save</button>`);
};

window.updatePack = async function (id) {
    const name = document.getElementById('m-pack-name').value;
    const description = document.getElementById('m-pack-desc').value;
    const sku = document.getElementById('m-pack-sku').value;
    const price = parseFloat(document.getElementById('m-pack-price').value);
    const validity_months = parseInt(document.getElementById('m-pack-validity').value);
    try {
        await api('PUT', `/api/v1/admin/subscription-packs/${id}`, { name, description, sku, price, validity_months });
        closeModal();
        toast('Pack updated', 'success');
        loadPacks(1);
    } catch (err) { toast(err.message, 'error'); }
};

window.deletePack = function (id, name) {
    openModal('Delete Pack', `<p>Are you sure you want to delete <strong>${name}</strong>?</p>`,
        `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
         <button class="btn btn-danger" onclick="confirmDeletePack(${id})">Delete</button>`);
};

window.confirmDeletePack = async function (id) {
    try {
        await api('DELETE', `/api/v1/admin/subscription-packs/${id}`);
        closeModal();
        toast('Pack deleted', 'success');
        loadPacks(1);
    } catch (err) { toast(err.message, 'error'); }
};

// ── Subscriptions ──

async function loadSubscriptions(page = 1) {
    const status = document.getElementById('sub-status-filter').value;
    const statusParam = status ? `&status=${status}` : '';
    try {
        const data = await api('GET', `/api/v1/admin/subscriptions?page=${page}&limit=10${statusParam}`);
        const subs = data.data || [];
        const tbody = document.getElementById('subscriptions-table');

        if (subs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No subscriptions found</td></tr>';
            document.getElementById('subscriptions-pagination').innerHTML = '';
            return;
        }

        tbody.innerHTML = subs.map(s => {
            let actions = '';
            if (s.status === 'requested') {
                actions = `<button class="btn btn-sm btn-success" onclick="approveSubscription(${s.id})">Approve</button>`;
            }
            if (s.status === 'active' || s.status === 'approved') {
                actions += `<button class="btn btn-sm btn-danger" onclick="unassignSubscription(${s.customer_id}, ${s.id})">Unassign</button>`;
            }
            return `<tr>
                <td>${s.id}</td>
                <td>${s.customer?.name || '—'}</td>
                <td>${s.pack?.name || '—'}</td>
                <td>${statusBadge(s.status)}</td>
                <td>${formatDate(s.assigned_at)}</td>
                <td>${formatDate(s.expires_at)}</td>
                <td class="actions">${actions || '—'}</td>
            </tr>`;
        }).join('');

        const pg = data.pagination;
        renderPagination('subscriptions-pagination', pg.page, pg.limit, pg.total, 'loadSubscriptions');
    } catch (err) { toast(err.message, 'error'); }
}
window.loadSubscriptions = loadSubscriptions;

document.getElementById('sub-status-filter').addEventListener('change', () => loadSubscriptions(1));

window.approveSubscription = async function (id) {
    try {
        await api('POST', `/api/v1/admin/subscriptions/${id}/approve`);
        toast('Subscription approved', 'success');
        loadSubscriptions(1);
        loadDashboard();
    } catch (err) { toast(err.message, 'error'); }
};

window.unassignSubscription = async function (customerId, subId) {
    openModal('Unassign Subscription', `<p>Are you sure you want to unassign subscription #${subId}?</p>`,
        `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
         <button class="btn btn-danger" onclick="confirmUnassign(${customerId}, ${subId})">Unassign</button>`);
};

window.confirmUnassign = async function (customerId, subId) {
    try {
        await api('DELETE', `/api/v1/admin/customers/${customerId}/subscription/${subId}`);
        closeModal();
        toast('Subscription unassigned', 'success');
        loadSubscriptions(1);
        loadDashboard();
    } catch (err) { toast(err.message, 'error'); }
};

window.assignSubscriptionModal = async function (customerId, customerName) {
    try {
        const data = await api('GET', '/api/v1/admin/subscription-packs?page=1&limit=100');
        const packs = data.data || [];
        if (packs.length === 0) return toast('No subscription packs available', 'error');

        const options = packs.map(p => `<option value="${p.id}">${p.name} — $${Number(p.price).toFixed(2)} (${p.validity_months}mo)</option>`).join('');
        openModal(`Assign Subscription to ${customerName}`, `
            <div class="form-group"><label>Subscription Pack</label><select id="m-assign-pack">${options}</select></div>
        `, `<button class="btn btn-outline" onclick="closeModal()">Cancel</button>
            <button class="btn btn-success" onclick="confirmAssign(${customerId})">Assign</button>`);
    } catch (err) { toast(err.message, 'error'); }
};

window.confirmAssign = async function (customerId) {
    const packId = parseInt(document.getElementById('m-assign-pack').value);
    try {
        await api('POST', `/api/v1/admin/customers/${customerId}/assign-subscription`, { pack_id: packId });
        closeModal();
        toast('Subscription assigned', 'success');
        loadSubscriptions(1);
        loadDashboard();
    } catch (err) { toast(err.message, 'error'); }
};

// ── Audit Logs ──

async function loadAuditLogs(page = 1) {
    try {
        const data = await api('GET', `/api/v1/admin/audit-logs?page=${page}&limit=20`);
        const logs = data.data || [];
        const tbody = document.getElementById('audit-table');

        if (logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No audit logs found</td></tr>';
            document.getElementById('audit-pagination').innerHTML = '';
            return;
        }

        tbody.innerHTML = logs.map(l => `
            <tr>
                <td>${l.id}</td>
                <td>${statusBadge(l.action)}</td>
                <td>${l.entity}</td>
                <td>${l.entity_id || '—'}</td>
                <td>${l.details || '—'}</td>
                <td>${l.ip_address || '—'}</td>
                <td>${formatDate(l.created_at)}</td>
            </tr>`).join('');

        const pg = data.pagination;
        renderPagination('audit-pagination', pg.page, pg.limit, pg.total, 'loadAuditLogs');
    } catch (err) { toast(err.message, 'error'); }
}
window.loadAuditLogs = loadAuditLogs;

// ── Init ──
checkAuth();
