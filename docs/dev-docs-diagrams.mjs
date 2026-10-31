import { box, arrow, svg } from './dev-docs-svg.mjs'

export default {
  architecture: svg(900, 470, [
    box(60, 10, 300, 60, 'Branch tills (kashi-pos)', 'orange', 'offline-first, one key per register'),
    box(540, 10, 300, 60, 'Staff browsers', 'blue', 'Vue 3 + Vuetify web app'),
    arrow([[210, 70], [210, 120], [330, 120]], 'HTTPS + WebSocket'),
    arrow([[690, 70], [690, 120], [570, 120]], 'HTTPS + WebSocket'),
    box(250, 110, 400, 60, 'Caddy', 'gray', '/api/* to the Go API   /* to ui/dist'),
    arrow([[450, 170], [450, 210]]),
    box(200, 210, 500, 150, 'Go API (Gin) on :8090', 'green'),
    box(220, 255, 210, 90, '/branch/*', 'plain', 'till sync, commands,\napp updates, WhatsApp'),
    box(470, 255, 210, 90, 'admin API', 'plain', 'login + role checks\n/admin/ws live push'),
    arrow([[450, 360], [450, 400]]),
    box(250, 400, 400, 56, 'PostgreSQL', 'blue', 'sqlc queries, LISTEN / NOTIFY'),
    box(720, 230, 170, 100, 'background', 'plain', 'recurring expenses\ntarget series\nsecurity guard'),
    arrow([[720, 280], [700, 280]]),
  ].join('')),

  syncChanges: svg(900, 240, [
    box(10, 60, 180, 80, 'any write', 'plain', 'handler, script,\nmanual SQL'),
    box(225, 60, 180, 80, 'Postgres NOTIFY', 'blue', "channel 'kashi_sync'\n(migration 000109)"),
    box(435, 60, 180, 80, 'kashi batches', 'green', 'one sync_changed\nper short window'),
    box(700, 60, 190, 80, 'till re-fetches', 'orange', 'GET sync/changes\n?since=cursor'),
    arrow([[190, 100], [225, 100]]), arrow([[405, 100], [435, 100]]), arrow([[615, 100], [700, 100]], 'WebSocket'),
    `<text class="note" x="450" y="190">The socket only carries a pointer, never data. A duplicate or lost message costs one cheap request.</text>`,
    `<text class="note" x="450" y="214">If the listener loses its connection, tills get pull_all and catch up on everything.</text>`,
  ].join('')),

  access: svg(900, 220, [
    box(10, 50, 140, 80, 'request', 'gray'),
    box(180, 50, 170, 80, 'authMiddleware', 'blue', 'PASETO token'),
    box(380, 50, 200, 80, 'permissionMiddleware', 'green', 'routeRules: section\nGET = view, else edit'),
    box(610, 50, 140, 80, 'activity log', 'plain'),
    box(770, 50, 120, 80, 'handler', 'plain'),
    arrow([[150, 90], [180, 90]]), arrow([[350, 90], [380, 90]]), arrow([[580, 90], [610, 90]]), arrow([[750, 90], [770, 90]]),
    `<text class="note" x="450" y="175">A route not listed in routeRules is admin-only, so a forgotten route fails closed.</text>`,
    `<text class="note" x="450" y="199">The UI hides pages to match; the API is what protects the data.</text>`,
  ].join('')),

  commands: svg(900, 200, [
    box(10, 40, 180, 80, 'admin', 'blue', 'creates a command'),
    box(225, 40, 190, 80, 'pending row', 'green', 'branch_commands\n(the audit trail)'),
    box(450, 40, 190, 80, 'WebSocket nudge', 'gray', 'till fetches now'),
    box(675, 40, 215, 80, 'till runs it, acks', 'orange', 'one failure does not\nblock the others'),
    arrow([[190, 80], [225, 80]]), arrow([[415, 80], [450, 80]]), arrow([[640, 80], [675, 80]]),
    `<text class="note" x="450" y="165">A disconnected till still picks it up on its next sync tick. The type must be in allowedBranchCommandTypes.</text>`,
  ].join('')),

  release: svg(900, 200, [
    box(10, 40, 170, 80, 'make publish', 'orange', 'in kashi-pos'),
    box(210, 40, 180, 80, 'createAppRelease', 'green', 'checks the SHA-256'),
    box(420, 40, 200, 80, 'admin picks target', 'blue', 'App Updates page'),
    box(650, 40, 240, 80, 'till downloads, installs', 'plain', 'reports each step back'),
    arrow([[180, 80], [210, 80]]), arrow([[390, 80], [420, 80]]), arrow([[620, 80], [650, 80]], 'nudge'),
    `<text class="note" x="450" y="165">Kashi cannot check the signature (the key is in kashi-pos). Rolling back is the same with an older target.</text>`,
  ].join('')),

  security: svg(900, 200, [
    box(10, 40, 150, 80, 'API request', 'gray'),
    box(190, 40, 190, 80, 'banned IP?', 'red', 'or outside the allowed\ncountries (not tills)'),
    box(410, 40, 190, 80, 'count the IP', 'blue', 'flushed every few\nseconds, not per request'),
    box(630, 40, 260, 80, 'raise alerts', 'orange', 'failed logins, then a ban;\nattack-path probing; foreign visits'),
    arrow([[160, 80], [190, 80]]), arrow([[380, 80], [410, 80]], 'no'), arrow([[600, 80], [630, 80]]),
    `<text class="note" x="450" y="165">Private networks and the server itself are never refused. Tills (/branch/...) skip the country filter.</text>`,
  ].join('')),

  whatsapp: svg(900, 200, [
    box(10, 40, 160, 80, 'sale lands', 'orange', 'from a till'),
    box(200, 40, 220, 80, 'rules, top to bottom', 'green', 'first enabled rule whose\nconditions all hold decides'),
    box(450, 40, 200, 80, 'send or skip', 'blue', 'its template, else the\nSettings default'),
    box(680, 40, 210, 80, 'Meta Cloud API', 'plain', 'webhook reports delivery,\nreads and replies'),
    arrow([[170, 80], [200, 80]]), arrow([[420, 80], [450, 80]]), arrow([[650, 80], [680, 80]]),
    `<text class="note" x="450" y="165">No rule matched means the default. A condition-less skip rule at the bottom turns the default off.</text>`,
  ].join('')),

  deploy: svg(900, 260, [
    box(10, 20, 150, 70, 'make deploy-api', 'orange'),
    box(190, 20, 160, 70, 'build linux binary', 'gray', 'trimmed, stripped'),
    box(380, 20, 150, 70, 'rsync', 'gray', 'binary + migrations'),
    box(560, 20, 150, 70, 'migrate up', 'green', 'pending only'),
    box(740, 20, 150, 70, 'restart', 'blue', 'systemd kashi-api'),
    arrow([[160, 55], [190, 55]]), arrow([[350, 55], [380, 55]]), arrow([[530, 55], [560, 55]]), arrow([[710, 55], [740, 55]]),
    box(10, 150, 150, 70, 'make deploy-ui', 'orange'),
    box(190, 150, 160, 70, 'build ui/dist', 'gray', 'VITE_API_URL=/api'),
    box(380, 150, 150, 70, 'swap in', 'green', 'Caddy web root'),
    box(560, 150, 330, 70, 'open tabs reload', 'blue', 'they notice version.json changed'),
    arrow([[160, 185], [190, 185]]), arrow([[350, 185], [380, 185]]), arrow([[530, 185], [560, 185]]),
  ].join('')),
}
