package main

const loginHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mihomo 订阅管理登录</title>
  <style>
    body{margin:0;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f5f7fa;color:#172033;display:grid;place-items:center;min-height:100vh}
    form{width:min(360px,calc(100vw - 32px));background:#fff;border:1px solid #d8dee8;border-radius:8px;padding:24px;box-shadow:0 12px 30px rgba(15,23,42,.08)}
    h1{font-size:20px;margin:0 0 18px}
    label{display:block;font-size:13px;margin:12px 0 6px;color:#526071}
    input{box-sizing:border-box;width:100%;border:1px solid #c8d0dc;border-radius:6px;padding:10px;font-size:14px}
    button{margin-top:18px;width:100%;border:0;border-radius:6px;background:#155dfc;color:white;padding:10px 12px;font-size:14px;cursor:pointer}
  </style>
</head>
<body>
  <form method="post" action="/login">
    <h1>Mihomo 订阅管理</h1>
    <label>账号</label>
    <input name="username" autocomplete="username" required>
    <label>密码</label>
    <input name="password" type="password" autocomplete="current-password" required>
    <button type="submit">登录</button>
  </form>
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
<header><h1>Mihomo 订阅管理</h1><a href="/logout">退出</a></header>
<main>
  <section>
    <h2>订阅</h2>
    <input id="subId" type="hidden">
    <label>名称</label><input id="subName" placeholder="例如：手机订阅">
    <label>代理节点</label>
    <div id="linkRows"></div>
    <button onclick="addLinkRow('', '')">添加节点</button>
    <label>采用的自定义 rules 组</label><select id="subGroups" multiple></select>
    <div class="row" style="margin-top:12px">
      <button class="primary" onclick="saveSubscription()">保存订阅</button>
      <button onclick="resetSubscriptionForm()">新建</button>
      <button onclick="exportData()">导出数据</button>
      <input id="importFile" type="file" accept="application/json" style="display:none" onchange="importData()">
      <button onclick="document.getElementById('importFile').click()">导入数据</button>
    </div>
    <p id="subStatus" class="status"></p>
    <div id="subscriptions"></div>
  </section>
  <section>
    <h2>自定义 Rules 组</h2>
    <input id="groupId" type="hidden">
    <label>名称</label><input id="groupName" placeholder="例如：AI 服务">
    <label>Rules，每行一条 Mihomo rule</label><textarea id="groupRules" placeholder="DOMAIN-SUFFIX,openai.com,PROXY"></textarea>
    <div class="row" style="margin-top:12px">
      <button class="primary" onclick="saveRuleGroup()">保存规则组</button>
      <button onclick="resetGroupForm()">新建</button>
    </div>
    <p id="groupStatus" class="status"></p>
    <div id="ruleGroups"></div>
  </section>
</main>
<script>
let subs=[], groups=[];
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
  document.getElementById('ruleGroups').innerHTML=groups.map(g=>'<div class="item"><strong>'+escapeHtml(g.name)+'</strong><div class="muted">'+escapeHtml(g.rules).replaceAll('\n','<br>')+'</div><div class="row" style="margin-top:10px"><button onclick="editGroup('+g.id+')">编辑</button><button class="danger" onclick="deleteGroup('+g.id+')">删除</button></div></div>').join('');
}
function renderSubs(){
  document.getElementById('subscriptions').innerHTML=subs.map(s=>'<div class="item"><strong>'+escapeHtml(s.name)+'</strong><div class="muted">'+escapeHtml(s.url)+'</div><div class="muted">'+(s.links||[]).length+' 个链接</div><div class="row" style="margin-top:10px"><button data-url="'+escapeHtml(s.url)+'" onclick="copyText(this.dataset.url)">复制 URL</button><button data-url="'+escapeHtml(s.url)+'" onclick="window.open(this.dataset.url,\'_blank\')">打开</button><button onclick="editSub('+s.id+')">编辑</button><button class="danger" onclick="deleteSub('+s.id+')">删除</button></div></div>').join('');
}
function addLinkRow(name='', url=''){
  const row=document.createElement('div');
  row.className='item link-row';
  row.innerHTML='<label>节点名称</label><input class="link-name" placeholder="例如：美国 01" value="'+escapeAttr(name)+'"><label>代理链接</label><textarea class="link-url" placeholder="vless://...">'+escapeHtml(url)+'</textarea><div class="row" style="margin-top:8px"><button onclick="this.closest(\'.link-row\').remove()">删除节点</button></div>';
  linkRows.appendChild(row);
}
function collectLinks(){
  return [...document.querySelectorAll('.link-row')].map(row=>({name:row.querySelector('.link-name').value.trim(), url:row.querySelector('.link-url').value.trim()})).filter(x=>x.url);
}
async function saveSubscription(){
  const id=document.getElementById('subId').value;
  const selected=[...document.getElementById('subGroups').selectedOptions].map(o=>Number(o.value));
  const payload={name:subName.value, links:collectLinks(), rule_group_ids:selected};
  const path=id?'/api/subscriptions/'+id:'/api/subscriptions';
  await api(path,{method:id?'PUT':'POST',body:JSON.stringify(payload)});
  subStatus.textContent='已保存'; resetSubscriptionForm(); await loadAll();
}
function editSub(id){
  const s=subs.find(x=>x.id===id); if(!s)return;
  subId.value=s.id; subName.value=s.name; linkRows.innerHTML=''; (s.links||[]).forEach(link=>addLinkRow(link.name||'', link.url||''));
  [...subGroups.options].forEach(o=>o.selected=(s.rule_group_ids||[]).includes(Number(o.value)));
}
async function deleteSub(id){ if(confirm('删除这个订阅？')){ await api('/api/subscriptions/'+id,{method:'DELETE'}); await loadAll(); } }
function resetSubscriptionForm(){ subId.value=''; subName.value=''; linkRows.innerHTML=''; addLinkRow('', ''); [...subGroups.options].forEach(o=>o.selected=false); }
async function saveRuleGroup(){
  const id=groupId.value; const payload={name:groupName.value,rules:groupRules.value};
  await api(id?'/api/rule-groups/'+id:'/api/rule-groups',{method:id?'PUT':'POST',body:JSON.stringify(payload)});
  groupStatus.textContent='已保存'; resetGroupForm(); await loadAll();
}
function editGroup(id){ const g=groups.find(x=>x.id===id); if(!g)return; groupId.value=g.id; groupName.value=g.name; groupRules.value=g.rules; }
async function deleteGroup(id){ if(confirm('删除这个规则组？')){ await api('/api/rule-groups/'+id,{method:'DELETE'}); await loadAll(); } }
function resetGroupForm(){ groupId.value=''; groupName.value=''; groupRules.value=''; }
async function exportData(){ location.href='/api/export'; }
async function importData(){
  const file=importFile.files[0]; if(!file)return;
  await api('/api/import',{method:'POST',body:await file.text()});
  importFile.value=''; await loadAll();
}
async function copyText(text){ await navigator.clipboard.writeText(text); subStatus.textContent='URL 已复制'; }
function escapeHtml(s){ return String(s||'').replace(/[&<>"']/g, m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m])); }
function escapeAttr(s){ return escapeHtml(s).replaceAll('\n',' '); }
resetSubscriptionForm();
loadAll().catch(e=>alert(e.message));
</script>
</body>
</html>`
