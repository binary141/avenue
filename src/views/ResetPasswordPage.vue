<template>
  <div class="page gap-5">
    <h1>Reset Password</h1>

    <div v-if="!token" class="form card flex flex-col w-full gap-4">
      <p class="hint">This reset link is invalid or has expired. Please request a new one.</p>
      <RouterLink :to="{ name: 'forgot-password' }" class="text-link">Request a new reset link</RouterLink>
    </div>

    <form v-else @submit.prevent="handleSubmit" class="form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>New Password</label>
        <div class="relative">
          <input
            v-model="newPassword"
            :type="showPassword ? 'text' : 'password'"
            required
            minlength="8"
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

      <div class="flex flex-col gap-3">
        <label>Confirm Password</label>
        <div class="relative">
          <input
            v-model="confirmPassword"
            :type="showConfirm ? 'text' : 'password'"
            required
            minlength="8"
            class="border rounded p-2 w-full pr-10"
          />
          <button
            type="button"
            @click="showConfirm = !showConfirm"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400"
          >
            <span v-if="showConfirm" class="text-2xl">🐵</span>
            <span v-else class="text-2xl">🙈</span>
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
  max-width: 500px;
}

.hint {
  color: rgba(255, 255, 255, 0.6);
  font-size: 14px;
}

.text-link {
  font-weight: bold;
}
.text-link:hover {
  color: rgb(141, 141, 255);
}
</style>
