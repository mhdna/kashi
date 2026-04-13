<template>
    <SparklineCard title="Items Sold" icon="mdi-cart-check" unit="" />
    <SparklineCard title="New Clients" icon="mdi-card-account-details" unit="" />
    <SparklineCard title="Revenue" icon="mdi-currency-usd" unit="$" />
    <SparklineCard title="Expenses" icon="mdi-currency-usd-off" unit="$" />
    <SparklineCard title="Invoices" icon="mdi-receipt-text-check" unit="" />
  </div>
  </v-card>
</v-row>

<v-row>
Latest Purchase Invoices
  <ServerSideTable />
</v-row>
</template>
