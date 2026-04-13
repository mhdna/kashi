<script setup lang="ts">
import Heatmap from '@/components/Stats/Heatmap.vue';
import PieChart from '@/components/Stats/PieChart.vue';

</script>

<template>
    <v-sheet color="surface-light" max-height="200" class="pa-0">
        <v-row class="pa-8">
            <v-col md="2">
                <InitialsAvatar initials="AM" />
            </v-col>
            <v-col class="pt-4">
                <div class="d-flex">
                    <div class="text-headline-large pb-2">
                        Aisha Miliki
                    </div>
                    <div class="d-flex mt-2 mx-3 px-2" style="background-color: gold; height: 25px; width: 120px;">
                        <v-icon icon="mdi-star" class="me-2" />
                        <div>
                            Gold Tier
                        </div>
                    </div>
                </div>
                <div class="d-flex">
                    <div style="width: 130px;" class="text-title-medium text-grey">
                        ID:
                    </div>
                    <div>
                        dfkj3-fjie2-jkfdi2-klio9
                    </div>
                </div>
                <div class="d-flex">
                    <div style="width: 130px;" class="text-title-medium text-grey">
                        Phone:
                    </div>
                    <div>
                        772837421
                    </div>
                </div>
                <div class="d-flex">
                    <div style="width: 130px;" class="text-title-medium text-grey">
                        First Purchase:
                    </div>
                    <div>
                        13-04-2026
                    </div>
                </div>
            </v-col>
            <v-col md="6" class="pt-4">
                <v-row>
                    <v-col>
                        <SparklineCard title="Money Spent" icon="mdi-currency-usd" unit="$" />
                        <SparklineCard title="Items Bought" icon="mdi-cart-check" unit="" />
                    </v-col>
                    <v-col>
                        <SparklineCard title="Points Remaining" icon="mdi-circle" unit="" />
                        <SparklineCard title="Points Redeemed" icon="mdi-check-circle" unit="" />
                    </v-col>
                </v-row>
            </v-col>
        </v-row>
    </v-sheet>
    <v-row>
        <v-col md="7">
            <v-card max-height="180" class="pa-4 mb-2" flat max-width="800">
                <v-card-title class="px-2 py-2 font-weight-bold">Activity Over the Year</v-card-title>
                <Heatmap />
            </v-card>
            <ServerSideTable apiURL="http://localhost:4123/invoices" rootKey="invoices" />
        </v-col>
        <v-col md="5">
            <v-card max-height="300" class="pa-4 mb-2" flat max-width="800">
                <v-card-title class="px-2 py-2 font-weight-bold">Most Bought Genre</v-card-title>
                <PieChart />
            </v-card>
            <v-card max-height="300" class="pa-4 mb-2" flat max-width="800">
                <v-card-title class="px-2 py-2 font-weight-bold">Most Bought Categories</v-card-title>
                <PieChart />
            </v-card>
        </v-col>
    </v-row>
</template>