<template>
  <template v-if="status === 'loaded'">
    <div class="content flex flex-col gap-6">
      <div class="header flex flex-row justify-end">

      </div>
      <RouterView></RouterView>
    </div>
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
import { setGlobalRequestHeader } from './utils/api';

const usersStore = useUsersStore();
const status = ref<"loading" | "loaded" | "error">("loading");

onMounted(() => {
  if (usersStore.token !== null) {
    setGlobalRequestHeader("Authorization", `Token ${usersStore.token}`);
  }
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
.header {
  width: 100%;
  height: 69px;
  padding-left: -12px;
  background-color: var(--primary);
}
.content {
  width: 100%;
  align-items: center;
}
</style>
