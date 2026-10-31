// Config for docs/build-dev-docs.mjs in this repo. Run with `make dev-docs`.
import diagrams from './dev-docs-diagrams.mjs'

export default {
  title: 'Kashi developer docs',
  out: 'dev-docs.html',
  mdiFrom: ['../ui/package.json'],
  favicon: { icon: 'mdiStorefront', bg: '#00796b' },
  source: { file: 'DEV_DOCS.md' },
  chips: [
    ['till-sync', 'Till sync', 'mdiSync'],
    ['access-control', 'Access', 'mdiShieldLock'],
    ['recipes', 'Recipes', 'mdiRocketLaunch'],
    ['gotchas-and-workarounds', 'Gotchas', 'mdiAlertOctagon'],
  ],
  diagrams,
}
