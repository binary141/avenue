<template>
  <template v-if="status === 'loaded'">
    <div class="content flex flex-col gap-6">

      <!-- TOP PURPLE HEADER BAR -->
      <div class="header flex flex-row items-center px-4">
        <div class="branding flex flex-row items-center gap-3">
          <img src="/avenue-logo.png" alt="Logo" class="logo" />
          <span class="avenue-text">AVENUE</span>
        </div>
      </div>

      <RouterView />
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
    const response = await usersStore.pullMe();

    if (response.ok) {
      status.value = "loaded";
      usersStore.logIn(response.body);
    } else {
      status.value = "error";
    }
  } else {
    status.value = "loaded";
  }

  document.documentElement.classList.remove("app-not-launched");
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Poppins:wght@600;700&display=swap');

.header {
  width: 100%;
  height: 90px;
  background-color: var(--primary);
  display: flex;
  align-items: center;
}

.logo {
  height: 75px;
  width: auto;
}

.avenue-text {
  font-size: 1.5rem;
  font-weight: 600;
  color: white;
}

.content {
  width: 100%;
  align-items: center;
}

.avenue-text {
  font-family: 'Poppins', sans-serif;
  font-size: 1.9rem;
  font-weight: 700;
  color: white;
  letter-spacing: 0.5px;
}
</style>
