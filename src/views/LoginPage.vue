<template>
  <div class="page auth-page gap-5">
    <form @submit.prevent="handleLogin" class="login-form auth-card card flex flex-col w-full gap-4">
      <div class="auth-icon">
        <svg data-dc-tpl="6" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect data-dc-tpl="7" x="5" y="11" width="14" height="10" rx="2"></rect>
          <path data-dc-tpl="8" d="M8 11V7a4 4 0 0 1 8 0v4"></path>
        </svg>
      </div>
      <h1 class="auth-heading">Welcome back</h1>
      <p class="auth-subtitle -mt-2">Sign in to your Avenue account</p>

      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" required />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password</label>

        <div class="password-field">
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            required
            class="w-full"
          />

          <button
            type="button"
            @click="showPassword = !showPassword"
            class="password-toggle"
          >
            <span v-if="showPassword" class="text-lg">🐵</span>
            <span v-else class="text-lg">🙈</span>
          </button>
        </div>
      </div>


      <ErrorMessage :msg="error" @clear="error = ''" />
      <SuccessMessage :msg="success" @clear="success = ''" />

      <AppButton type="submit">LOGIN</AppButton>

      <p class="forgot-link"><RouterLink :to="{ name: 'forgot-password' }" class="text-link">Forgot your password?</RouterLink></p>
    </form>

    <p v-if="canRegister">Don't have an account? <RouterLink :to="{ name: 'signup' }" class="text-link">Sign Up</RouterLink> instead.</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users';
import { useRouter, useRoute } from 'vue-router';
import ErrorMessage from './components/ErrorMessage.vue';
import SuccessMessage from './components/SuccessMessage.vue';
import api from '@/utils/api'

const usersStore = useUsersStore();
const router = useRouter();
const route = useRoute();

const email = ref('')
const password = ref('')

const error = ref<string | undefined>();
const success = ref('');
const submitting = ref(false);
const showPassword = ref(false);
const canRegister = ref(false);

async function loginMeta() {
  const response = await api({
    url: 'loginMeta',
    method: 'GET',
  })

  if (!response.ok) {
    console.log(response)
    return
  }

  canRegister.value = response.body.registration_enabled !== "false"
}

onMounted(() => {
  loginMeta()
  if (route.query.reset === '1') {
    success.value = 'Your password has been reset. You can now log in.'
  }
})

async function handleLogin() {
  error.value = undefined;
  submitting.value = true;

  const response = await usersStore.logInAPI({ email: email.value, password: password.value });
  submitting.value = false;

  if (response.status === 200) {
    usersStore.logIn(response.body.user_data);
    const redirect = (route.query.next || route.query.redirect) as string | undefined;
    const saferedirect = redirect && !redirect.startsWith('/logout') ? redirect : undefined;
    router.replace(saferedirect || { name: "home" });
  } else {
    error.value = response.body.error;
  }
}
</script>

<style scoped>
.login-form {
  max-width: 440px;
}

.text-link {
  font-weight: bold;
  color: var(--accent);
}
.text-link:hover {
  color: var(--accent-hover);
}

.forgot-link {
  text-align: center;
  font-size: 13px;
}
</style>
