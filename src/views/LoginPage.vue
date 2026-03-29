<template>
  <div class="page gap-5">
    <h1>Login</h1>

    <form @submit.prevent="handleLogin" class="login-form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" required />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password</label>

        <div class="relative">
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            required
            class="border rounded p-2 w-full pr-10"
          />

          <button
            type="button"
            @click="showPassword = !showPassword"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400"
          >
            <span v-if="showPassword" class="text-2xl">🐵</span>
            <span v-else class="text-2xl">🙈</span>
          </button>
        </div>
      </div>


      <ErrorMessage :msg="error" @clear="error = ''" />
      <SuccessMessage :msg="success" @clear="success = ''" />

      <AppButton type="submit">LOGIN</AppButton>

      <p class="forgot-link"><RouterLink :to="{ name: 'forgot-password' }" class="text-link">Forgot your password?</RouterLink></p>
    </form>

    <p v-if="canRegister">Already have an account? <RouterLink :to="{ name: 'signup' }" class="text-link">Sign Up</RouterLink> instead.</p>
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
    usersStore.setToken(response.body.session_id);
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

.forgot-link {
  text-align: center;
  font-size: 13px;
}
</style>
