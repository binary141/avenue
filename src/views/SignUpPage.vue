<template>
  <div class="page auth-page gap-5">
    <form @submit.prevent="handleSignUp" class="signup-form auth-card card flex flex-col w-full gap-4">
      <div class="auth-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="3.2"/><path d="M3 20c1-3.2 3.6-5 6-5s5 1.8 6 5"/><path d="M18 8h4M20 6v4"/></svg>
      </div>
      <h1 class="auth-heading">Create an account</h1>
      <p class="auth-subtitle -mt-2">Get started with Avenue</p>

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
          <button type="button" @click="showPassword = !showPassword" class="password-toggle">
            <span v-if="showPassword" class="text-lg">🐵</span>
            <span v-else class="text-lg">🙈</span>
          </button>
        </div>
      </div>

      <div class="flex flex-col gap-3">
        <label>Confirm Password</label>
        <div class="password-field">
          <input
            v-model="confirmPassword"
            :type="showConfirmPassword ? 'text' : 'password'"
            required
            class="w-full"
          />
          <button type="button" @click="showConfirmPassword = !showConfirmPassword" class="password-toggle">
            <span v-if="showConfirmPassword" class="text-lg">🐵</span>
            <span v-else class="text-lg">🙈</span>
          </button>
        </div>
      </div>

      <ErrorMessage v-if="error">{{ error }}</ErrorMessage>

      <AppButton type="submit">SIGN UP</AppButton>
    </form>

    <p>Already have an account? <RouterLink :to="{ name: 'login' }" class="text-link">Login</RouterLink> instead.</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users';
import { useRouter } from 'vue-router';
import ErrorMessage from './components/ErrorMessage.vue';

const usersStore = useUsersStore();
const router = useRouter();

const email = ref('')
const password = ref('')

const error = ref<string | undefined>();
const submitting = ref(false);

async function handleSignUp() {
  error.value = undefined;
  submitting.value = true;

  if (password.value !== confirmPassword.value) {
    error.value = "Passwords do not match!";
    submitting.value = false;
    return;
  }

  const response = await usersStore.signUpAPI({
    email: email.value,
    password: password.value,
  });

  submitting.value = false;

  if (response.status === 201 || response.status === 200) {
    router.replace({ name: "login" });
  } else {
    error.value = response.body.error || "Sign up failed!";
  }
}

const confirmPassword = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)

</script>

<style scoped>
.signup-form {
  max-width: 440px;
}

.text-link {
  font-weight: bold;
  color: var(--accent);
}
.text-link:hover {
  color: var(--accent-hover);
}
</style>
