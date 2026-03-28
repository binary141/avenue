<template>
  <div class="page gap-5">
    <h1>Forgot Password</h1>

    <form @submit.prevent="handleSubmit" class="form card flex flex-col w-full gap-4">
      <p class="hint">Enter your email address and we'll send you a link to reset your password.</p>

      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" required />
      </div>

      <ErrorMessage :msg="error" @clear="error = ''" />
      <SuccessMessage :msg="success" @clear="success = ''" />

      <AppButton type="submit" :disabled="submitting">SEND RESET LINK</AppButton>
    </form>

    <p><RouterLink :to="{ name: 'login' }" class="text-link">Back to Login</RouterLink></p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import AppButton from './components/AppButton.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import SuccessMessage from './components/SuccessMessage.vue';
import api from '@/utils/api';

const email = ref('');
const error = ref('');
const success = ref('');
const submitting = ref(false);

async function handleSubmit() {
  error.value = '';
  success.value = '';
  submitting.value = true;

  const response = await api({
    url: 'forgot-password',
    method: 'POST',
    json: { email: email.value },
  });

  submitting.value = false;

  if (response.ok || response.status === 204) {
    success.value = 'If an account exists for that email, a reset link has been sent.';
    email.value = '';
  } else {
    error.value = response.body?.error || 'Something went wrong. Please try again.';
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
