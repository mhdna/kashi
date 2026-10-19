<template>
  <div class="page-root">
    <v-card flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-book-open-variant" />
        <span>User Guide</span>
      </v-card-title>
      <v-divider />

      <div class="guide d-flex ga-6 pa-5">
        <nav class="guide-toc flex-shrink-0">
          <a
            v-for="section in sections"
            :key="section.id"
            class="d-block text-body-2 py-1"
            :href="`#${section.id}`"
          >{{ section.title }}</a>
        </nav>

        <div class="guide-body flex-grow-1">
          <section v-for="section in sections" :id="section.id" :key="section.id" class="mb-8">
            <h2 class="text-h6 mb-3">{{ section.title }}</h2>
            <template v-for="(block, i) in section.blocks" :key="i">
              <h3 v-if="block.kind === 'heading'" class="text-subtitle-1 font-weight-bold mt-4 mb-1">
                {{ block.text }}
              </h3>
              <p v-else-if="block.kind === 'p'" class="text-body-2 mb-2" v-html="block.text" />
              <ol v-else-if="block.kind === 'steps'" class="text-body-2 mb-2 ps-5">
                <li v-for="(step, j) in block.items" :key="j" class="mb-1" v-html="step" />
              </ol>
              <ul v-else-if="block.kind === 'list'" class="text-body-2 mb-2 ps-5">
                <li v-for="(item, j) in block.items" :key="j" class="mb-1" v-html="item" />
              </ul>
              <v-alert
                v-else-if="block.kind === 'tip'"
                class="mb-2"
                density="compact"
                type="info"
                variant="tonal"
              >
                <span v-html="block.text" />
              </v-alert>
            </template>
          </section>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup>
  // One section per workflow, each a list of blocks: a heading, a paragraph,
  // steps, a bullet list or a tip. Text is rendered with v-html so <b> and
  // <code> work -- never put user data in here.
  const sections = [
    {
      id: 'clients-export',
      title: 'Exporting clients',
      blocks: [
        {
          kind: 'p',
          text: 'On the <b>Clients</b> page, <b>Export CSV</b> downloads the clients of the tab you are on (Retail or Wholesale) as a spreadsheet-ready file.',
        },
        {
          kind: 'steps',
          items: [
            'Open <b>Clients</b> and pick the <b>Retail</b> or <b>Wholesale</b> tab.',
            'Click <b>Export CSV</b>.',
            'Tick the <b>fields</b> you want as columns.',
            'Optionally choose <b>branches</b> and a <b>date range</b> (see below).',
            'Optionally tick <b>No duplicates</b>.',
            'Click <b>Export</b>. One file downloads as a <code>.csv</code>; several branches download as one <code>.zip</code> with a file per branch.',
          ],
        },

        { kind: 'heading', text: 'Which clients belong to a branch' },
        {
          kind: 'p',
          text: 'A client belongs to a branch if they <b>bought something there</b>. Being in a till\'s client list is not enough. Cancelled invoices don\'t count, and neither do branches or registers hidden from reports, unless you tick <b>Include cancelled invoices and hidden branches</b> in the export. Cancelled invoices then add to Total spent and Invoices like any other.',
        },
        {
          kind: 'list',
          items: [
            '<b>No branch selected:</b> one file with every client of the tab, including those who never bought anything.',
            '<b>One or more branches selected:</b> only clients who bought at those branches, one file per branch. A client who bought at two selected branches appears in both files (unless <b>No duplicates</b> is on).',
            '<b>Date range:</b> keeps only clients who bought in that period. <b>Came until</b> is inclusive.',
          ],
        },
        {
          kind: 'tip',
          text: '<b>Total spent</b> and <b>Invoices</b> only count the branches and dates you chose, not the client\'s lifetime. Purchases made centrally (not at a branch till) are only counted when no branch is selected.',
        },

        { kind: 'heading', text: 'No duplicates' },
        {
          kind: 'p',
          text: 'Turn this on when the files will be used separately, for example one campaign per branch, and nobody should be contacted twice. People are matched by <b>phone number</b>, however it is written (<code>03 123 456</code> and <code>+961 3 123 456</code> are the same person).',
        },
        {
          kind: 'list',
          items: [
            'Anyone whose number appears in <b>more than one file</b> is removed from <b>all</b> of them.',
            'Inside a single file, two clients with the same number are reduced to one row, the bigger spender.',
            'Clients with no phone number are never treated as duplicates.',
          ],
        },

        { kind: 'heading', text: 'Phone check (Lebanese)' },
        {
          kind: 'p',
          text: 'Tick <b>Phone check (Lebanese)</b> to add a column that says <b>Valid</b> or <b>Invalid</b> next to each number. Filter the column in Excel to find numbers to fix or skip.',
        },
        {
          kind: 'list',
          items: [
            'Valid <b>mobiles</b>: <code>03</code> or <code>70 / 71 / 76 / 78 / 79 / 81</code> followed by the rest of the number (7 or 8 digits without the leading 0).',
            'Valid <b>landlines</b>: an area code <code>01</code> or <code>04</code>–<code>09</code> followed by 6 digits. Landlines count as Valid.',
            'Spaces, dashes, brackets, <code>+961</code>, <code>00961</code>, the leading <code>0</code> and Arabic digits are all accepted.',
            'Anything else is <b>Invalid</b>: too short or too long, a foreign number, an unused prefix such as <code>02</code> or <code>72</code>, or letters in the number.',
          ],
        },

        { kind: 'heading', text: 'Why a branch has fewer clients than expected' },
        {
          kind: 'p',
          text: 'The export only lists clients that can be tied to an invoice. A branch will look short when:',
        },
        {
          kind: 'list',
          items: [
            'Its older sales were <b>imported from the old system</b> without client details. Those invoices have no client, so nobody can be attributed from them.',
            'Cashiers sold <b>without choosing a client</b>.',
            'The branch is <b>hidden from reports</b>, or the invoices are <b>cancelled</b> (tick <b>Include cancelled invoices and hidden branches</b> to count them).',
            'A <b>date range</b> is set and the client last bought before it.',
          ],
        },
      ],
    },
  ]
</script>

<style scoped>
.guide-toc {
  position: sticky;
  top: 16px;
  align-self: flex-start;
  min-width: 180px;
}

.guide-toc a {
  color: rgb(var(--v-theme-primary));
  text-decoration: none;
}

.guide-body {
  max-width: 760px;
  min-width: 0;
}

@media (max-width: 960px) {
  .guide {
    flex-direction: column;
  }

  .guide-toc {
    position: static;
  }
}
</style>
