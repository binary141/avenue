<template>
  <template v-if="status === 'loaded'">
    <RouterView></RouterView>
  </template>

  <template v-else-if="status === 'loading'">
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
import { onMounted, ref } from 'vue';
import AppButton from './views/components/AppButton.vue';
import SpinnerView from './views/components/SpinnerView.vue';
import { useUsersStore } from './stores/users';

const usersStore = useUsersStore();
const status = ref<"loading" | "loaded" | "error">("loading");

onMounted(() => {
  getUserAndLogin();
})

async function getUserAndLogin() {
  if (usersStore.token) {
    // has a user token from previous login. Load user data
    const response = await usersStore.pullMe();

    if (response.ok) {
      status.value = "loaded";
    } else {
      status.value = "error";
    }
  } else {
    // no user token, nothing to load.
    status.value = "loaded";
  }
}
</script>

<style scoped>

</style>
