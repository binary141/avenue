<template>
  <template v-if="status === 'loaded'">
    <div class="content flex flex-col gap-6">

      <!-- TOP PURPLE HEADER BAR -->
      <div class="header flex flex-row items-center px-4">
        <!-- LEFT BRANDING -->
        <div class="branding flex flex-row items-center gap-3">
          <img src="/avenue-logo.png" alt="Logo" class="logo" @click="home"/>
          <span class="avenue-text">AVENUE</span>
        </div>

        <!-- RIGHT PFP -->
        <div class="ml-auto relative">
          <img
            src="/pfp.svg"
            class="w-10 h-10 cursor-pointer"
            @click="showMenu = !showMenu"
          />

          <!-- DROPDOWN MENU -->
          <div
            v-if="showMenu"
            class="absolute right-0 mt-2 w-40 bg-white shadow-lg rounded-xl p-2 flex flex-col z-50"
          >
            <button v-if="isAdmin" class="text-left px-3 py-2 hover:bg-gray-100 rounded-lg text-black"
                    @click="goToAdmin">
              Admin
            </button>
            <button class="text-left px-3 py-2 hover:bg-gray-100 rounded-lg text-black"
                    @click="goToProfile">
              Profile Settings
            </button>
            <button class="text-left px-3 py-2 hover:bg-gray-100 rounded-lg text-red-600"
                    @click="logout">
              Logout
            </button>
          </div>
        </div>
      </div>

      <RouterView @close-menu="showMenu = false"/>
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
import AppButton from './views/components/AppButton.vue';
import SpinnerView from './views/components/SpinnerView.vue';
import { useUsersStore } from './stores/users';
import { setGlobalRequestHeader } from './utils/api';
import { useRoute, useRouter } from 'vue-router';

const usersStore = useUsersStore();
const status = ref<"loading" | "loaded" | "error">("loading");
const showMenu = ref(false);
const isAdmin = ref(false);
const router = useRouter();
const route = useRoute();

watch(
  () => route.fullPath,
  () => {
    if (usersStore.token !== null) {
      setGlobalRequestHeader("Authorization", `Token ${usersStore.token}`);
    }
    getUserAndLogin();
  }
)

onMounted(() => {
  if (usersStore.token !== null) {
    setGlobalRequestHeader("Authorization", `Token ${usersStore.token}`);
  }
  getUserAndLogin();
})

function home() {
  router.push({ path: '/', query: { folderId: '' }})
  showMenu.value = false
}

function goToAdmin() {
  router.push("/admin")
  showMenu.value = false
}

function goToProfile() {
  router.push("/profile")
  showMenu.value = false
}

async function logout() {
  const response = await usersStore.logOut();

  if (response && !response.ok && response.status != 401) {
    console.error(response)
  }

  router.push("/login")
  showMenu.value = false
}

async function getUserAndLogin() {
  if (usersStore.token) {
    const response = await usersStore.pullMe();

    if (response.ok) {
      status.value = "loaded";
      usersStore.logIn(response.body);

      let isAdminLocal = usersStore.userData.data.isAdmin;
      console.log("Admin: ", isAdminLocal)
      if (isAdminLocal !== undefined) {
        isAdmin.value = isAdminLocal;
        console.log("ADMIN: ", isAdmin.value)
      }
    } else if (response.status == 401) {
      usersStore.logOut();
      status.value = "loaded";
    } else {
      console.log(response.body)
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
