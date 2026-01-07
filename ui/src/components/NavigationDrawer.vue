<template>
  <v-navigation-drawer location="left" permanent density="compact">
    <template v-slot:prepend>
      <v-list-item
        lines="two"
        prepend-avatar="https://randomuser.me/api/portraits/women/81.jpg"
        subtitle="Logged in"
        title="Jane Smith"
      ></v-list-item>
    </template>

    <v-divider></v-divider>

    <v-list density="compact" nav den>
      <v-list-item
        to="/"
        prepend-icon="mdi-home-city"
        title="Login"
        value="home"
      ></v-list-item>
      <v-list-item
        to="/dashboard"
        prepend-icon="mdi-account"
        title="Dashboard"
        value="account"
      ></v-list-item>
      <v-list-item
        to="/stats"
        prepend-icon="mdi-chart-bar"
        title="Statistics"
        value="stats"
      ></v-list-item>
      <v-list-item
        to="/users"
        prepend-icon="mdi-account-group-outline"
        title="Users"
        value="users"
      ></v-list-item>
      <v-list-item
        to="/purchase-invoice"
        prepend-icon="mdi-invoice"
        title="Purchase Invoice"
        value="purchase-invoice"
      ></v-list-item>
      <v-list-item
        to="/sales-invoices"
        prepend-icon="mdi-invoice"
        title="Sales Invoice"
        value="sales-invoices"
      ></v-list-item>
      <v-list-item
        to="/employees"
        prepend-icon="mdi-account-group"
        title="Employees"
        value="sales-invoices"
      ></v-list-item>
    </v-list>
  </v-navigation-drawer>
  <!-- <v-main style="height: 250px"></v-main> -->
  <!-- <v-slot /> -->
</template>
