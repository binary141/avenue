<template>
  <template v-if="status === 'loaded'">
    <div class="content flex flex-col gap-6">

      <!-- TOP PURPLE HEADER BAR -->
      <div class="header flex flex-row items-center px-4 gap-3">
        <button
          v-if="isLoggedIn"
          class="mobile-menu-btn"
          aria-label="Toggle navigation"
          @click="sidebarOpen = !sidebarOpen"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
        </button>

        <!-- LEFT BRANDING -->
        <div class="branding flex flex-row items-center gap-3">
          <img :src="logo" alt="Logo" class="logo" @click="home"/>
          <span class="avenue-text">AVENUE</span>
        </div>
      </div>

      <div v-if="isLoggedIn" class="app-shell flex flex-row gap-6 w-full">
        <AppSidebar
          :open="sidebarOpen"
          :is-admin="isAdmin"
          :sharing-enabled="usersStore.fileSharingEnabled || usersStore.folderSharingEnabled"
          @close="sidebarOpen = false"
          @navigate="sidebarOpen = false"
          @logout="logout"
        />

        <div class="app-main flex-1 flex flex-col items-center">
          <RouterView @close-menu="sidebarOpen = false"/>
        </div>
      </div>

      <RouterView v-else />
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
import { onMounted, ref, watch } from 'vue';
import logo from '@/assets/avenue-logo.png'
import AppButton from './views/components/AppButton.vue';
import SpinnerView from './views/components/SpinnerView.vue';
import AppSidebar from './views/components/AppSidebar.vue';
import { useUsersStore } from './stores/users';
import { useRoute, useRouter } from 'vue-router';

const usersStore = useUsersStore();
const status = ref<"loading" | "loaded" | "error">("loading");
const sidebarOpen = ref(false);
const isAdmin = ref(false);
const isLoggedIn = ref(false);
const router = useRouter();
const route = useRoute();

watch(
  () => route.fullPath,
  () => {
    getUserAndLogin();
  }
)

onMounted(() => {
  getUserAndLogin();
})

function home() {
  router.push({ path: '/', query: { folderId: '' }})
  sidebarOpen.value = false
}

async function logout() {
  const response = await usersStore.logOut();

  if (response && !response.ok && response.status != 401) {
    console.error(response)
  }

  isLoggedIn.value = false

  router.push("/login")
  sidebarOpen.value = false
}

async function getUserAndLogin() {
  // Whether we're already logged in tells us how to react to a 401 below:
  // a fresh anonymous visit shouldn't be bounced to /login (public routes
  // like a share link must stay visible), but a session that just expired
  // mid-visit should be.
  const wasLoggedIn = usersStore.loggedIn;
  const response = await usersStore.pullMe();

  if (response.ok) {
    status.value = "loaded";
    usersStore.logIn(response.body);
    isLoggedIn.value = true

    const isAdminLocal = usersStore.userData.data.isAdmin;
    if (isAdminLocal !== undefined) {
      isAdmin.value = isAdminLocal;
    }
  } else if (response.status == 401) {
    usersStore.resetSession();
    isLoggedIn.value = false;
    status.value = "loaded";
    if (wasLoggedIn) {
      router.push({ name: "login", query: { next: route.fullPath } });
    }
  } else {
    status.value = "error";
  }

  document.documentElement.classList.remove("app-not-launched");
}
</script>

<style scoped>
.header {
  width: calc(100% + 2 * var(--page-padding));
  margin: calc(-1 * var(--page-padding)) calc(-1 * var(--page-padding)) 0;
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

.app-shell {
  align-items: flex-start;
}

.app-main {
  min-width: 0;
}

.mobile-menu-btn {
  display: none;
  width: 2.2rem;
  height: 2.2rem;
  color: white;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  flex-shrink: 0;
}

.mobile-menu-btn svg {
  width: 1.4rem;
  height: 1.4rem;
}

.mobile-menu-btn:hover {
  background-color: rgba(255, 255, 255, 0.12);
}

@media (max-width: 860px) {
  .mobile-menu-btn {
    display: inline-flex;
  }
}
</style>
