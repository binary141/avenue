<template>
  <div class="page gap-5">
    <h1>SIGN UP</h1>

    <form @submit.prevent="handleSignUp" class="signup-form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="text" required />
      </div>

        <div>
          <label class="text-gray-300 text-sm mb-1 block">Password</label>
          <div class="relative">
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              required
              class="border rounded p-2 w-full"
            />
            <button type="button" @click="showPassword = !showPassword" class="absolute right-2 top-2 text-gray-400">
              <span v-if="showPassword" class="text-2xl">️🐵</span>
              <span v-else class="text-2xl">🙈</span>
            </button>
          </div>
        </div>

        <div>
          <label class="text-gray-300 text-sm mb-1 block">Confirm Password</label>
          <div class="relative">
            <input
              v-model="confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              required
              class="border rounded p-2 w-full"
            />
            <button type="button" @click="showConfirmPassword = !showConfirmPassword" class="absolute right-2 top-2 text-gray-400">
              <span v-if="showConfirmPassword" class="text-2xl">🐵</span>
              <span v-else class="text-2xl">🙈</span>
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

  console.log(response);

  if (response.status === 201 || response.status === 200) {
    usersStore.setToken(response.body.session_id);
    usersStore.logIn(response.body.user_data);
    router.replace({ name: "home" });
  } else {
    error.value = response.body.error || "Sign up failed!";
  }
}

const confirmPassword = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)

function handleSubmit() {
  if (password.value !== confirmPassword.value) {
    alert("Passwords do not match!")
    return
  }
  alert(`Signed up with ${email.value}`)
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
