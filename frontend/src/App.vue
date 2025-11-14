<template>
  <template v-if="siteStore.appLaunchDataStatus === 'loaded'">
    <RouterView></RouterView>
  </template>

  <template v-else-if="siteStore.appLaunchDataStatus === 'loading'">
    <SpinnerView />
  </template>

  <template v-else>
    <div class="page">
      <div class="card flex flex-col align-center gap-6">
        <p>An unexpected error occured. Please check your connection and try again later.</p>

        <AppButton>Try Again</AppButton>
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useSiteStore } from './stores/site';
import AppButton from './views/components/AppButton.vue';
import SpinnerView from './views/components/SpinnerView.vue';

const siteStore = useSiteStore();

onMounted(async () => {
  await siteStore.loadAppLaunchData();
})
</script>

<style scoped>

</style>
