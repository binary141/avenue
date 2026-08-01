<template>
  <div class="page auth-page gap-5">
    <div class="auth-card card flex flex-col w-full gap-4">
      <h1 v-if="!error" class="auth-heading">Signing in&hellip;</h1>

      <!-- No @clear handler: unlike a transient form toast, this page has no
      retry action, so the error and the link back to login must stay put
      rather than disappear on ErrorMessage's 3s auto-dismiss timer. -->
      <ErrorMessage :msg="error" />

      <p v-if="error"><RouterLink :to="{ name: 'login' }" class="text-link">Back to Login</RouterLink></p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useUsersStore } from '@/stores/users';
import ErrorMessage from './components/ErrorMessage.vue';

const usersStore = useUsersStore();
const router = useRouter();
const route = useRoute();

const error = ref<string | undefined>();

onMounted(async () => {
  const providerError = route.query.error as string | undefined;
  if (providerError) {
    error.value = providerError;
    return;
  }

  const code = route.query.code as string | undefined;
  if (!code) {
    error.value = 'Missing OAuth code.';
    return;
  }

  const response = await usersStore.exchangeOAuthCode(code);

  if (response.ok) {
    usersStore.logIn(response.body.user_data);
    router.replace({ name: 'home' });
  } else {
    error.value = response.body?.error || 'Sign in failed. Please try again.';
  }
});
</script>

<style scoped>
.auth-card {
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
