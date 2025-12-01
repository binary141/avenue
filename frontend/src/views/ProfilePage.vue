<template>
  <div class="page gap-5">
    <h1>Profile</h1>

    <form @submit.prevent="updateProfile" class="login-form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password</label>
        <input v-model="password" type="password" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password Confirmation</label>
        <input v-model="passwordConfirmation" type="password" />
      </div>

      <ErrorMessage v-if="error">{{ error }}</ErrorMessage>

      <AppButton type="submit">UPDATE</AppButton>
    </form>

    <p>Already have an account? <RouterLink :to="{ name: 'signup' }" class="text-link">Sign Up</RouterLink> instead.</p>
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
const passwordConfirmation = ref('')

const error = ref<string | undefined>();
const submitting = ref(false);

function updateProfile() {
  let payload = {};

  if (password.value) {
    if (password.value !== passwordConfirmation.value) {
      error.value = "Passwords need to match"
      submitting.value = false;
      return
    }

    let minPasswordLen = 8;
    if (password.value.length < minPasswordLen) {
      error.value = "Password need to be at least "+minPasswordLen+" characters long"
      submitting.value = false;
      return
    }

    payload.password = password.value
  }

  if (email.value) {
    payload.email = email.value
  }

  if (Object.keys(payload).length >= 1) {
    console.log("post to server");
  }

  submitting.value = false;
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
