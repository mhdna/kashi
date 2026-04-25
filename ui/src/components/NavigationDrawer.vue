<template>
  <v-navigation-drawer location="left" permanent density="compact" v-model="props.showDrawer" width="240">
    <v-list density="compact" nav>
      <template v-for="item in navItems" :key="item.title">
        <!-- Group item -->
        <v-list-group v-if="item.children" :value="item.title" :prepend-icon="item.icon">
          <template v-slot:activator="{ props }">
            <v-list-item v-bind="props" :title="item.title" />
          </template>
          <v-list-item v-for="child in item.children" :key="child.title" :to="child.to" :prepend-icon="child.icon"
            :title="child.title" :value="child.title" class="mx-4 me-0" />
        </v-list-group>

        <!-- Single item -->
        <v-list-item v-else :to="item.to" :prepend-icon="item.icon" :title="item.title" :value="item.title" />
      </template>
    </v-list>

    <template v-slot:append>
      <v-list density="compact" nav>
        <div class="pa-2">
          <v-list-item v-for="item in appendItems" :key="item.title" :to="item.to" :prepend-icon="item.icon"
            :title="item.title" :value="item.title" />
        </div>
      </v-list>
    </template>
  </v-navigation-drawer>
</template>

<script setup>
const props = defineProps({
  showDrawer: { type: Boolean, required: true }
})

const navItems = [
  { title: 'Dashboard', icon: 'mdi-home', to: '/dashboard' },
  { title: 'Reports', icon: 'mdi-chart-bar', to: '/stats' },
  {
    title: 'Sales', icon: 'mdi-invoice', children: [
      { title: 'Sales Invoices', icon: 'mdi-format-list-bulleted-square', to: '/sales-invoices' },
      { title: 'Purchase Invoice', icon: 'mdi-invoice', to: '/purchase-invoice' },
      { title: 'Clients', icon: 'mdi-account', to: '/clients' },
    ]
  },
  {
    title: 'Inventory', icon: 'mdi-cart', children: [
      { title: 'Inventory', icon: 'mdi-store-outline', to: '/inventory' },
      { title: 'Inventory Analysis', icon: 'mdi-sine-wave', to: '/inventory-analysis' },
      { title: 'Assets', icon: 'mdi-hammer-wrench', to: '/assets' },
      { title: 'Transfers', icon: 'mdi-transfer', to: '/transfers' },
    ]
  },
  { title: 'Warehouses', icon: 'mdi-warehouse', to: '/warehouses' },
  {
    title: 'Products', icon: 'mdi-package-variant', children: [
      { title: 'All Products', icon: 'mdi-package-variant', to: '/products' },
      { title: 'Barcodes', icon: 'mdi-barcode', to: '/barcode-printing' },
      { title: 'Colors & Sizes', icon: 'mdi-palette', to: '/colors_sizes' },
    ]
  },
  { title: 'Discounts', icon: 'mdi-sale', to: '/discounts' },
  {
    title: 'Company Affairs', icon: 'mdi-domain', children: [
      { title: 'Users', icon: 'mdi-account-group-outline', to: '/users' },
      { title: 'Employees', icon: 'mdi-account-group', to: '/employees' },
      { title: 'Attendance', icon: 'mdi-calendar', to: '/attendance' },
    ]
  },
  {
    title: 'Accounting', icon: 'mdi-account-cash', children: [
      { title: 'Expenses', icon: 'mdi-currency-usd-off', to: '/expenses' },
      { title: 'Employees', icon: 'mdi-account-group', to: '/employees' },
      { title: 'Attendance', icon: 'mdi-calendar', to: '/attendance' },
    ]
  },
]

const appendItems = [
  { title: 'Calendar', icon: 'mdi-calendar', to: '/calendar' },
  { title: 'Storage', icon: 'mdi-google-drive', to: '/storage' },
  { title: 'About', icon: 'mdi-information', to: '/about' },
]
</script>

<style scoped>
.v-list {
  padding: 0 !important;
}

.v-list-group {
  padding: 0 !important;
}

.v-list-item {
  padding: 0;
  padding-left: 10px;
  padding-right: 10px;
}
</style>