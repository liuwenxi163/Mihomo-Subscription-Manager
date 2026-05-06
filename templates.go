package main

const loginHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mihomo 订阅管理登录</title>
  <style>
    body{margin:0;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f5f7fa;color:#172033;display:grid;place-items:center;min-height:100vh}
    .lang{position:fixed;top:16px;right:16px;width:auto;border:1px solid #c8d0dc;border-radius:6px;padding:7px 9px;background:#fff}
    form{width:min(360px,calc(100vw - 32px));background:#fff;border:1px solid #d8dee8;border-radius:8px;padding:24px;box-shadow:0 12px 30px rgba(15,23,42,.08)}
    h1{font-size:20px;margin:0 0 18px}
    label{display:block;font-size:13px;margin:12px 0 6px;color:#526071}
    input{box-sizing:border-box;width:100%;border:1px solid #c8d0dc;border-radius:6px;padding:10px;font-size:14px}
    button{margin-top:18px;width:100%;border:0;border-radius:6px;background:#155dfc;color:white;padding:10px 12px;font-size:14px;cursor:pointer}
  </style>
</head>
<body>
  <form method="post" action="/login">
    <select id="langSelect" class="lang" onchange="setLang(this.value)" aria-label="Language"><option value="en">English</option><option value="zh">中文</option></select>
    <h1 data-i18n="loginTitle">Mihomo Subscription Manager</h1>
    <label data-i18n="username">Username</label>
    <input name="username" autocomplete="username" required>
    <label data-i18n="password">Password</label>
    <input name="password" type="password" autocomplete="current-password" required>
    <button type="submit" data-i18n="login">Log in</button>
  </form>
  <script>
    const i18n={en:{loginTitle:'Mihomo Subscription Manager',username:'Username',password:'Password',login:'Log in'},zh:{loginTitle:'Mihomo 订阅管理',username:'账号',password:'密码',login:'登录'}};
    function defaultLang(){return (navigator.language||'').toLowerCase().startsWith('zh')?'zh':'en'}
    function setLang(lang){lang=lang==='zh'?'zh':'en';localStorage.setItem('ui_lang',lang);document.documentElement.lang=lang==='zh'?'zh-CN':'en';langSelect.value=lang;document.querySelectorAll('[data-i18n]').forEach(el=>el.textContent=i18n[lang][el.dataset.i18n]||el.textContent)}
    setLang(localStorage.getItem('ui_lang')||defaultLang());
  </script>
</body>
</html>`

const adminHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mihomo 订阅管理</title>
  <style>
    *{box-sizing:border-box} body{margin:0;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f8fb;color:#172033}
    header{height:56px;display:flex;align-items:center;justify-content:space-between;padding:0 22px;border-bottom:1px solid #d9e0ea;background:#fff}
    header h1{font-size:18px;margin:0} header a{color:#526071;text-decoration:none;font-size:14px}
    header .right{display:flex;gap:10px;align-items:center} header select{width:auto;min-height:0;padding:6px 8px}
    main{display:grid;grid-template-columns:minmax(360px,1fr) minmax(360px,1fr);gap:18px;padding:18px;max-width:1280px;margin:0 auto}
    section{background:#fff;border:1px solid #d9e0ea;border-radius:8px;padding:16px}
    h2{font-size:16px;margin:0 0 12px} h3{font-size:14px;margin:16px 0 8px;color:#334155}
    label{display:block;font-size:13px;margin:10px 0 6px;color:#526071}
    input,textarea,select{width:100%;border:1px solid #c8d0dc;border-radius:6px;padding:9px 10px;font-size:14px;background:#fff}
    textarea{min-height:118px;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;resize:vertical}
    select{min-height:92px}
    button{border:1px solid #c8d0dc;border-radius:6px;background:#fff;color:#172033;padding:8px 11px;font-size:14px;cursor:pointer}
    button.primary{background:#155dfc;border-color:#155dfc;color:#fff}
    button.danger{border-color:#d92d20;color:#b42318}
    .row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.row button{white-space:nowrap}
    .item{border:1px solid #e1e7ef;border-radius:8px;padding:12px;margin:10px 0;background:#fbfcfe}
    .item strong{display:block;margin-bottom:6px}.muted{color:#64748b;font-size:12px;word-break:break-all}.status{font-size:13px;color:#155dfc;min-height:20px}
    @media(max-width:860px){main{grid-template-columns:1fr;padding:12px} header{padding:0 14px}}
  </style>
</head>
<body>
<header><h1 data-i18n="appTitle">Mihomo Subscription Manager</h1><div class="right"><select id="langSelect" onchange="setLang(this.value)" aria-label="Language"><option value="en">English</option><option value="zh">中文</option></select><a href="/logout" data-i18n="logout">Log out</a></div></header>
<main>
  <section>
    <h2 data-i18n="subscriptions">Subscriptions</h2>
    <input id="subId" type="hidden">
    <label data-i18n="name">Name</label><input id="subName" data-i18n-placeholder="subNamePh" placeholder="Example: phone">
    <label data-i18n="proxyNodes">Proxy nodes</label>
    <div id="linkRows"></div>
    <button onclick="addLinkRow('', '')" data-i18n="addNode">Add node</button>
    <label data-i18n="selectedRules">Selected custom rule groups</label><select id="subGroups" multiple></select>
    <div class="row" style="margin-top:12px">
      <button class="primary" onclick="saveSubscription()" data-i18n="saveSubscription">Save subscription</button>
      <button onclick="resetSubscriptionForm()" data-i18n="new">New</button>
      <button onclick="exportData()" data-i18n="exportData">Export data</button>
      <input id="importFile" type="file" accept="application/json" style="display:none" onchange="importData()">
      <button onclick="document.getElementById('importFile').click()" data-i18n="importData">Import data</button>
    </div>
    <p id="subStatus" class="status"></p>
    <div id="subscriptions"></div>
  </section>
  <section>
    <h2 data-i18n="ruleGroups">Custom Rule Groups</h2>
    <input id="groupId" type="hidden">
    <label data-i18n="name">Name</label><input id="groupName" data-i18n-placeholder="groupNamePh" placeholder="Example: AI services">
    <label data-i18n="rulesLines">Rules, one Mihomo rule per line</label><textarea id="groupRules" placeholder="DOMAIN-SUFFIX,openai.com,PROXY"></textarea>
    <div class="row" style="margin-top:12px">
      <button class="primary" onclick="saveRuleGroup()" data-i18n="saveRuleGroup">Save rule group</button>
      <button onclick="resetGroupForm()" data-i18n="new">New</button>
    </div>
    <p id="groupStatus" class="status"></p>
    <div id="ruleGroups"></div>
  </section>
</main>
<script>
let subs=[], groups=[], currentLang='en';
const i18n={
  en:{appTitle:'Mihomo Subscription Manager',logout:'Log out',subscriptions:'Subscriptions',name:'Name',subNamePh:'Example: phone',proxyNodes:'Proxy nodes',addNode:'Add node',selectedRules:'Selected custom rule groups',saveSubscription:'Save subscription',new:'New',exportData:'Export data',importData:'Import data',ruleGroups:'Custom Rule Groups',groupNamePh:'Example: AI services',rulesLines:'Rules, one Mihomo rule per line',saveRuleGroup:'Save rule group',nodeName:'Node name',nodeNamePh:'Example: US 01',proxyLink:'Proxy link',deleteNode:'Delete node',linksCount:'links',copyUrl:'Copy URL',open:'Open',edit:'Edit',delete:'Delete',saved:'Saved',urlCopied:'URL copied',deleteSubConfirm:'Delete this subscription?',deleteGroupConfirm:'Delete this rule group?'},
  zh:{appTitle:'Mihomo 订阅管理',logout:'退出',subscriptions:'订阅',name:'名称',subNamePh:'例如：手机订阅',proxyNodes:'代理节点',addNode:'添加节点',selectedRules:'采用的自定义 rules 组',saveSubscription:'保存订阅',new:'新建',exportData:'导出数据',importData:'导入数据',ruleGroups:'自定义 Rules 组',groupNamePh:'例如：AI 服务',rulesLines:'Rules，每行一条 Mihomo rule',saveRuleGroup:'保存规则组',nodeName:'节点名称',nodeNamePh:'例如：美国 01',proxyLink:'代理链接',deleteNode:'删除节点',linksCount:'个链接',copyUrl:'复制 URL',open:'打开',edit:'编辑',delete:'删除',saved:'已保存',urlCopied:'URL 已复制',deleteSubConfirm:'删除这个订阅？',deleteGroupConfirm:'删除这个规则组？'}
};
function defaultLang(){return (navigator.language||'').toLowerCase().startsWith('zh')?'zh':'en'}
function t(key){return (i18n[currentLang]&&i18n[currentLang][key])||i18n.en[key]||key}
function setLang(lang){
  currentLang=lang==='zh'?'zh':'en'; localStorage.setItem('ui_lang',currentLang); document.documentElement.lang=currentLang==='zh'?'zh-CN':'en'; langSelect.value=currentLang;
  document.querySelectorAll('[data-i18n]').forEach(el=>el.textContent=t(el.dataset.i18n));
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el=>el.placeholder=t(el.dataset.i18nPlaceholder));
  renderGroups(); renderSubs();
  document.querySelectorAll('.link-row').forEach(updateLinkRowText);
}
async function api(path, options={}){
  const res = await fetch(path, {headers:{'Content-Type':'application/json'}, ...options});
  if(!res.ok){ throw new Error(await res.text()); }
  if(res.status===204) return null;
  return await res.json();
}
async function loadAll(){
  [subs, groups] = await Promise.all([api('/api/subscriptions'), api('/api/rule-groups')]);
  renderGroups(); renderSubs();
}
function renderGroups(){
  const select=document.getElementById('subGroups');
  select.innerHTML=groups.map(g=>'<option value="'+g.id+'">'+escapeHtml(g.name)+'</option>').join('');
  document.getElementById('ruleGroups').innerHTML=groups.map(g=>'<div class="item"><strong>'+escapeHtml(g.name)+'</strong><div class="muted">'+escapeHtml(g.rules).replaceAll('\n','<br>')+'</div><div class="row" style="margin-top:10px"><button onclick="editGroup('+g.id+')">'+t('edit')+'</button><button class="danger" onclick="deleteGroup('+g.id+')">'+t('delete')+'</button></div></div>').join('');
}
function renderSubs(){
  document.getElementById('subscriptions').innerHTML=subs.map(s=>'<div class="item"><strong>'+escapeHtml(s.name)+'</strong><div class="muted">'+escapeHtml(s.url)+'</div><div class="muted">'+(s.links||[]).length+' '+t('linksCount')+'</div><div class="row" style="margin-top:10px"><button data-url="'+escapeHtml(s.url)+'" onclick="copyText(this.dataset.url)">'+t('copyUrl')+'</button><button data-url="'+escapeHtml(s.url)+'" onclick="window.open(this.dataset.url,\'_blank\')">'+t('open')+'</button><button onclick="editSub('+s.id+')">'+t('edit')+'</button><button class="danger" onclick="deleteSub('+s.id+')">'+t('delete')+'</button></div></div>').join('');
}
function addLinkRow(name='', url=''){
  const row=document.createElement('div');
  row.className='item link-row';
  row.innerHTML='<label data-link-label="nodeName"></label><input class="link-name" data-link-placeholder="nodeNamePh" value="'+escapeAttr(name)+'"><label data-link-label="proxyLink"></label><textarea class="link-url" placeholder="vless://...">'+escapeHtml(url)+'</textarea><div class="row" style="margin-top:8px"><button data-link-button="deleteNode" onclick="this.closest(\'.link-row\').remove()"></button></div>';
  linkRows.appendChild(row);
  updateLinkRowText(row);
}
function updateLinkRowText(row){row.querySelectorAll('[data-link-label]').forEach(el=>el.textContent=t(el.dataset.linkLabel));row.querySelectorAll('[data-link-placeholder]').forEach(el=>el.placeholder=t(el.dataset.linkPlaceholder));row.querySelectorAll('[data-link-button]').forEach(el=>el.textContent=t(el.dataset.linkButton));}
function collectLinks(){
  return [...document.querySelectorAll('.link-row')].map(row=>({name:row.querySelector('.link-name').value.trim(), url:row.querySelector('.link-url').value.trim()})).filter(x=>x.url);
}
async function saveSubscription(){
  const id=document.getElementById('subId').value;
  const selected=[...document.getElementById('subGroups').selectedOptions].map(o=>Number(o.value));
  const payload={name:subName.value, links:collectLinks(), rule_group_ids:selected};
  const path=id?'/api/subscriptions/'+id:'/api/subscriptions';
  await api(path,{method:id?'PUT':'POST',body:JSON.stringify(payload)});
  subStatus.textContent=t('saved'); resetSubscriptionForm(); await loadAll();
}
function editSub(id){
  const s=subs.find(x=>x.id===id); if(!s)return;
  subId.value=s.id; subName.value=s.name; linkRows.innerHTML=''; (s.links||[]).forEach(link=>addLinkRow(link.name||'', link.url||''));
  [...subGroups.options].forEach(o=>o.selected=(s.rule_group_ids||[]).includes(Number(o.value)));
}
async function deleteSub(id){ if(confirm(t('deleteSubConfirm'))){ await api('/api/subscriptions/'+id,{method:'DELETE'}); await loadAll(); } }
function resetSubscriptionForm(){ subId.value=''; subName.value=''; linkRows.innerHTML=''; addLinkRow('', ''); [...subGroups.options].forEach(o=>o.selected=false); }
async function saveRuleGroup(){
  const id=groupId.value; const payload={name:groupName.value,rules:groupRules.value};
  await api(id?'/api/rule-groups/'+id:'/api/rule-groups',{method:id?'PUT':'POST',body:JSON.stringify(payload)});
  groupStatus.textContent=t('saved'); resetGroupForm(); await loadAll();
}
function editGroup(id){ const g=groups.find(x=>x.id===id); if(!g)return; groupId.value=g.id; groupName.value=g.name; groupRules.value=g.rules; }
async function deleteGroup(id){ if(confirm(t('deleteGroupConfirm'))){ await api('/api/rule-groups/'+id,{method:'DELETE'}); await loadAll(); } }
function resetGroupForm(){ groupId.value=''; groupName.value=''; groupRules.value=''; }
async function exportData(){ location.href='/api/export'; }
async function importData(){
  const file=importFile.files[0]; if(!file)return;
  await api('/api/import',{method:'POST',body:await file.text()});
  importFile.value=''; await loadAll();
}
async function copyText(text){ await navigator.clipboard.writeText(text); subStatus.textContent=t('urlCopied'); }
function escapeHtml(s){ return String(s||'').replace(/[&<>"']/g, m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m])); }
function escapeAttr(s){ return escapeHtml(s).replaceAll('\n',' '); }
resetSubscriptionForm();
setLang(localStorage.getItem('ui_lang')||defaultLang());
loadAll().catch(e=>alert(e.message));
</script>
</body>
</html>`
