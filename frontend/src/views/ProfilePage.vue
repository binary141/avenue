<template>
  <div class="page gap-5">
    <h1>Profile</h1>

    <form @submit.prevent="updateProfile" class="login-form card flex flex-col w-full gap-4">
      <div class="flex flex-col gap-3">
        <label>First Name</label>
        <input v-model="fName" type="text" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Last Name</label>
        <input v-model="lName" type="text" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password</label>
        <input v-model="password" type="password" autocomplete="new-password" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password Confirmation</label>
        <input v-model="passwordConfirmation" type="password" autocomplete="new-password" />
      </div>

      <ErrorMessage v-if="error">{{ error }}</ErrorMessage>

      <AppButton type="submit">UPDATE</AppButton>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users'
import { useRouter } from 'vue-router'
import ErrorMessage from './components/ErrorMessage.vue'

const usersStore = useUsersStore()
const router = useRouter()

// --- Pre-populate from user store ---
const originalEmail = ref(usersStore.userData.data.email)  // ← existing email from backend

// Form state
const email = ref(originalEmail.value)
const fName = ref('')
const lName = ref('')
const password = ref('')
const passwordConfirmation = ref('')

const error = ref<string | undefined>()
const submitting = ref(false)

// Track dirty state
const isEmailDirty = computed(() => email.value !== originalEmail.value)

function updateProfile() {
  submitting.value = true
  error.value = undefined

  const payload: Record<string, any> = {}

  if (isEmailDirty.value) {
    payload.email = email.value
  }

  if (password.value) {
    if (password.value !== passwordConfirmation.value) {
      error.value = "Passwords need to match"
      submitting.value = false
      return
    }

    const minPasswordLen = 8
    if (password.value.length < minPasswordLen) {
      error.value = `Password needs to be at least ${minPasswordLen} characters long`
      submitting.value = false
      return
    }

    payload.password = password.value
  }

  if (Object.keys(payload).length === 0) {
    console.log("nothing to send to server: ", payload)
    submitting.value = false
    return
  }

  console.log("POST to server:", payload)

  submitting.value = false
}
</script>

<style scoped>
.login-form {
  max-width: 500px;
}

.text-link {
  font-weight: bold;
}
.text-link:hover {
  color: rgb(141, 141, 255);
}
</style>

