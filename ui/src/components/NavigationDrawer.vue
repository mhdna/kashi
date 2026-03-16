<template>
  <v-navigation-drawer location="left" permanent density="compact" v-model="props.showDrawer">
    <template v-slot:prepend>
      <v-list-item lines="two" title="Test Person">
        <!-- subtitle="Logged in" -->
        <template v-slot:prepend>
          <v-avatar color="blue-darken-4" class="text-white" size="40">
            MN
          </v-avatar>
        </template>
      </v-list-item>
    </template>

    <v-divider></v-divider>

    <v-list density="compact" nav den>
      <!-- <v-list-item to="/" prepend-icon="mdi-home-city" title="Login" value="home"></v-list-item> -->
      <v-list-item to="/warehouses" prepend-icon="mdi-warehouse" title="Warehouses" value="Warehouses"></v-list-item>
      <v-list-item to="/dashboard" prepend-icon="mdi-account" title="Dashboard" value="account"></v-list-item>
      <v-list-item to="/stats" prepend-icon="mdi-chart-bar" title="Statistics" value="stats"></v-list-item>
      <v-list-item to="/users" prepend-icon="mdi-account-group-outline" title="Users" value="users"></v-list-item>
      <v-list-item to="/barcode-printing" prepend-icon="mdi-barcode" title="Barcode Printing"
        value="barcode-printing"></v-list-item>

      <!-- <v-list-group value="products"> -->
      <!--   <template v-slot:activator="{ props }"> -->
      <!--     <v-list-item v-bind="props" title="Products Management"></v-list-item> -->
      <!--   </template> -->

      <!--   <v-list-item v-for="([title, icon], i) in product_management_items" :key="i" :prepend-icon="icon" :title="title" -->
      <!--     :value="title"></v-list-item> -->
      <!-- </v-list-group> -->

      <v-list-item to="/inventory" prepend-icon="mdi-store-outline" title="Inventory" value="inventory"></v-list-item>
      <v-list-item to="/purchase-invoice" prepend-icon="mdi-invoice" title="Purchase Invoice"
        value="purchase-invoice"></v-list-item>
      <v-list-item to="/sales-invoices" prepend-icon="mdi-invoice" title="Sales Invoice"
        value="sales-invoices"></v-list-item>
      <v-list-item to="/employees" prepend-icon="mdi-account-group" title="Employees"
        value="sales-invoices"></v-list-item>
      <v-list-item to="/colors_sizes" prepend-icon="mdi-palette" title="Colors & Sizes"
        value="colors_sizes"></v-list-item>
      <v-list-item to="/products" prepend-icon="mdi-package-variant" title="Products" value="products"></v-list-item>
      <v-list-item to="/clients" prepend-icon="mdi-person" title="Clients" value="clients"></v-list-item>
      <v-list-item to="/transfers" prepend-icon="mdi-transfer" title="Tranfers" value="tranfers"></v-list-item>
      <v-list-item to="/attendance" prepend-icon="mdi-calendar" title="Attendance" value="Attendance"></v-list-item>
      <v-list-item to="/storage" prepend-icon="mdi-usb-flash-drive-outline" title="Storage"
        value="Storage"></v-list-item>
      <v-list-item to="/about" prepend-icon="mdi-info" title="About" value="about"></v-list-item>
      <v-list-item to="/settings" prepend-icon="mdi-geer" title="Settings" value="settings"></v-list-item>
    </v-list>
  </v-navigation-drawer>
  <!-- <v-main style="height: 250px"></v-main> -->
  <!-- <v-slot /> -->
</template>

<script setup>
const props = defineProps({
  showDrawer: {
    type: Boolean,
    required: true
  }
})

// const product_management_items = [
//   ['All Products', 'mdi-package-variant'],
//   ['Barcodes', 'mdi-barcode'],
// ]
</script>