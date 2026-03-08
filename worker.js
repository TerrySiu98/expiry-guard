// Cloudflare Workers - Expiry Guard Pro
// (Classic Edition - Feishu Webhook, TG Bot Commands, Cost Tracking, Multi-Currency)

// ===== R2 Storage =====
class R2Storage {
    constructor(bucket) { this.bucket = bucket; }
    
    async getJSON(key) { 
        const obj = await this.bucket?.get(key); 
        return obj ? JSON.parse(await obj.text()) : null; 
    }
    
    async putJSON(key, data) { 
        await this.bucket?.put(key, JSON.stringify(data)); 
    }
    
    async delete(key) { 
        await this.bucket?.delete(key); 
    }
    
    async getUsers() { return (await this.getJSON('users.json')) || []; }
    async saveUsers(users) { await this.putJSON('users.json', users); }
    
    async getItems(userId) { return (await this.getJSON(`items/${userId}.json`)) || []; }
    async saveItems(userId, items) { await this.putJSON(`items/${userId}.json`, items); }
    
    async getSettings() { return (await this.getJSON('config/settings.json')) || {}; }
    async saveSettings(settings) { await this.putJSON('config/settings.json', settings); }
    
    async getSession(sid) { return await this.getJSON(`sessions/${sid}.json`); }
    async saveSession(sid, data) { await this.putJSON(`sessions/${sid}.json`, data); }
}

// ===== Utilities & Constants =====
function getCookie(request, name) { 
    const match = (request.headers.get('Cookie') || '').match(new RegExp(`${name}=([^;]+)`)); 
    return match ? match[1] : null; 
}

function setCookie(name, value, maxAge = 2592000) { 
    return `${name}=${value}; Path=/; HttpOnly; SameSite=Strict; Max-Age=${maxAge}`; 
}

function generateSessionId() { 
    return Array.from(crypto.getRandomValues(new Uint8Array(32))).map(b => b.toString(16).padStart(2, '0')).join(''); 
}

async function hashPassword(password) { 
    const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(password)); 
    return Array.from(new Uint8Array(hash)).map(b => b.toString(16).padStart(2, '0')).join(''); 
}

const CommonTimezones = [
    { Val: "Asia/Shanghai", LabelKey: "TZ_Shanghai" }, 
    { Val: "Asia/Tokyo", LabelKey: "TZ_Tokyo" },
    { Val: "Asia/Seoul", LabelKey: "TZ_Seoul" }, 
    { Val: "Asia/Singapore", LabelKey: "TZ_Singapore" },
    { Val: "Europe/London", LabelKey: "TZ_London" }, 
    { Val: "Europe/Berlin", LabelKey: "TZ_Berlin" },
    { Val: "America/New_York", LabelKey: "TZ_NY" }, 
    { Val: "America/Los_Angeles", LabelKey: "TZ_LA" }, 
    { Val: "UTC", LabelKey: "TZ_UTC" }
];

// 内置静态汇率表 (以 CNY 为基准 1)
const Currencies = {
    "CNY": { Symbol: "¥", Rate: 1, Name: "人民币 (CNY)" },
    "USD": { Symbol: "$", Rate: 7.25, Name: "美元 (USD)" },
    "EUR": { Symbol: "€", Rate: 7.85, Name: "欧元 (EUR)" },
    "GBP": { Symbol: "£", Rate: 9.15, Name: "英镑 (GBP)" },
    "JPY": { Symbol: "¥", Rate: 0.048, Name: "日元 (JPY)" }
};

async function getAuthUser(request, storage) {
    const sid = getCookie(request, 'session'); 
    if (!sid) return null;
    const session = await storage.getSession(sid); 
    if (!session) return null;
    const users = await storage.getUsers(); 
    return users.find(u => u.Username === session.username);
}

// ===== 2FA Functions =====
function generate2FACode() {
    return Math.floor(100000 + Math.random() * 900000).toString();
}

async function save2FACode(storage, username, code) {
    await storage.putJSON(`2fa/${username}.json`, { code, timestamp: Date.now() });
}

async function verify2FACode(storage, username, code) {
    const data = await storage.getJSON(`2fa/${username}.json`);
    if (!data) return false;
    
    if (Date.now() - data.timestamp > 300000) {
        await storage.delete(`2fa/${username}.json`);
        return false;
    }
    
    if (data.code === code) {
        await storage.delete(`2fa/${username}.json`);
        return true;
    }
    return false;
}

// ===== Notifications =====
async function sendTelegramNotification(botToken, chatId, message) {
    if (!botToken || !chatId) return false;
    try {
        const res = await fetch(`https://api.telegram.org/bot${botToken}/sendMessage`, { 
            method: 'POST', 
            headers: { 'Content-Type': 'application/json' }, 
            body: JSON.stringify({ 
                chat_id: chatId, 
                text: message.replace(/<br>/g, '\n'), 
                parse_mode: 'HTML', 
                disable_web_page_preview: true 
            }) 
        });
        return res.ok;
    } catch (e) { return false; }
}

async function sendFeishuNotification(webhookUrl, title, message) {
    if (!webhookUrl) return false;
    try {
        const text = `🔔 ${title}\n\n${message.replace(/<br>/g, '\n').replace(/<\/?b>/g, '').replace(/<\/?i>/g, '')}`;
        const res = await fetch(webhookUrl, { 
            method: 'POST', 
            headers: { 'Content-Type': 'application/json' }, 
            body: JSON.stringify({ msg_type: "text", content: { text: text } }) 
        });
        return res.ok;
    } catch (e) { return false; }
}

// ===== HTML Template =====
const HTML_TEMPLATE = `<!DOCTYPE html>
<html lang="{{.User.Language}}">
<head>
    <meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ExpiryGuard Pro</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <script>tailwind.config={darkMode:'class',theme:{extend:{colors:{darkbg:'#0f172a',darkcard:'#1e293b'}}}}</script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
        body { font-family: 'Inter', sans-serif; transition: background-color 0.3s; }
        .toast { transform: translateX(100%); transition: all 0.4s cubic-bezier(0.68, -0.55, 0.27, 1.55); opacity: 0; }
        .toast.show { transform: translateX(0); opacity: 1; }
        .glass-card { background: rgba(255, 255, 255, 0.95); backdrop-filter: blur(10px); }
        .dark .glass-card { background: rgba(30, 41, 59, 0.9); }
        @keyframes breathe { 0%, 100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.7); } 50% { transform: scale(1.05); box-shadow: 0 0 20px 10px rgba(59, 130, 246, 0); } }
        .logo-breathe { animation: breathe 3s infinite; }
        .progress-bar { height: 4px; border-radius: 2px; background: #e2e8f0; margin-top: 4px; overflow: hidden; }
        .progress-fill { height: 100%; transition: width 0.5s ease; }
        th.sortable { cursor: pointer; transition: color 0.2s; } th.sortable:hover { color: #3b82f6; }
        /* 隐藏货币下拉框的默认箭头以保持简洁 */
        .currency-select { appearance: none; -webkit-appearance: none; -moz-appearance: none; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 dark:bg-darkbg dark:text-slate-200 min-h-screen flex flex-col">

    <div id="toast-container" class="fixed top-5 right-5 z-50">
        {{if .Message}}<div id="toast" class="toast show flex items-center p-4 mb-4 bg-white/90 backdrop-blur rounded-xl shadow-2xl dark:bg-slate-800/90 border-l-4 border-blue-500"><div class="text-blue-500 mr-3"><i class="fas fa-info-circle text-xl"></i></div><div class="text-sm font-medium">{{index .T .Message}}</div></div><script>setTimeout(() => { const t = document.getElementById('toast'); t.classList.remove('show'); setTimeout(()=>t.remove(), 400); }, 3500);</script>{{end}}
    </div>

    {{if or (eq .Page "login") (eq .Page "register") }}
    <div class="flex-1 flex flex-col items-center justify-center bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 relative">
        <div class="absolute top-6 right-6 flex gap-3">
             <form id="langForm" action="/set-lang" method="POST"><input type="hidden" name="page" value="{{.Page}}"><select name="lang" onchange="this.form.submit()" class="bg-white/10 text-white text-xs p-2 rounded-lg backdrop-blur border border-white/20 outline-none cursor-pointer hover:bg-white/20 transition"><option value="zh" {{if eq .User.Language "zh"}}selected{{end}}>🇨🇳 中文</option><option value="en" {{if eq .User.Language "en"}}selected{{end}}>🇺🇸 English</option></select></form>
             <button onclick="toggleDark()" class="bg-white/10 text-white p-2 rounded-lg w-9 h-9 flex items-center justify-center backdrop-blur border border-white/20 hover:bg-white/20 transition"><i class="fas fa-moon"></i></button>
        </div>
        
        <div class="glass-card p-10 rounded-3xl shadow-2xl w-full max-w-sm relative overflow-hidden transition-all duration-300">
            <div class="text-center mb-8"><div class="inline-flex items-center justify-center w-14 h-14 rounded-full bg-blue-100 text-blue-600 mb-4 logo-breathe"><i class="fas fa-shield-alt text-2xl"></i></div><h1 class="text-2xl font-extrabold tracking-tight text-slate-900 dark:text-white">ExpiryGuard</h1></div>
            {{if eq .Page "login"}}
                {{if eq .LoginStep "2fa"}}
                <form action="/login?step=2fa" method="POST" class="space-y-6">
                    <input type="hidden" name="username" value="{{.Username}}">
                    <p class="text-center text-green-500 text-sm font-medium"><i class="fas fa-check-circle"></i> {{.T.Msg_CodeSent}}</p>
                    <input type="text" name="code" class="w-full px-4 py-3 border rounded-xl text-center tracking-[0.5em] text-2xl font-bold font-mono dark:bg-slate-700 dark:border-slate-600 focus:ring-2 focus:ring-green-500 outline-none" placeholder="000000" maxlength="6" required autofocus>
                    <button type="submit" class="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3 rounded-xl shadow-lg transition">验证并登录</button>
                </form>
                <div class="mt-6 text-center text-sm"><a href="/login" class="text-slate-500 hover:text-slate-800 dark:hover:text-white transition">← 返回登录</a></div>
                {{else}}
                <form action="/login?step=login" method="POST" class="space-y-5"><div class="relative"><i class="fas fa-user absolute left-4 top-3.5 text-slate-400"></i><input type="text" name="username" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none focus:ring-2 focus:ring-blue-500 transition" placeholder="{{.T.User}}" required></div><div class="relative"><i class="fas fa-lock absolute left-4 top-3.5 text-slate-400"></i><input type="password" name="password" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none focus:ring-2 focus:ring-blue-500 transition" placeholder="{{.T.Pass}}" required></div><button type="submit" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-3.5 rounded-xl shadow-lg shadow-blue-500/30 transition transform hover:-translate-y-0.5">{{.T.BtnLogin}}</button></form>
                <div class="mt-8 text-center text-sm"><span class="text-slate-500">{{.T.NoAccount}}</span><a href="/register" class="text-blue-600 font-bold hover:underline ml-1">{{.T.SignUp}}</a></div>
                {{end}}
            {{else if eq .Page "register"}}
                <h2 class="text-center font-bold text-lg mb-6 text-slate-700 dark:text-slate-300">{{.T.RegTitle}}</h2><form action="/register" method="POST" class="space-y-5"><div class="relative"><i class="fas fa-user absolute left-4 top-3.5 text-slate-400"></i><input type="text" name="username" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600" placeholder="{{.T.User}}" required></div><div class="relative"><i class="fas fa-lock absolute left-4 top-3.5 text-slate-400"></i><input type="password" name="password" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600" placeholder="{{.T.Pass}}" required></div><button type="submit" class="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3.5 rounded-xl shadow-lg shadow-green-500/30 transition transform hover:-translate-y-0.5">{{.T.BtnReg}}</button></form><div class="mt-6 text-center text-sm"><a href="/login" class="text-slate-500 hover:text-slate-800 dark:hover:text-white transition">← {{.T.LoginTitle}}</a></div>
            {{end}}
            
            <div class="mt-10 text-center">
                <a href="https://t.me/TerrySiu98" target="_blank" class="text-xs text-slate-400 hover:text-blue-500 transition flex items-center justify-center gap-1 opacity-70 hover:opacity-100">
                    <span>Designed by Terry</span> <i class="fas fa-external-link-alt text-[9px]"></i>
                </a>
            </div>
        </div>
    </div>
    
    {{else}}
    <div class="flex flex-col md:flex-row min-h-screen">
        <aside class="w-64 bg-slate-900 text-slate-300 md:fixed md:inset-y-0 z-20 hidden md:flex flex-col shadow-2xl">
            <div class="p-6 flex items-center gap-3 text-white font-bold text-xl border-b border-slate-800"><i class="fas fa-shield-alt text-blue-500"></i> ExpiryGuard</div>
            <nav class="flex-1 p-4 space-y-1">
                <a href="/" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "home"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-th-large w-5 text-center"></i> <span>{{.T.Dashboard}}</span></a>
                <a href="/profile" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "profile"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-user-cog w-5 text-center"></i> <span>{{.T.Profile}}</span></a>
                {{if eq .User.Role "admin"}}
                <div class="text-[10px] font-bold text-slate-500 uppercase px-3 mb-2 mt-6">Admin</div>
                <a href="/admin" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "admin"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-cogs w-5 text-center"></i> <span>{{.T.GlobalSet}}</span></a>
                <a href="/admin/users" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "users"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-users w-5 text-center"></i> <span>{{.T.UserMgmt}}</span></a>
                {{end}}
            </nav>
            <div class="p-4 border-t border-slate-800 flex justify-between items-center gap-2">
                <form id="sideLangForm" action="/set-lang" method="POST" class="flex-1"><input type="hidden" name="page" value="{{.Page}}"><select name="lang" onchange="this.form.submit()" class="w-full bg-slate-800 text-xs text-slate-300 p-1 rounded border border-slate-700 outline-none"><option value="zh" {{if eq .User.Language "zh"}}selected{{end}}>🇨🇳 中文</option><option value="en" {{if eq .User.Language "en"}}selected{{end}}>🇺🇸 EN</option></select></form>
                <button onclick="toggleDark()" class="text-slate-400 hover:text-white p-1"><i class="fas fa-moon"></i></button>
                <a href="/logout" class="text-red-400 hover:text-white p-1"><i class="fas fa-sign-out-alt"></i></a>
            </div>
        </aside>

        <main class="flex-1 md:ml-64 w-full flex flex-col">
            <div class="md:hidden bg-slate-900 text-white p-4 flex justify-between items-center sticky top-0 z-30 shadow-md"><span class="font-bold">ExpiryGuard</span><button onclick="toggleMenu()" class="text-xl">☰</button></div>
            <div id="mobileMenu" class="fixed inset-0 bg-slate-900/95 z-40 hidden flex-col p-8 space-y-6 text-white text-lg font-bold md:hidden"><button onclick="toggleMenu()" class="absolute top-4 right-4 text-2xl">✕</button><a href="/">{{.T.Dashboard}}</a><a href="/profile">{{.T.Profile}}</a>{{if eq .User.Role "admin"}}<a href="/admin">{{.T.GlobalSet}}</a><a href="/admin/users">{{.T.UserMgmt}}</a>{{end}}<a href="/logout" class="text-red-400">{{.T.Logout}}</a></div>

            <div class="max-w-7xl mx-auto p-4 md:p-8 w-full flex-1">
                {{if eq .Page "home"}}
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-gradient-to-r from-blue-500 to-blue-600 text-white p-5 rounded-xl shadow-lg flex justify-between items-center"><div><div class="text-blue-100 text-xs font-bold uppercase">{{.T.Total}}</div><div class="text-3xl font-bold">{{.Stats.Total}}</div></div><div class="text-blue-300 text-4xl opacity-50"><i class="fas fa-cube"></i></div></div>
                    <div class="bg-gradient-to-r from-orange-400 to-orange-500 text-white p-5 rounded-xl shadow-lg flex justify-between items-center"><div><div class="text-orange-100 text-xs font-bold uppercase">{{.T.Expiring}}</div><div class="text-3xl font-bold">{{.Stats.Expiring}}</div></div><div class="text-orange-200 text-4xl opacity-50"><i class="fas fa-clock"></i></div></div>
                    <div class="bg-gradient-to-r from-red-500 to-red-600 text-white p-5 rounded-xl shadow-lg flex justify-between items-center"><div><div class="text-red-100 text-xs font-bold uppercase">{{.T.Urgent}}</div><div class="text-3xl font-bold">{{.Stats.Urgent}}</div></div><div class="text-red-300 text-4xl opacity-50"><i class="fas fa-exclamation-triangle"></i></div></div>
                    <div class="bg-gradient-to-r from-purple-500 to-pink-500 text-white p-5 rounded-xl shadow-lg flex justify-between items-center"><div><div class="text-purple-100 text-xs font-bold uppercase">{{.T.Cost30}} <span class="text-[10px] opacity-75 ml-1">({{.BaseCurName}})</span></div><div class="text-3xl font-bold">{{.BaseCurSymbol}}{{.Stats.ProjectedCost}}</div></div><div class="text-purple-300 text-4xl opacity-50"><i class="fas fa-wallet"></i></div></div>
                </div>

                <div class="bg-white dark:bg-darkcard rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 p-5">
                    <div class="flex flex-col md:flex-row justify-between items-center mb-4 gap-3">
                        <div class="flex gap-2 items-center w-full md:w-auto"><h2 class="text-lg font-bold">{{.T.MyAssets}}</h2><a href="/export" class="text-xs bg-slate-100 dark:bg-slate-700 px-3 py-1.5 rounded-lg hover:bg-slate-200 transition font-medium text-slate-600 dark:text-slate-300"><i class="fas fa-download mr-1"></i>{{.T.Export}}</a><button onclick="openModal('importModal')" class="bg-slate-100 dark:bg-slate-700 px-3 py-1.5 text-xs font-bold rounded-lg transition text-slate-600 dark:text-slate-300"><i class="fas fa-file-import mr-1"></i> {{.T.Import}}</button></div>
                        <div class="relative w-full md:w-64"><input type="text" id="searchInput" placeholder="{{.T.Search}}" class="w-full pl-9 pr-4 py-2 bg-slate-50 dark:bg-slate-800 border-none rounded-lg text-sm focus:ring-2 focus:ring-blue-500 transition"><i class="fas fa-search absolute left-3 top-2.5 text-slate-400 text-xs"></i></div>
                    </div>
                    <button onclick="openModal('addModal')" class="w-full bg-slate-800 dark:bg-blue-600 text-white py-3 rounded-lg text-sm font-bold shadow-md hover:bg-slate-900 transition mb-4"><i class="fas fa-plus mr-1"></i> {{.T.Add}}</button>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left border-collapse min-w-[700px]" id="assetTable">
                            <thead class="text-xs text-slate-400 uppercase border-b dark:border-slate-700"><tr><th class="pb-3 pl-2 sortable" onclick="sortTable(0)">{{.T.Cat}} ⇅</th><th class="pb-3 sortable" onclick="sortTable(1)">{{.T.Name}} ⇅</th><th class="pb-3 sortable" onclick="sortNumber(2)">{{.T.Cost}} ⇅</th><th class="pb-3 w-1/4 sortable" onclick="sortTable(3)">{{.T.Date}} ⇅</th><th class="pb-3 text-right pr-2">{{.T.Action}}</th></tr></thead>
                            <tbody class="text-sm">
                                {{range .Items}}
                                <tr class="border-b dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 search-item transition-all duration-200">
                                    <td class="py-3 pl-2"><span class="px-2.5 py-1 bg-slate-100 dark:bg-slate-700 rounded-full text-xs font-medium cat-label">{{.Category}}</span></td>
                                    <td class="py-3 font-semibold text-slate-700 dark:text-slate-200 search-text">{{.Name}} {{if .Link}}<a href="{{.Link}}" target="_blank" class="text-blue-500 ml-1" title="直达链接"><i class="fas fa-external-link-alt text-[10px]"></i></a>{{end}}</td>
                                    <td class="py-3 font-mono text-slate-500 font-medium">{{.DisplaySymbol}}{{.Cost}} <span class="text-[10px] text-slate-400 font-sans ml-1">({{.Currency}})</span></td>
                                    <td class="py-3"><div class="font-mono text-blue-600 dark:text-blue-400 font-medium">{{.Date}}</div><div class="progress-bar dark:bg-slate-700" data-date="{{.Date}}"><div class="progress-fill"></div></div></td>
                                    <td class="py-3 text-right pr-2 space-x-2"><button onclick="openView('{{.Name}}', '{{.Detail}}', '{{.Link}}')" class="text-green-600 bg-green-50 dark:bg-green-900/30 px-2.5 py-1.5 rounded-lg text-xs font-medium"><i class="fas fa-eye"></i></button><button onclick="openEdit('{{.ID}}','{{.Category}}','{{.Name}}','{{.Date}}','{{.Cost}}','{{.Currency}}','{{.Link}}','{{.Detail}}')" class="text-blue-500 p-1"><i class="fas fa-edit"></i></button><form action="/item/del" method="POST" onsubmit="return confirm('{{$.T.ConfirmDel}}')" class="inline"><input type="hidden" name="id" value="{{.ID}}"><button class="text-slate-400 hover:text-red-500 p-1"><i class="fas fa-trash-alt"></i></button></form></td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{else if eq .Page "profile"}}
                <div class="bg-white dark:bg-darkcard rounded-xl shadow-sm p-6 border border-slate-100 dark:border-slate-700">
                    <h2 class="text-xl font-bold mb-6 flex items-center gap-2"><i class="fas fa-bell text-blue-500"></i> {{.T.NotifySettings}}</h2>
                    {{if and .Settings.tg_bot_username .Settings.tg_token}}
                    <div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-xl mb-6 flex justify-between items-center border border-blue-100 dark:border-blue-800"><div class="text-blue-800 dark:text-blue-300 font-bold flex items-center gap-2"><div class="relative flex h-3 w-3"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span><span class="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span></div> 机器人在线</div><a href="https://t.me/{{.Settings.tg_bot_username}}" target="_blank" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-bold shadow-md transition">打开机器人</a></div>
                    {{end}}
                    <form action="/profile/update" method="POST" class="space-y-5">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5"><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.ChatID}} (Telegram)</label><div class="flex gap-2"><input type="text" name="chat_id" value="{{.User.ChatID}}" class="flex-1 p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 focus:ring-2 focus:ring-blue-500 outline-none"><button type="button" onclick="window.open('https://t.me/userinfobot')" class="px-3 bg-slate-100 dark:bg-slate-700 text-xs font-bold rounded-lg hover:bg-slate-200 transition">{{.T.GetID}}</button></div></div><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.WebhookUrl}} (Feishu/DingTalk)</label><input type="text" name="email" value="{{.User.Email}}" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 focus:ring-2 focus:ring-blue-500 outline-none"></div></div>
                        <div class="grid grid-cols-1 md:grid-cols-3 gap-5">
                            <div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.Timezone}}</label><select name="timezone" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 outline-none">{{range .Timezones}}<option value="{{.Val}}" {{if eq $.User.Timezone .Val}}selected{{end}}>{{index $.T .LabelKey}}</option>{{end}}</select></div>
                            <div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.NotifyTime}}</label><select name="notify_time" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 outline-none">{{range .Hours}}<option value="{{.}}" {{if eq $.User.NotifyTime .}}selected{{end}}>{{printf "%02d:00" .}}</option>{{end}}</select></div>
                            <div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.BaseCurrency}}</label><select name="base_currency" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 outline-none">{{range .CurrencyKeys}}<option value="{{.}}" {{if eq $.User.BaseCurrency .}}selected{{end}}>{{index $.CurrencyMap . "Name"}}</option>{{end}}</select></div>
                        </div>
                        <button type="submit" class="w-full bg-slate-800 dark:bg-blue-600 text-white py-3 rounded-lg font-bold shadow-lg hover:bg-slate-900 transition">{{.T.Save}}</button>
                    </form>
                    <form action="/test-notify" method="POST" class="mt-6 pt-6 border-t dark:border-slate-700"><button class="w-full bg-purple-50 text-purple-600 py-3 rounded-lg border border-purple-200 text-sm font-bold dark:bg-purple-900/10 dark:border-purple-800 dark:text-purple-300 transition"><i class="fas fa-paper-plane mr-1"></i> {{.T.TestBtn}}</button></form>
                </div>
                {{else if eq .Page "admin"}}
                <div class="max-w-4xl mx-auto bg-white dark:bg-darkcard rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 p-6">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-xl font-bold flex items-center gap-2"><i class="fas fa-cogs text-slate-500"></i> {{.T.GlobalSet}}</h2>
                        <a href="/admin/backup" target="_blank" class="bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 px-3 py-1.5 rounded-lg text-xs font-bold hover:bg-slate-200 dark:hover:bg-slate-600 transition"><i class="fas fa-database mr-1"></i> {{.T.Backup}}</a>
                    </div>
                    <form action="/admin/update" method="POST" class="space-y-5">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5"><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">TG Bot Username</label><input type="text" name="tg_bot_username" value="{{.Settings.tg_bot_username}}" placeholder="@YourBot" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 outline-none"></div><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">TG Bot Token</label><input type="password" name="tg_token" value="{{.Settings.tg_token}}" placeholder="123456:ABC-DEF..." class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 outline-none"></div></div>
                        <div class="flex gap-3">
                           <button type="submit" class="flex-1 bg-slate-800 text-white py-3 rounded-lg font-bold dark:bg-blue-600 shadow-lg">{{.T.Save}}</button>
                           <a href="/admin/tg-webhook" class="flex-1 bg-blue-50 text-blue-600 flex items-center justify-center rounded-lg font-bold border border-blue-200 hover:bg-blue-100 transition"><i class="fas fa-plug mr-1"></i> {{.T.SetWebhook}}</a>
                        </div>
                    </form>
                    <div class="mt-8 pt-8 border-t dark:border-slate-700"><h3 class="text-lg font-bold mb-3 flex items-center gap-2 text-slate-700 dark:text-slate-300"><i class="fas fa-vial"></i> {{.T.Simulate_Title}}</h3><form action="/admin/simulate" method="POST"><button type="submit" class="bg-orange-500 hover:bg-orange-600 text-white px-5 py-2.5 rounded-lg font-bold shadow transition"><i class="fas fa-bell mr-1"></i> {{.T.Simulate_Btn}}</button></form></div>
                </div>
                {{else if eq .Page "users"}}
                <div class="bg-white dark:bg-darkcard rounded-xl p-6">
                    <h2 class="text-xl font-bold mb-4">{{.T.UserMgmt}}</h2>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left border-collapse min-w-[700px]"><thead class="text-xs uppercase text-slate-400 border-b dark:border-slate-700"><tr><th class="pb-2">{{.T.Head_ID}}</th><th class="pb-2">{{.T.Head_User}}</th><th class="pb-2">{{.T.Head_Role}}</th><th class="pb-2">{{.T.Head_Info}}</th><th class="pb-2">{{.T.Head_IP}}</th><th class="pb-2">{{.T.Head_Action}}</th></tr></thead>
                        <tbody>{{range .Users}}<tr class="border-b dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800"><td class="py-3 text-xs text-slate-400">#{{.ID}}</td><td class="font-bold">{{.Username}}</td><td><span class="px-2 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-xs">{{.Role}}</span></td><td class="text-xs"><span class="mr-2 font-bold">{{len .Items}} Assets</span></td><td class="text-xs font-mono text-slate-400">{{if .LastIP}}{{.LastIP}}{{else}}未知{{end}}</td><td class="py-3">{{if ne .Role "admin"}}<div class="flex items-center gap-2"><form action="/admin/reset-pwd" method="POST" onsubmit="return confirm('{{$.T.ConfirmReset}}')" class="m-0"><input type="hidden" name="id" value="{{.ID}}"><button class="bg-blue-100 text-blue-600 hover:bg-blue-200 px-3 py-1 rounded-lg text-xs font-bold transition">{{$.T.ResetPwd}}</button></form><form action="/admin/users/del" method="POST" onsubmit="return confirm('{{$.T.ConfirmDelUser}}')" class="m-0"><input type="hidden" name="id" value="{{.ID}}"><button class="bg-red-100 text-red-600 hover:bg-red-200 px-3 py-1 rounded-lg text-xs font-bold transition">{{$.T.DelUser}}</button></form></div>{{end}}</td></tr>{{end}}</tbody></table>
                    </div>
                </div>
                {{end}}
            </div>
        </main>
    </div>
    {{end}}

    <div id="addModal" class="fixed inset-0 z-[100] hidden bg-black/50 backdrop-blur-sm items-center justify-center p-4"><div class="glass-card p-6 rounded-2xl w-full max-w-lg shadow-2xl relative"><h3 class="font-bold mb-4 text-lg">{{.T.Add}}</h3><form action="/item/add" method="POST" class="space-y-4">
        <input type="text" name="category" list="cat_list" placeholder="{{.T.Ph_Cat}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required><datalist id="cat_list">{{range .UserCats}}<option value="{{.}}">{{end}}</datalist>
        <input type="text" name="name" placeholder="{{.T.Name}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required>
        <div class="flex gap-2">
            <input type="date" name="date" class="w-1/2 p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required>
            <div class="w-1/2 flex border rounded-xl dark:bg-slate-700 dark:border-slate-600 overflow-hidden focus-within:ring-2 focus-within:ring-blue-500">
                <select name="currency" class="currency-select bg-slate-100 dark:bg-slate-600 px-3 outline-none border-r dark:border-slate-500 text-sm font-bold text-center cursor-pointer">
                    {{range .CurrencyKeys}}<option value="{{.}}" {{if eq $.User.BaseCurrency .}}selected{{end}}>{{index $.CurrencyMap . "Symbol"}}</option>{{end}}
                </select>
                <input type="number" step="0.01" name="cost" placeholder="{{.T.Cost}} (Opt)" class="w-full p-3 bg-transparent outline-none font-mono">
            </div>
        </div>
        <input type="url" name="link" placeholder="{{.T.Link}} (Opt)" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none">
        <textarea name="detail" rows="3" placeholder="{{.T.Details}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 font-mono text-sm outline-none"></textarea>
        <div class="flex gap-3"><button type="button" onclick="closeModal('addModal')" class="flex-1 py-3 bg-slate-100 text-slate-600 rounded-xl font-bold">{{.T.Cancel}}</button><button type="submit" class="flex-1 py-3 bg-blue-600 text-white rounded-xl font-bold shadow-lg">{{.T.Save}}</button></div>
    </form></div></div>
    
    <div id="editModal" class="fixed inset-0 z-[100] hidden bg-black/50 backdrop-blur-sm items-center justify-center p-4"><div class="glass-card p-6 rounded-2xl w-full max-w-lg shadow-2xl relative"><h3 class="font-bold mb-4 text-lg">{{.T.Edit}}</h3><form action="/item/update" method="POST" class="space-y-4">
        <input type="hidden" name="id" id="edit_id">
        <input type="text" name="category" id="edit_category" list="cat_list" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required>
        <input type="text" name="name" id="edit_name" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none">
        <div class="flex gap-2">
            <input type="date" name="date" id="edit_date" class="w-1/2 p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none">
            <div class="w-1/2 flex border rounded-xl dark:bg-slate-700 dark:border-slate-600 overflow-hidden focus-within:ring-2 focus-within:ring-blue-500">
                <select name="currency" id="edit_currency" class="currency-select bg-slate-100 dark:bg-slate-600 px-3 outline-none border-r dark:border-slate-500 text-sm font-bold text-center cursor-pointer">
                    {{range .CurrencyKeys}}<option value="{{.}}">{{index $.CurrencyMap . "Symbol"}}</option>{{end}}
                </select>
                <input type="number" step="0.01" name="cost" id="edit_cost" class="w-full p-3 bg-transparent outline-none font-mono">
            </div>
        </div>
        <input type="url" name="link" id="edit_link" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none">
        <textarea name="detail" id="edit_detail" rows="3" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 font-mono text-sm outline-none"></textarea>
        <div class="flex gap-3"><button type="button" onclick="closeModal('editModal')" class="flex-1 py-3 bg-slate-100 text-slate-600 rounded-xl font-bold">{{.T.Cancel}}</button><button type="submit" class="flex-1 py-3 bg-blue-600 text-white rounded-xl font-bold shadow-lg">{{.T.Save}}</button></div>
    </form></div></div>

    <div id="viewModal" class="fixed inset-0 z-[100] hidden bg-black/50 backdrop-blur-sm items-center justify-center p-4"><div class="glass-card p-6 rounded-2xl w-full max-w-2xl mx-4 shadow-2xl relative"><div class="flex justify-between items-start mb-4"><h3 class="font-bold text-xl" id="view_title"></h3><div class="flex gap-2"><a href="#" id="view_link_btn" target="_blank" class="hidden text-white bg-blue-500 hover:bg-blue-600 text-sm font-bold px-3 py-1 rounded-lg transition"><i class="fas fa-external-link-alt"></i> Renew</a><button onclick="copyContent()" class="text-slate-500 hover:text-slate-700 text-sm font-bold"><i class="fas fa-copy"></i></button></div></div><div class="bg-slate-50 dark:bg-slate-900 p-6 rounded-xl border dark:border-slate-700 mb-6 max-h-[60vh] overflow-y-auto"><pre id="view_content" class="whitespace-pre-wrap font-mono text-sm break-all text-slate-700 dark:text-slate-300"></pre></div><button onclick="closeModal('viewModal')" class="w-full py-3 bg-slate-800 text-white rounded-xl font-bold">{{.T.Close}}</button></div></div>

    <div id="importModal" class="fixed inset-0 z-[100] hidden bg-black/50 backdrop-blur-sm items-center justify-center p-4"><div class="bg-white dark:bg-darkcard rounded-xl p-6 max-w-md w-full mx-4 shadow-2xl relative"><h3 class="text-lg font-bold mb-4">导入资产</h3><form action="/import" method="POST" enctype="multipart/form-data"><input type="file" name="file" accept=".json" required class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 mb-4"><div class="flex gap-2"><button type="submit" class="flex-1 bg-blue-600 text-white py-2 rounded-lg font-bold hover:bg-blue-700">导入</button><button type="button" onclick="closeModal('importModal')" class="flex-1 bg-slate-200 dark:bg-slate-700 py-2 rounded-lg font-bold">取消</button></div></form></div></div>

    <script>
        const searchInput = document.getElementById('searchInput');
        if(searchInput){ 
            searchInput.addEventListener('input', e => { 
                const t = e.target.value.toLowerCase(); 
                document.querySelectorAll('.search-item').forEach(i => i.style.display = i.innerText.toLowerCase().includes(t)?'':'none'); 
            }); 
        }
        function toggleMenu(){
            document.getElementById('mobileMenu').classList.toggle('hidden');
            document.getElementById('mobileMenu').classList.toggle('flex');
        }
        function toggleDark(){
            document.documentElement.classList.toggle('dark');
            localStorage.setItem('theme', document.documentElement.classList.contains('dark') ? 'dark' : 'light');
        }
        if(localStorage.getItem('theme')==='dark'){document.documentElement.classList.add('dark');}
        
        function openModal(id){
            const el = document.getElementById(id);
            el.classList.remove('hidden');
            el.classList.add('flex');
        }
        function closeModal(id){
            const el = document.getElementById(id);
            el.classList.add('hidden');
            el.classList.remove('flex');
        }
        function openEdit(id,c,n,d,cost,cur,l,det){
            document.getElementById('edit_id').value=id; document.getElementById('edit_category').value=c;
            document.getElementById('edit_name').value=n; document.getElementById('edit_date').value=d;
            document.getElementById('edit_cost').value=cost; document.getElementById('edit_currency').value=cur||'CNY';
            document.getElementById('edit_link').value=l; document.getElementById('edit_detail').value=det; 
            openModal('editModal');
        }
        function openView(n,det,l){
            document.getElementById('view_title').innerText=n; 
            document.getElementById('view_content').innerText=det;
            const btn=document.getElementById('view_link_btn');
            if(l && l !== ''){btn.href=l;btn.classList.remove('hidden');btn.classList.add('inline-block');}
            else{btn.classList.add('hidden');}
            openModal('viewModal');
        }
        function copyContent(){navigator.clipboard.writeText(document.getElementById('view_content').innerText);}
        
        function getIcon(cat) { 
            const c = cat.toLowerCase(); 
            if(c.includes('域名')||c.includes('domain')) return '🌐 '; 
            if(c.includes('服务器')||c.includes('vps')||c.includes('node')) return '💻 '; 
            if(c.includes('证书')||c.includes('ssl')) return '🔒 '; 
            return '📁 '; 
        }
        
        document.querySelectorAll('.cat-label').forEach(el => { el.innerText = getIcon(el.innerText) + el.innerText; });
        document.querySelectorAll('.progress-bar').forEach(bar => {
            const diffDays = Math.ceil((new Date(bar.dataset.date) - new Date()) / 86400000);
            let c = '#22c55e'; 
            if(diffDays<=30) c='#f97316'; 
            if(diffDays<=7) c='#ef4444'; 
            if(diffDays<0) c='#94a3b8';
            const f = bar.querySelector('.progress-fill'); 
            f.style.width = Math.min(100, Math.max(0, (diffDays/365)*100)) + '%'; 
            f.style.backgroundColor = c;
        });

        let sortDir = true;
        function sortTable(col) { 
            const tb = document.getElementById("assetTable").tBodies[0]; 
            const rows = Array.from(tb.rows); 
            sortDir = !sortDir; 
            rows.sort((a,b) => a.cells[col].innerText.localeCompare(b.cells[col].innerText) * (sortDir ? 1 : -1)); 
            tb.append(...rows); 
        }
        function sortNumber(col) { 
            const tb = document.getElementById("assetTable").tBodies[0]; 
            const rows = Array.from(tb.rows); 
            sortDir = !sortDir; 
            rows.sort((a,b) => {
                // 提取纯数字部分进行排序，忽略货币符号
                let valA = parseFloat(a.cells[col].innerText.replace(/[^0-9.-]+/g,"")) || 0;
                let valB = parseFloat(b.cells[col].innerText.replace(/[^0-9.-]+/g,"")) || 0;
                return (valA - valB) * (sortDir ? 1 : -1);
            }); 
            tb.append(...rows); 
        }
    </script>
</body>
</html>`;

// ===== AST Template Renderer =====
function renderTemplate(data) {
    const tokens = []; 
    const regex = /{{([\s\S]*?)}}/g; 
    let lastIndex = 0, match;
    
    while ((match = regex.exec(HTML_TEMPLATE)) !== null) {
        if (match.index > lastIndex) tokens.push({ t: 'txt', v: HTML_TEMPLATE.slice(lastIndex, match.index) });
        let c = match[1].trim();
        if (c.startsWith('if ')) tokens.push({ t: 'if', c: c.slice(3) }); 
        else if (c.startsWith('else if ')) tokens.push({ t: 'elseif', c: c.slice(8) }); 
        else if (c === 'else') tokens.push({ t: 'else' }); 
        else if (c.startsWith('range ')) tokens.push({ t: 'range', c: c.slice(6) }); 
        else if (c === 'end') tokens.push({ t: 'end' }); 
        else tokens.push({ t: 'var', v: c });
        lastIndex = regex.lastIndex;
    }
    if (lastIndex < HTML_TEMPLATE.length) tokens.push({ t: 'txt', v: HTML_TEMPLATE.slice(lastIndex) });
  
    function getVal(k, ctx) {
        if (k === '.') return ctx; 
        if (k.startsWith('$.T.')) return data.T?.[k.slice(4)];
        if (k.startsWith('$.')) { 
            let parts = k.slice(2).split('.'); let v = data; 
            for (let p of parts) v = v?.[p]; return v; 
        }
        if (k.startsWith('"') && k.endsWith('"')) return k.slice(1, -1); 
        if (!isNaN(k)) return Number(k);
        let target = k.startsWith('.') ? k.slice(1) : k; 
        let parts = target.split('.'); let v = ctx;
        for (let p of parts) v = v?.[p]; 
        if (v !== undefined) return v; 
        v = data; 
        for (let p of parts) v = v?.[p]; 
        return v;
    }

    function evalCond(cond, ctx) {
        if (cond.startsWith('and ')) {
            const parts = cond.substring(4).split(/\s+(?=[\w\.\$])/);
            return parts.every(p => evalCond(p.trim(), ctx));
        }
        if (cond.startsWith('or ')) {
            return cond.substring(3).match(/\(([^)]+)\)/g).some(p => evalCond(p.slice(1, -1), ctx));
        }
        const m = cond.match(/^(eq|ne)\s+([\w\.\$]+)\s+(.+)$/); 
        if (m) { 
            let v1 = getVal(m[2], ctx), v2 = getVal(m[3], ctx); 
            return m[1] === 'eq' ? v1 === v2 : v1 !== v2; 
        }
        return !!getVal(cond, ctx);
    }

    function run(toks, ctx) {
        let out = ''; let i = 0;
        while (i < toks.length) {
            let t = toks[i];
            if (t.t === 'txt') { out += t.v; i++; }
            else if (t.t === 'var') {
                if (t.v.startsWith('index $.T .')) { let key = getVal(t.v.slice(11), ctx); out += data.T?.[key] || ''; }
                else if (t.v.startsWith('index .T .')) { let key = getVal(t.v.slice(10), ctx); out += data.T?.[key] || ''; }
                else if (t.v.startsWith('index $.CurrencyMap .')) { 
                    let parts = t.v.split(' '); let k = getVal(parts[2], ctx); let f = getVal(parts[3], ctx);
                    out += data.CurrencyMap?.[k]?.[f] || '';
                }
                else if (t.v.startsWith('printf "%02d:00" ')) { let v = getVal(t.v.slice(17), ctx); out += (v < 10 ? '0' + v : v) + ':00'; }
                else if (t.v.startsWith('len ')) { out += (getVal(t.v.slice(4), ctx) || []).length; }
                else { let res = getVal(t.v, ctx); out += (res !== undefined && res !== null) ? res : ''; } 
                i++;
            }
            else if (t.t === 'if') {
                let blocks = [{ c: t.c, toks: [] }], cur = blocks[0], depth = 1; i++;
                while (i < toks.length && depth > 0) { 
                    let inner = toks[i]; 
                    if (inner.t === 'if' || inner.t === 'range') depth++; 
                    else if (inner.t === 'end') depth--;
                    
                    if (depth === 1 && inner.t === 'elseif') { 
                        cur = { c: inner.c, toks: [] }; blocks.push(cur); i++; continue; 
                    } else if (depth === 1 && inner.t === 'else') { 
                        cur = { c: 'true', toks: [] }; blocks.push(cur); i++; continue; 
                    }
                    if (depth > 0) cur.toks.push(inner); 
                    i++;
                }
                for (let b of blocks) { 
                    if (b.c === 'true' || evalCond(b.c, ctx)) { out += run(b.toks, ctx); break; } 
                }
            }
            else if (t.t === 'range') {
                let innerToks = [], depth = 1; i++;
                while (i < toks.length && depth > 0) { 
                    let inner = toks[i]; 
                    if (inner.t === 'if' || inner.t === 'range') depth++; 
                    else if (inner.t === 'end') depth--; 
                    if (depth > 0) innerToks.push(inner); 
                    i++; 
                }
                let arr = getVal(t.c, ctx) || []; 
                for (let item of arr) out += run(innerToks, item);
            } else { i++; }
        } 
        return out;
    }
    return run(tokens, data);
}

// ===== Language Pack =====
const LangMap = {
  zh: { BaseCurrency: "本位币", Msg_CodeSent: "验证码已发送至您的设备", LoginTitle: "登录系统", User: "用户名", Pass: "密码", BtnLogin: "登录", NoAccount: "没有账号？", SignUp: "立即注册", RegTitle: "注册", BtnReg: "确认注册", Dashboard: "控制台", Profile: "个人设置", Admin: "系统管理", Logout: "退出", Total: "总资产", Expiring: "30天内到期", Urgent: "急需处理", Cost30: "30天预估", MyAssets: "资产列表", Export: "导出", Import: "导入", Search: "搜索...", Cat: "分类", Name: "名称", Date: "到期日", Cost: "金额", Link: "直达链接", Action: "操作", Add: "添加资产", Details: "详情", View: "查看", Edit: "编辑", Del: "删除", Save: "保存", Cancel: "取消", Close: "关闭", NotifySettings: "通知设置", ChatID: "TG ID", WebhookUrl: "Webhook URL", Timezone: "时区", NotifyTime: "每日推送", GetID: "获取 ID", TestBtn: "发送测试", GlobalSet: "系统设置", UserMgmt: "用户管理", SetWebhook: "一键绑定 TG Webhook", Backup: "备份", Simulate_Title: "报警测试", Simulate_Btn: "测试报警", Head_ID: "ID", Head_User: "用户", Head_Role: "角色", Head_Info: "信息", Head_IP: "IP", Head_Action: "操作", ResetPwd: "重置密码", DelUser: "删除", ConfirmReset: "确定要重置该用户的密码为 123456 吗？", ConfirmDelUser: "确定要彻底删除该用户及其所有资产吗？", Msg_Add: "添加成功", Msg_Upd: "更新成功", Msg_Del: "删除成功", Msg_Saved: "保存成功", Msg_Sent: "发送成功", Msg_WebhookSet: "TG Webhook 绑定成功！", Msg_LoginSuccess: "登录成功", Msg_RegSuccess: "注册成功", Msg_LoginErr: "登录失败", Msg_UserExists: "用户已存在", TZ_Shanghai: "🇨🇳 中国 (北京)", TZ_Tokyo: "🇯🇵 日本 (东京)", TZ_Seoul: "🇰🇷 韩国 (首尔)", TZ_Singapore: "🇸🇬 新加坡", TZ_London: "🇬🇧 英国 (伦敦)", TZ_Berlin: "🇩🇪 德国 (柏林)", TZ_NY: "🇺🇸 美国 (纽约)", TZ_LA: "🇺🇸 美国 (洛杉矶)", TZ_UTC: "🌐 UTC" },
  en: { BaseCurrency: "Base Currency", Msg_CodeSent: "2FA Code sent to your device", LoginTitle: "Login", User: "User", Pass: "Pass", BtnLogin: "Sign In", NoAccount: "No account?", SignUp: "Sign up", RegTitle: "Register", BtnReg: "Register", Dashboard: "Dashboard", Profile: "Settings", Admin: "Admin", Logout: "Logout", Total: "Assets", Expiring: "Expiring(30d)", Urgent: "Urgent", Cost30: "30d Cost", MyAssets: "My Assets", Export: "Export", Import: "Import", Search: "Search...", Cat: "Cat", Name: "Name", Date: "Due Date", Cost: "Cost", Link: "Link", Action: "Action", Add: "Add", Details: "Details", View: "View", Edit: "Edit", Del: "Del", Save: "Save", Cancel: "Cancel", Close: "Close", NotifySettings: "Notifications", ChatID: "TG ID", WebhookUrl: "Webhook URL", Timezone: "Timezone", NotifyTime: "Time", GetID: "Get ID", TestBtn: "Test Alert", GlobalSet: "Global Settings", UserMgmt: "Users", SetWebhook: "Bind TG Webhook", Backup: "Backup", Simulate_Title: "Simulate", Simulate_Btn: "Run", Head_ID: "ID", Head_User: "User", Head_Role: "Role", Head_Info: "Info", Head_IP: "IP", Head_Action: "Action", ResetPwd: "Reset Pwd", DelUser: "Del", ConfirmReset: "Reset user password to 123456?", ConfirmDelUser: "Delete user and all their assets?", Msg_Add: "Added", Msg_Upd: "Updated", Msg_Del: "Deleted", Msg_Saved: "Saved", Msg_Sent: "Sent", Msg_WebhookSet: "Webhook Set!", Msg_LoginSuccess: "Welcome", Msg_RegSuccess: "Registered", Msg_LoginErr: "Error", Msg_UserExists: "Exists", TZ_Shanghai: "🇨🇳 China", TZ_Tokyo: "🇯🇵 Japan", TZ_Seoul: "🇰🇷 South Korea (Seoul)", TZ_Singapore: "🇸🇬 Singapore", TZ_London: "🇬🇧 UK (London)", TZ_Berlin: "🇩🇪 Germany (Berlin)", TZ_NY: "🇺🇸 USA (New York)", TZ_LA: "🇺🇸 USA (Los Angeles)", TZ_UTC: "🌐 UTC" }
};

// ===== Route Handlers =====
async function handleExport(request, storage, user) {
    const items = await storage.getItems(user.ID);
    const data = JSON.stringify(items, null, 2);
    return new Response(data, {
        headers: {
            'Content-Type': 'application/json',
            'Content-Disposition': `attachment; filename="expiry-guard-export-${Date.now()}.json"`
        }
    });
}

async function handleImport(request, storage, user) {
    try {
        const fd = await request.formData();
        const file = fd.get('file');
        if (!file) {
            return Response.redirect(new URL('/?msg=导入失败：未选择文件', request.url).toString(), 302);
        }
        const text = await file.text();
        const items = JSON.parse(text);
        const existing = await storage.getItems(user.ID);
        const merged = [...existing, ...items];
        await storage.saveItems(user.ID, merged);
        return Response.redirect(new URL('/?msg=导入成功', request.url).toString(), 302);
    } catch (e) {
        return Response.redirect(new URL(`/?msg=导入失败：${e.message}`, request.url).toString(), 302);
    }
}

async function handleAdminBackup(request, storage, user) {
    const users = await storage.getUsers();
    const settings = await storage.getSettings();
    const allItems = {};
    for (const u of users) {
        const items = await storage.getItems(u.ID);
        if (items && items.length > 0) {
            allItems[u.Username] = items;
        }
    }
    const backup = {
        users: users,
        settings: settings,
        items: allItems,
        timestamp: new Date().toISOString()
    };
    return new Response(JSON.stringify(backup, null, 2), {
        headers: {
            'Content-Type': 'application/json',
            'Content-Disposition': `attachment; filename="expiry-guard-backup-${Date.now()}.json"`
        }
    });
}

async function handleRequest(request, env) {
    const url = new URL(request.url); 
    const path = url.pathname; 
    const storage = new R2Storage(env.BUCKET);
  
    if (path === '/telegram-webhook' && request.method === 'POST') {
        return handleTelegramWebhook(request, storage, env);
    }
  
    if (path === '/login') return handleLogin(request, storage);
    if (path === '/register') return handleRegister(request, storage);
    if (path === '/set-lang') return handleSetLang(request, storage);
    
    if (path === '/logout') {
        return new Response('', { 
            status: 302, 
            headers: { 'Location': '/login', 'Set-Cookie': setCookie('session', '', 0) }
        });
    }

    const user = await getAuthUser(request, storage);
    if (!user) return Response.redirect(new URL('/login', request.url).toString(), 302);

    if (path === '/') return handleHome(request, storage, user);
    if (path === '/item/add') return handleAddItem(request, storage, user);
    if (path === '/item/update') return handleUpdateItem(request, storage, user);
    if (path === '/item/del') return handleDeleteItem(request, storage, user);
    
    if (path === '/profile') return handleProfile(request, storage, user);
    if (path === '/profile/update') return handleProfileUpdate(request, storage, user);
    if (path === '/test-notify') return handleTestNotify(request, storage, user);

    if (user.Role !== 'admin') return Response.redirect(new URL('/', request.url).toString(), 302);
    if (path === '/export') return handleExport(request, storage, user);
    if (path === '/import' && request.method === 'POST') return handleImport(request, storage, user);
    if (path === '/admin/backup') return handleAdminBackup(request, storage, user);
    
    if (path === '/admin') return handleAdmin(request, storage, user);
    if (path === '/admin/update') return handleAdminUpdate(request, storage, user);
    if (path === '/admin/tg-webhook') return handleAdminSetWebhook(request, storage);
    if (path === '/admin/users') return handleAdminUsers(request, storage, user);
    if (path === '/admin/simulate') return handleAdminSimulate(request, storage, user);
    
    if (path === '/admin/reset-pwd' && request.method === 'POST') return handleAdminResetPwd(request, storage, user);
    if (path === '/admin/users/del' && request.method === 'POST') return handleAdminDelUser(request, storage, user);

    return new Response('Not Found', { status: 404 });
}

// -- Auth Logic --
async function handleLogin(request, storage) {
    const url = new URL(request.url);
    const msg = url.searchParams.get('msg');
    const step = url.searchParams.get('step') || 'login';
    
    if (request.method === 'GET') {
        return new Response(renderTemplate({ Page: 'login', User: { Language: 'zh' }, T: LangMap.zh, LoginStep: step, Message: msg }), { headers: { 'Content-Type': 'text/html' } });
    }
    
    const fd = await request.formData();
    
    if (step === 'login') {
        const users = await storage.getUsers(); 
        const hashedPwd = await hashPassword(fd.get('password'));
        const user = users.find(x => x.Username === fd.get('username') && x.Password === hashedPwd);
        
        if (!user) {
            return new Response(renderTemplate({ Page: 'login', User: { Language: 'zh' }, T: LangMap.zh, LoginStep: 'login', Message: 'Msg_LoginErr' }), { headers: { 'Content-Type': 'text/html' } });
        }
        
        if (user.ChatID) {
            const settings = await storage.getSettings();
            if (settings.tg_token) {
                const code = generate2FACode();
                await save2FACode(storage, user.Username, code);
                const ip = request.headers.get('CF-Connecting-IP') || '未知 IP';
                const tgMsg = `<b>登录验证</b>\n\n您的登录验证码是： <b>${code}</b>\n\n请求来源 IP： ${ip}\n\n<i>🛡️ ExpiryGuard System</i>`;
                await sendTelegramNotification(settings.tg_token, user.ChatID, tgMsg);
                return new Response(renderTemplate({ Page: 'login', User: { Language: 'zh' }, T: LangMap.zh, LoginStep: '2fa', Username: user.Username, Message: 'Msg_CodeSent' }), { headers: { 'Content-Type': 'text/html' } });
            }
        }
        
        const ip = request.headers.get('CF-Connecting-IP') || '未知 IP';
        const userIdx = users.findIndex(x => x.Username === user.Username);
        users[userIdx].LastIP = ip;
        await storage.saveUsers(users);

        const sid = generateSessionId(); 
        await storage.saveSession(sid, { username: user.Username });
        return new Response('', { status: 302, headers: { 'Location': '/', 'Set-Cookie': setCookie('session', sid) } });
    }
    
    if (step === '2fa') {
        const username = fd.get('username');
        const code = fd.get('code');
        
        if (await verify2FACode(storage, username, code)) {
            const users = await storage.getUsers(); 
            const ip = request.headers.get('CF-Connecting-IP') || '未知 IP';
            const userIdx = users.findIndex(x => x.Username === username);
            if (userIdx !== -1) {
                users[userIdx].LastIP = ip;
                await storage.saveUsers(users);
            }
            const sid = generateSessionId(); 
            await storage.saveSession(sid, { username });
            return new Response('', { status: 302, headers: { 'Location': '/', 'Set-Cookie': setCookie('session', sid) } });
        } else {
            return new Response(renderTemplate({ Page: 'login', User: { Language: 'zh' }, T: LangMap.zh, LoginStep: '2fa', Username: username, Message: '验证码错误或已过期' }), { headers: { 'Content-Type': 'text/html' } });
        }
    }
}

async function handleRegister(request, storage) {
    if (request.method === 'GET') {
        return new Response(renderTemplate({ Page: 'register', User: { Language: 'zh' }, T: LangMap.zh }), { headers: { 'Content-Type': 'text/html' } });
    }
    const fd = await request.formData(); 
    const u = fd.get('username'); 
    const users = await storage.getUsers();
    if (users.find(x => x.Username === u)) {
        return new Response(renderTemplate({ Page: 'register', User: { Language: 'zh' }, T: LangMap.zh, Message: 'Msg_UserExists' }), { headers: { 'Content-Type': 'text/html' } });
    }
    users.push({ 
        ID: users.length > 0 ? Math.max(...users.map(x => x.ID)) + 1 : 1, 
        Username: u, 
        Password: await hashPassword(fd.get('password')), 
        Role: users.length === 0 ? 'admin' : 'user', 
        Language: 'zh', 
        Timezone: 'Asia/Shanghai', 
        NotifyTime: 9,
        BaseCurrency: 'CNY' // 新用户默认本位币
    });
    await storage.saveUsers(users); 
    return Response.redirect(new URL('/login?msg=Msg_RegSuccess', request.url).toString(), 302);
}

async function handleSetLang(request, storage) {
    const fd = await request.formData(); 
    const lang = fd.get('lang');
    const page = fd.get('page');
    const user = await getAuthUser(request, storage); 
    if (user) { 
        user.Language = lang; 
        const users = await storage.getUsers(); 
        const idx = users.findIndex(u => u.ID === user.ID); 
        users[idx] = user; 
        await storage.saveUsers(users); 
    }
    let targetPath = '/';
    if (page === 'login' || page === 'register') { targetPath = '/' + page; }
    return new Response('', { status: 302, headers: { 'Location': new URL(targetPath, request.url).toString(), 'Set-Cookie': setCookie('lang', lang, 31536000) } });
}

// -- App Logic --
async function handleHome(request, storage, user) {
    const items = await storage.getItems(user.ID); 
    const now = new Date(); 
    items.sort((a,b) => a.Date.localeCompare(b.Date));
    
    // 获取用户本位币信息，如果没设置默认走 CNY
    const baseCurStr = user.BaseCurrency || 'CNY';
    const baseCurInfo = Currencies[baseCurStr] || Currencies["CNY"];
    
    const stats = { Total: items.length, Expiring: 0, Urgent: 0, ProjectedCost: 0 };
    
    items.forEach(i => { 
        // 兼容旧数据（旧数据没有 Currency 字段默认视为 CNY）
        i.Currency = i.Currency || 'CNY';
        i.DisplaySymbol = Currencies[i.Currency]?.Symbol || '¥';
        
        const diffDays = (new Date(i.Date) - now) / 86400000; 
        if (diffDays <= 30 && diffDays >= 0) { 
            stats.Expiring++; 
            
            // 汇率转换计算：先转成统一的基准单位（如 CNY），再转成用户的本位币
            const itemCost = parseFloat(i.Cost || 0);
            const itemRate = Currencies[i.Currency]?.Rate || 1;
            const targetRate = baseCurInfo.Rate;
            stats.ProjectedCost += itemCost * (itemRate / targetRate); 
        }
        if (diffDays <= 7 && diffDays >= 0) stats.Urgent++; 
    });
    
    stats.ProjectedCost = stats.ProjectedCost.toFixed(2);
    
    const data = { 
        Page: 'home', 
        User: user, 
        T: LangMap[user.Language || 'zh'], 
        Stats: stats, 
        Items: items, 
        UserCats: [...new Set(items.map(i => i.Category))],
        CurrencyKeys: Object.keys(Currencies),
        CurrencyMap: Currencies,
        BaseCurSymbol: baseCurInfo.Symbol,
        BaseCurName: baseCurStr,
        Message: new URL(request.url).searchParams.get('msg') 
    };
    return new Response(renderTemplate(data), { headers: { 'Content-Type': 'text/html' } });
}

async function handleAddItem(request, storage, user) {
    const fd = await request.formData(); 
    const items = await storage.getItems(user.ID);
    items.push({ 
        ID: items.length > 0 ? Math.max(...items.map(i => i.ID)) + 1 : 1, 
        Category: fd.get('category'), 
        Name: fd.get('name'), 
        Date: fd.get('date'), 
        Currency: fd.get('currency') || 'CNY',
        Cost: fd.get('cost') || '0', 
        Link: fd.get('link') || '', 
        Detail: fd.get('detail') || '' 
    });
    await storage.saveItems(user.ID, items); 
    return Response.redirect(new URL('/?msg=Msg_Add', request.url).toString(), 302);
}

async function handleUpdateItem(request, storage, user) {
    const fd = await request.formData(); 
    const items = await storage.getItems(user.ID); 
    const id = parseInt(fd.get('id')); 
    const idx = items.findIndex(i => i.ID === id);
    if (idx !== -1) { 
        items[idx] = { 
            ...items[idx], 
            Category: fd.get('category'), 
            Name: fd.get('name'), 
            Date: fd.get('date'), 
            Currency: fd.get('currency') || 'CNY',
            Cost: fd.get('cost') || '0', 
            Link: fd.get('link') || '', 
            Detail: fd.get('detail') || '' 
        }; 
        await storage.saveItems(user.ID, items); 
    }
    return Response.redirect(new URL('/?msg=Msg_Upd', request.url).toString(), 302);
}

async function handleDeleteItem(request, storage, user) {
    const items = await storage.getItems(user.ID); 
    const fd = await request.formData();
    const idToDelete = parseInt(fd.get('id'));
    await storage.saveItems(user.ID, items.filter(i => i.ID !== idToDelete));
    return Response.redirect(new URL('/?msg=Msg_Del', request.url).toString(), 302);
}

// -- Profile & Settings --
async function handleProfile(request, storage, user) { 
    const settings = await storage.getSettings();
    const data = { 
        Page: 'profile', 
        User: user, 
        Settings: settings,
        T: LangMap[user.Language || 'zh'], 
        Timezones: CommonTimezones, 
        Hours: Array.from({length:24},(_,i)=>i), 
        CurrencyKeys: Object.keys(Currencies),
        CurrencyMap: Currencies,
        Message: new URL(request.url).searchParams.get('msg') 
    };
    return new Response(renderTemplate(data), { headers: { 'Content-Type': 'text/html' } }); 
}

async function handleProfileUpdate(request, storage, user) {
    const fd = await request.formData(); 
    const users = await storage.getUsers(); 
    const u = users.find(x => x.ID === user.ID);
    if (u) { 
        u.ChatID = fd.get('chat_id'); 
        u.Email = fd.get('email'); 
        u.Timezone = fd.get('timezone'); 
        u.NotifyTime = parseInt(fd.get('notify_time')); 
        u.BaseCurrency = fd.get('base_currency') || 'CNY';
        await storage.saveUsers(users); 
    }
    return Response.redirect(new URL('/profile?msg=Msg_Saved', request.url).toString(), 302);
}

async function handleAdmin(request, storage, user) { 
    const settings = await storage.getSettings();
    const data = { Page: 'admin', User: user, T: LangMap[user.Language || 'zh'], Settings: settings, Message: new URL(request.url).searchParams.get('msg') };
    return new Response(renderTemplate(data), { headers: { 'Content-Type': 'text/html' } }); 
}

async function handleAdminUpdate(request, storage, user) {
    const settings = await storage.getSettings(); 
    const fd = await request.formData();
    settings.tg_token = fd.get('tg_token') || ''; 
    settings.tg_bot_username = fd.get('tg_bot_username') || '';
    await storage.saveSettings(settings);
    return Response.redirect(new URL('/admin?msg=Msg_Saved', request.url).toString(), 302);
}

async function handleAdminSetWebhook(request, storage) {
    const settings = await storage.getSettings();
    const token = settings.tg_token;
    if(token) {
        const originUrl = new URL(request.url).origin;
        await fetch(`https://api.telegram.org/bot${token}/setWebhook?url=${originUrl}/telegram-webhook`);
    }
    return Response.redirect(new URL('/admin?msg=Msg_WebhookSet', request.url).toString(), 302);
}

async function handleAdminUsers(request, storage, user) { 
    const users = await storage.getUsers(); 
    for (let u of users) u.Items = await storage.getItems(u.ID); 
    const data = { Page: 'users', User: user, T: LangMap[user.Language || 'zh'], Users: users };
    return new Response(renderTemplate(data), { headers: { 'Content-Type': 'text/html' } }); 
}

async function handleAdminResetPwd(request, storage, adminUser) {
    const fd = await request.formData();
    const targetId = parseInt(fd.get('id'));
    const users = await storage.getUsers();
    const targetIdx = users.findIndex(u => u.ID === targetId);
    if (targetIdx !== -1) {
        users[targetIdx].Password = await hashPassword('123456');
        await storage.saveUsers(users);
    }
    return Response.redirect(new URL('/admin/users?msg=密码已成功重置为 123456', request.url).toString(), 302);
}

async function handleAdminDelUser(request, storage, adminUser) {
    const fd = await request.formData();
    const targetId = parseInt(fd.get('id'));
    let users = await storage.getUsers();
    users = users.filter(u => u.ID !== targetId);
    await storage.saveUsers(users);
    await storage.delete(`items/${targetId}.json`);
    return Response.redirect(new URL('/admin/users?msg=用户及对应资产已彻底删除', request.url).toString(), 302);
}

async function handleTestNotify(request, storage, user) {
    const s = await storage.getSettings(); 
    let errorMsg = '';
    const msg = `ExpiryGuard: 这是一条测试消息！`;
    if (!s.tg_token) { errorMsg = '未配置 Telegram Bot Token'; } 
    else if (!user.ChatID) { errorMsg = '未填写 Telegram Chat ID'; } 
    else {
        try {
            await sendTelegramNotification(s.tg_token, user.ChatID, `<b>[测试]</b>\n${msg}`);
            return Response.redirect(new URL('/profile?msg=Msg_Sent', request.url).toString(), 302);
        } catch (e) { errorMsg = 'Telegram 发送失败: ' + e.message; }
    }
    return Response.redirect(new URL(`/profile?msg=${encodeURIComponent(errorMsg)}`, request.url).toString(), 302);
}

async function handleAdminSimulate(request, storage, user) {
    const s = await storage.getSettings(); 
    let sent = false; 
    const msg = `• [域名] example.com (7 天后到期)\n• [服务器] HK-Node-01 (已过期!)`;
    if (s.tg_token && user.ChatID) { 
        await sendTelegramNotification(s.tg_token, user.ChatID, `⚠️ <b>模拟报警</b>\n\n${msg}`); 
        sent = true; 
    }
    if (user.Email) { 
        await sendFeishuNotification(user.Email, "模拟报警", msg); 
        sent = true; 
    }
    return Response.redirect(new URL(`/admin?msg=${sent ? 'Msg_Sent' : 'Msg_Fail'}`, request.url).toString(), 302);
}

// ===== Telegram Webhook Processor =====
async function handleTelegramWebhook(request, storage, env) {
    try {
        const body = await request.json(); 
        if (!body.message || !body.message.text) return new Response('OK');
        const chatId = body.message.chat.id.toString(); 
        const text = body.message.text.trim();
        const settings = await storage.getSettings();
        const token = settings.tg_token; 
        if (!token) return new Response('OK');
        const users = await storage.getUsers(); 
        const user = users.find(u => u.ChatID === chatId);
        
        if (!user) { 
            await sendTelegramNotification(token, chatId, "❌ 未绑定 ExpiryGuard 账号，请前往系统【个人设置】绑定此 ID。"); 
            return new Response('OK'); 
        }

        if (text === '/ping') {
            await sendTelegramNotification(token, chatId, `🏓 Pong! 系统运行正常，当前用户：<b>${user.Username}</b>`);
        } else if (text === '/list') {
            const items = await storage.getItems(user.ID); 
            const now = new Date(); 
            let txt = `📊 <b>近期资产状态</b>\n\n`;
            items.forEach(i => { 
                const d = (new Date(i.Date) - now) / 86400000; 
                if(d <= 30 && d >= 0) txt += `▪️ [${i.Category}] ${i.Name} - 剩 ${Math.ceil(d)} 天\n`; 
            });
            await sendTelegramNotification(token, chatId, txt === `📊 <b>近期资产状态</b>\n\n` ? `✅ 30天内没有即将过期的资产！` : txt);
        } else if (text.startsWith('/add ')) {
            const parts = text.split(/\s+/);
            if (parts.length >= 4) {
                const cat = parts[1], name = parts[2], date = parts[3], cost = parts[4] || "0";
                const items = await storage.getItems(user.ID);
                items.push({ 
                    ID: items.length > 0 ? Math.max(...items.map(i => i.ID)) + 1 : 1, 
                    Category: cat, 
                    Name: name, 
                    Date: date, 
                    Currency: user.BaseCurrency || 'CNY',
                    Cost: cost, 
                    Link: '', 
                    Detail: 'TG Bot录入' 
                });
                await storage.saveItems(user.ID, items);
                await sendTelegramNotification(token, chatId, `✅ 添加成功！\n名称: <b>${name}</b>\n到期日: ${date}\n成本: ${cost}`);
            } else { 
                await sendTelegramNotification(token, chatId, "⚠️ 格式错误！请使用:\n`/add 分类 名称 YYYY-MM-DD 成本`"); 
            }
        } else {
            await sendTelegramNotification(token, chatId, "💡 <b>支持的指令:</b>\n/list - 近期到期资产\n/add [分类] [名称] [YYYY-MM-DD] [金额] - 极速录入\n/ping - 测试连通性");
        }
    } catch (e) { console.error(e); }
    return new Response('OK');
}

// ===== Worker Entry (Classic Syntax) =====
addEventListener('fetch', event => {
    const env = { BUCKET: typeof BUCKET !== 'undefined' ? BUCKET : null };
    event.respondWith(handleRequest(event.request, env));
});

// ===== Cron Job Trigger (Classic Syntax) =====
addEventListener('scheduled', event => {
    const env = { BUCKET: typeof BUCKET !== 'undefined' ? BUCKET : null };
    event.waitUntil(handleScheduled(event, env));
});

async function handleScheduled(event, env) {
    const storage = new R2Storage(env.BUCKET);
    const settings = await storage.getSettings();
    const users = await storage.getUsers();
    const nowUTC = new Date();

    for (const user of users) {
        const localTimeStr = nowUTC.toLocaleString("en-US", {timeZone: user.Timezone || "Asia/Shanghai"});
        const localTime = new Date(localTimeStr);
        
        if (localTime.getHours() === user.NotifyTime) {
            const items = await storage.getItems(user.ID);
            let alerts = [];
            const todayMidnight = new Date(localTime.getFullYear(), localTime.getMonth(), localTime.getDate());

            items.forEach(it => {
                const targetDate = new Date(it.Date);
                const diffDays = Math.round((targetDate - todayMidnight) / 86400000);
                if (diffDays === 7 || diffDays === 3 || diffDays === 1 || diffDays === 0) {
                    alerts.push(`• [${it.Category}] <b>${it.Name}</b> (${diffDays} 天后到期)`);
                } else if (diffDays < 0 && diffDays >= -7) {
                    alerts.push(`• [${it.Category}] <b>${it.Name}</b> (已过期)`);
                }
            });

            if (alerts.length > 0) {
                const msgBody = `${alerts.join('\n')}\n\n<i>🛡️ ExpiryGuard System</i>`;
                if (settings.tg_token && user.ChatID) {
                    await sendTelegramNotification(settings.tg_token, user.ChatID, `⚠️ <b>续费提醒</b>\n\n${msgBody}`);
                }
                if (user.Email) {
                    await sendFeishuNotification(user.Email, "续费提醒", msgBody); 
                }
            }
        }
    }
}