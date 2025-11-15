<template>
  <div class="page gap-5">
    <h1>Login</h1>

    <form @submit.prevent="handleLogin" class="login-form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>Username</label>
        <input v-model="username" type="text" required />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password</label>
        <input v-model="password" type="password" required />
      </div>

      <AppButton type="submit">LOGIN</AppButton>
    </form>

    <p>Already have an account? <RouterLink :to="{ name: 'signup' }" class="text-link">Sign Up</RouterLink> instead.</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users';
import { useRouter } from 'vue-router';

const usersStore = useUsersStore();
const router = useRouter();

const username = ref('')
const password = ref('')

const errors = ref<Record<string, string[]>>({});
const submitting = ref(false);

async function handleLogin() {
  errors.value = {};
  submitting.value = true;

  const response = await usersStore.logInAPI({ username: username.value, password: password.value });
  submitting.value = false;

  console.log(response)

  if (response.status === 200) {
    usersStore.setToken(response.body.token);
    usersStore.logIn(response.body.user_data);
    router.replace({ name: "home" });
  }
}
</script>

<style scoped>
.login-form {
  max-width: 500px;
}

.password-container {
  position: relative;
  width: 100%;
}

.text-link {
  font-weight: bold;
}
.text-link:hover {
  color: rgb(141, 141, 255);
}
</style>
