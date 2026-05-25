package admin

const adminHTML = `
{{define "login"}}
<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Admin go-battle-ia</title>
  <style>{{template "style"}}</style>
</head>
<body class="login-body">
  <main class="login-panel">
    <h1>Admin</h1>
    <p>Interface de gestion backend.</p>
    {{if .Error}}<div class="alert error">{{.Error}}</div>{{end}}
    <form method="post" action="/admin/login">
      <label>Login
        <input name="username" autocomplete="username" required>
      </label>
      <label>Password
        <input name="password" type="password" autocomplete="current-password" required>
      </label>
      <button type="submit">Se connecter</button>
    </form>
  </main>
</body>
</html>
{{end}}

{{define "dashboard"}}
<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Admin go-battle-ia</title>
  <style>{{template "style"}}</style>
</head>
<body>
  <header class="topbar">
    <div>
      <strong>go-battle-ia admin</strong>
      <span>Connecte: {{.AdminUsername}}</span>
    </div>
    <form method="post" action="/admin/logout">
      <button class="ghost" type="submit">Logout</button>
    </form>
  </header>

  <main class="page">
    {{if .Flash}}<div class="alert ok">{{.Flash}}</div>{{end}}
    {{if .Error}}<div class="alert error">{{.Error}}</div>{{end}}

    <section class="grid metrics">
      <article><small>DB</small><strong class="{{if .Health.DatabaseOK}}good{{else}}bad{{end}}">{{.Health.Database}}</strong></article>
      <article><small>Users</small><strong>{{.Stats.Users}}</strong></article>
      <article><small>Battles</small><strong>{{.Stats.Battles}}</strong></article>
      <article><small>Quetes Battle</small><strong>{{.Stats.BattleQuests}}</strong></article>
      <article><small>Quetes RP</small><strong>{{.Stats.RolePlayQuests}}</strong></article>
      <article><small>Lives</small><strong>{{.Stats.LiveSessions}}</strong></article>
      <article><small>Streaming</small><strong>{{.Stats.LiveStreaming}}</strong></article>
      <article><small>Ended</small><strong>{{.Stats.LiveEnded}}</strong></article>
    </section>

    <section class="panel">
      <h2>Controle Backend</h2>
      <div class="two">
        <div>
          <h3>Sante</h3>
          <dl>
            <dt>Horodatage</dt><dd>{{.Health.Now}}</dd>
            <dt>Database</dt><dd>{{.Health.Database}}</dd>
            <dt>Port</dt><dd>{{.Config.AppPort}}</dd>
            <dt>GIN_MODE</dt><dd>{{.Config.GinMode}}</dd>
          </dl>
        </div>
        <div>
          <h3>Charge</h3>
          <dl>
            <dt>APP_MAX_CONCURRENT_REQUESTS</dt><dd>{{.Config.MaxConcurrentRequests}}</dd>
            <dt>APP_QUEUE_TIMEOUT_SECONDS</dt><dd>{{.Config.QueueTimeoutSeconds}}</dd>
            <dt>APP_MAX_BODY_BYTES</dt><dd>{{.Config.MaxBodyBytes}}</dd>
            <dt>DB_MAX_OPEN_CONNS</dt><dd>{{.Config.DBMaxOpenConns}}</dd>
            <dt>DB_MAX_IDLE_CONNS</dt><dd>{{.Config.DBMaxIdleConns}}</dd>
          </dl>
        </div>
      </div>
    </section>

    <section class="panel">
      <h2>Cron Quetes IA</h2>
      <div class="cron-summary">
        <article><small>Etat</small><strong class="{{if .Cron.Enabled}}good{{else}}bad{{end}}">{{if .Cron.Enabled}}actif{{else}}inactif{{end}}</strong></article>
        <article><small>Timezone</small><strong>{{.Cron.Timezone}}</strong></article>
        <article><small>Fenetre</small><strong>{{.Cron.Window}}</strong></article>
        <article><small>Limite</small><strong>{{.Cron.Limit}}</strong></article>
        <article><small>Prochain run</small><strong>{{.Cron.NextRun}}</strong></article>
      </div>
      <table class="cron-table">
        <thead><tr><th>Job</th><th>Dernier run</th><th>Provider</th><th>Step</th><th>Status</th><th>Duree</th><th>Message</th></tr></thead>
        <tbody>
          <tr>
            <td>battle</td>
            <td>{{if .Cron.Battle.LastRunID}}{{.Cron.Battle.LastRunID}}{{else}}-{{end}}</td>
            <td>{{if .Cron.Battle.LastProvider}}{{.Cron.Battle.LastProvider}} / {{.Cron.Battle.LastModel}}{{else}}-{{end}}</td>
            <td>{{if .Cron.Battle.LastStep}}{{.Cron.Battle.LastStep}}{{else}}-{{end}}</td>
            <td><span class="status {{.Cron.Battle.LastStatus}}">{{if .Cron.Battle.LastStatus}}{{.Cron.Battle.LastStatus}}{{else}}idle{{end}}</span></td>
            <td>{{.Cron.Battle.LastDurationMS}} ms</td>
            <td class="prewrap">{{if .Cron.Battle.LastError}}{{.Cron.Battle.LastError}}{{else}}{{.Cron.Battle.LastMessage}}{{end}}</td>
          </tr>
          <tr>
            <td>roleplay</td>
            <td>{{if .Cron.RolePlay.LastRunID}}{{.Cron.RolePlay.LastRunID}}{{else}}-{{end}}</td>
            <td>{{if .Cron.RolePlay.LastProvider}}{{.Cron.RolePlay.LastProvider}} / {{.Cron.RolePlay.LastModel}}{{else}}-{{end}}</td>
            <td>{{if .Cron.RolePlay.LastStep}}{{.Cron.RolePlay.LastStep}}{{else}}-{{end}}</td>
            <td><span class="status {{.Cron.RolePlay.LastStatus}}">{{if .Cron.RolePlay.LastStatus}}{{.Cron.RolePlay.LastStatus}}{{else}}idle{{end}}</span></td>
            <td>{{.Cron.RolePlay.LastDurationMS}} ms</td>
            <td class="prewrap">{{if .Cron.RolePlay.LastError}}{{.Cron.RolePlay.LastError}}{{else}}{{.Cron.RolePlay.LastMessage}}{{end}}</td>
          </tr>
        </tbody>
      </table>
      <h3>Traces recentes</h3>
      <table class="cron-table">
        <thead><tr><th>Date</th><th>Job</th><th>Run</th><th>Provider</th><th>Step</th><th>Status</th><th>Message</th></tr></thead>
        <tbody>
        {{range .Cron.Logs}}
          <tr>
            <td>{{.CreatedAt}}</td>
            <td>{{if .Job}}{{.Job}}{{else}}-{{end}}</td>
            <td>{{if .RunID}}{{.RunID}}{{else}}-{{end}}</td>
            <td>{{if .Provider}}{{.Provider}}{{if .Model}} / {{.Model}}{{end}}{{else}}-{{end}}</td>
            <td>{{.Step}}</td>
            <td><span class="status {{.Status}}">{{.Status}}</span></td>
            <td class="prewrap">{{.Message}}</td>
          </tr>
        {{else}}
          <tr><td colspan="7">Aucune trace cron en memoire.</td></tr>
        {{end}}
        </tbody>
      </table>
    </section>

    <section class="grid forms">
      <article class="panel">
        <h2>Creer Quete Battle</h2>
        <form method="post" action="/admin/quests/battle">
          <input name="title" placeholder="Titre" required>
          <textarea name="content" placeholder="Question complete" required></textarea>
          <div class="row">
            <input name="theme" placeholder="Theme">
            <input name="level" placeholder="Niveau">
          </div>
          <div class="row">
            <input name="point" type="number" placeholder="Points">
            <input name="xp" type="number" placeholder="XP">
            <input name="coin" type="number" placeholder="Coins">
          </div>
          <div class="row">
            <input name="slug" placeholder="Slug optionnel">
            <select name="status">
              <option value="published">published</option>
              <option value="draft">draft</option>
              <option value="archived">archived</option>
            </select>
          </div>
          <textarea name="metadata" placeholder='Metadata JSON optionnel: {"tag":"fun"}'></textarea>
          <button type="submit">Sauvegarder</button>
        </form>
      </article>

      <article class="panel">
        <h2>Creer Quete RP</h2>
        <form method="post" action="/admin/quests/rp">
          <input name="title" placeholder="Titre" required>
          <textarea name="summary" placeholder="Resume court"></textarea>
          <textarea name="prompt" placeholder="Prompt complet jouable" required></textarea>
          <div class="row">
            <input name="theme" placeholder="Theme">
            <input name="level" placeholder="Niveau">
          </div>
          <div class="row">
            <input name="xp" type="number" placeholder="XP">
            <input name="coin" type="number" placeholder="Coins">
            <select name="status">
              <option value="published">published</option>
              <option value="draft">draft</option>
              <option value="archived">archived</option>
            </select>
          </div>
          <textarea name="metadata" placeholder='Metadata JSON optionnel: {"ton":"noir"}'></textarea>
          <button type="submit">Sauvegarder</button>
        </form>
      </article>
    </section>

    <section class="grid forms">
      <article class="panel">
        <h2>Generation IA Battle</h2>
        <form method="post" action="/admin/generate/battle">
          <div class="row">
            <select name="provider">
              <option value="mistral">mistral</option>
              <option value="openai">openai</option>
              <option value="openrouter">openrouter</option>
              <option value="xia">xia</option>
            </select>
            <input name="model" placeholder="Modele" required>
            <input name="count" type="number" value="10" min="1" max="20">
          </div>
          <input name="api_key" type="password" placeholder="Cle API non stockee" required>
          <button type="submit">Generer et sauvegarder</button>
        </form>
      </article>

      <article class="panel">
        <h2>Generation IA RP</h2>
        <form method="post" action="/admin/generate/rp">
          <div class="row">
            <select name="provider">
              <option value="mistral">mistral</option>
              <option value="openai">openai</option>
              <option value="openrouter">openrouter</option>
              <option value="xia">xia</option>
            </select>
            <input name="model" placeholder="Modele" required>
            <input name="count" type="number" value="10" min="1" max="20">
          </div>
          <input name="api_key" type="password" placeholder="Cle API non stockee" required>
          <button type="submit">Generer et sauvegarder</button>
        </form>
      </article>
    </section>

    <section class="panel">
      <h2>Live Sessions</h2>
      <table>
        <thead><tr><th>ID</th><th>Channel</th><th>Mode</th><th>Status</th><th>Viewers</th><th>Action</th></tr></thead>
        <tbody>
        {{range .Recent.LiveSessions}}
          <tr>
            <td>{{.Id}}</td><td>{{.ChannelKey}}</td><td>{{.Mode}}</td><td>{{.Status}}</td><td>{{.ViewerCount}}</td>
            <td>
              {{if ne .Status "ended"}}
              <form method="post" action="/admin/live/{{.Id}}/end"><button class="danger" type="submit">End</button></form>
              {{else}}-
              {{end}}
            </td>
          </tr>
        {{else}}
          <tr><td colspan="6">Aucun live.</td></tr>
        {{end}}
        </tbody>
      </table>
    </section>

    <section class="grid lists">
      <article class="panel">
        <h2>Dernieres Quetes Battle</h2>
        <ul>{{range .Recent.BattleQuests}}<li><strong>#{{.Id}}</strong> {{.Title}} <small>{{.Status}}</small></li>{{else}}<li>Aucune.</li>{{end}}</ul>
      </article>
      <article class="panel">
        <h2>Dernieres Quetes RP</h2>
        <ul>{{range .Recent.RolePlayQuests}}<li><strong>#{{.Id}}</strong> {{.Title}} <small>{{.Status}}</small></li>{{else}}<li>Aucune.</li>{{end}}</ul>
      </article>
      <article class="panel">
        <h2>Dernieres Battles</h2>
        <ul>{{range .Recent.Battles}}<li><strong>#{{.Id}}</strong> {{.Title}} <small>{{.Status}}</small></li>{{else}}<li>Aucune.</li>{{end}}</ul>
      </article>
    </section>
  </main>
</body>
</html>
{{end}}

{{define "style"}}
*{box-sizing:border-box}body{margin:0;background:#f6f7f9;color:#17202a;font:14px/1.45 Inter,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.topbar{height:60px;background:#111827;color:#fff;display:flex;align-items:center;justify-content:space-between;padding:0 24px;position:sticky;top:0;z-index:2}.topbar span{margin-left:14px;color:#cbd5e1}.page{max-width:1280px;margin:0 auto;padding:24px}.grid{display:grid;gap:16px}.metrics{grid-template-columns:repeat(8,minmax(0,1fr));margin-bottom:16px}.metrics article,.panel{background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px}.metrics small{display:block;color:#6b7280}.metrics strong{font-size:22px}.forms{grid-template-columns:repeat(2,minmax(0,1fr));margin-bottom:16px}.lists{grid-template-columns:repeat(3,minmax(0,1fr));margin-bottom:40px}.two{display:grid;grid-template-columns:1fr 1fr;gap:20px}h1,h2,h3{margin:0 0 12px}h2{font-size:18px}h3{font-size:15px;color:#374151}form{margin:0}input,textarea,select{width:100%;border:1px solid #cfd6df;border-radius:6px;padding:10px;margin-bottom:10px;background:#fff;color:#111827}textarea{min-height:88px;resize:vertical}.row{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}button{border:0;border-radius:6px;background:#2563eb;color:#fff;padding:10px 14px;cursor:pointer;font-weight:600}.ghost{background:#374151}.danger{background:#dc2626;padding:7px 10px}.alert{border-radius:8px;padding:12px 14px;margin-bottom:16px}.ok{background:#ecfdf5;color:#065f46;border:1px solid #a7f3d0}.error{background:#fef2f2;color:#991b1b;border:1px solid #fecaca}.good{color:#047857}.bad{color:#b91c1c}table{width:100%;border-collapse:collapse}th,td{text-align:left;border-bottom:1px solid #e5e7eb;padding:10px;vertical-align:middle}dl{display:grid;grid-template-columns:220px 1fr;gap:8px;margin:0}dt{color:#6b7280}dd{margin:0;font-family:ui-monospace,SFMono-Regular,Menlo,monospace}ul{margin:0;padding-left:18px}li{margin:8px 0}.cron-summary{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px;margin-bottom:14px}.cron-summary article{border:1px solid #e5e7eb;border-radius:8px;padding:12px;background:#f9fafb}.cron-summary small{display:block;color:#6b7280}.cron-summary strong{display:block;margin-top:4px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:13px;overflow-wrap:anywhere}.cron-table{margin-bottom:16px}.status{display:inline-block;border-radius:999px;background:#e5e7eb;color:#374151;padding:2px 8px;font-size:12px;font-weight:700}.status.completed,.status.acquired,.status.released,.status.running,.status.enabled{background:#dcfce7;color:#166534}.status.failed,.status.invalid,.status.release_failed{background:#fee2e2;color:#991b1b}.status.skipped{background:#fef3c7;color:#92400e}.status.started{background:#dbeafe;color:#1d4ed8}.prewrap{white-space:pre-wrap;overflow-wrap:anywhere;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}.login-body{min-height:100vh;display:grid;place-items:center;background:#111827}.login-panel{width:min(420px,calc(100vw - 32px));background:#fff;border-radius:8px;padding:28px}.login-panel h1{font-size:26px}.login-panel p{color:#6b7280;margin-top:0}@media(max-width:980px){.metrics,.forms,.lists,.two,.cron-summary{grid-template-columns:1fr}.row{grid-template-columns:1fr}.topbar{padding:0 14px}.page{padding:14px}}
{{end}}
`
