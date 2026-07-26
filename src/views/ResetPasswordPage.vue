<template>
  <div class="page auth-page gap-5">
    <div v-if="!token" class="form auth-card card flex flex-col w-full gap-4">
      <div class="auth-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="10" width="16" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>
      </div>
      <h1 class="auth-heading">Reset password</h1>
      <p class="auth-subtitle -mt-2">This reset link is invalid or has expired. Please request a new one.</p>
      <RouterLink :to="{ name: 'forgot-password' }" class="text-link" style="text-align: center;">Request a new reset link</RouterLink>
    </div>

    <form v-else @submit.prevent="handleSubmit" class="form auth-card card flex flex-col w-full gap-4">
      <div class="auth-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="10" width="16" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>
      </div>
      <h1 class="auth-heading">Reset password</h1>
      <p class="auth-subtitle -mt-2">Choose a new password for your account</p>

      <div class="flex flex-col gap-3">
        <label>New Password</label>
        <div class="password-field">
          <input
            v-model="newPassword"
            :type="showPassword ? 'text' : 'password'"
            required
            minlength="8"
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

      <div class="flex flex-col gap-3">
        <label>Confirm Password</label>
        <div class="password-field">
          <input
            v-model="confirmPassword"
            :type="showConfirm ? 'text' : 'password'"
            required
            minlength="8"
            class="w-full"
          />
          <button
            type="button"
            @click="showConfirm = !showConfirm"
            class="password-toggle"
          >
            <span v-if="showConfirm" class="text-lg">🐵</span>
            <span v-else class="text-lg">🙈</span>
          </button>
        </div>
      </div>

      <ErrorMessage :msg="error" @clear="error = ''" />

      <AppButton type="submit" :disabled="submitting">RESET PASSWORD</AppButton>
    </form>

    <p><RouterLink :to="{ name: 'login' }" class="text-link">Back to Login</RouterLink></p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import AppButton from './components/AppButton.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import api from '@/utils/api';

const route = useRoute();
const router = useRouter();

const token = computed(() => route.query.token as string | undefined);

const newPassword = ref('');
const confirmPassword = ref('');
const showPassword = ref(false);
const showConfirm = ref(false);
const error = ref('');
const submitting = ref(false);

async function handleSubmit() {
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'Passwords do not match.';
    return;
  }

  error.value = '';
  submitting.value = true;

  const response = await api({
    url: 'reset-password',
    method: 'POST',
    json: { token: token.value, newPassword: newPassword.value },
  });

  submitting.value = false;

  if (response.ok || response.status === 204) {
    router.replace({ name: 'login', query: { reset: '1' } });
  } else {
    error.value = response.body?.error || 'This reset link is invalid or has expired.';
  }
}
</script>

<style scoped>
.form {
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
