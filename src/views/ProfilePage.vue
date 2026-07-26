<template>
  <div class="app-page gap-5">
    <h1>Profile</h1>

    <form @submit.prevent="updateProfile" class="login-form card flex flex-col w-full gap-4">

      <div class="flex flex-col gap-3">
        <label>First Name</label>
        <input v-model="fName" type="text" autocomplete="off" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Last Name</label>
        <input v-model="lName" type="text" autocomplete="off" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Email</label>
        <input v-model="email" type="email" />
      </div>

      <!-- Hidden fake password field to disable autofill -->
      <input type="password" style="display:none" autocomplete="new-password" />

      <div class="flex flex-col gap-3">
        <label>Password</label>

        <div class="relative">
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
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
        <label>Password Confirmation</label>
        <input v-model="passwordConfirmation" :type="showPassword ? 'text' : 'password'" autocomplete="new-password" />
        <span v-if="passwordMismatch" class="field-error">Passwords need to match</span>
      </div>

      <div v-if="password" class="flex flex-col gap-3">
        <label>Current Password</label>

        <div class="relative">
          <input
            v-model="currentPassword"
            :type="showCurrentPassword ? 'text' : 'password'"
            autocomplete="current-password"
            class="border rounded p-2 w-full pr-10"
          />

          <button
            type="button"
            @click="showCurrentPassword = !showCurrentPassword"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400"
          >
            <span v-if="showCurrentPassword" class="text-2xl">🐵</span>
            <span v-else class="text-2xl">🙈</span>
          </button>
        </div>
      </div>

      <ErrorMessage :msg=error @clear="error = ''"/>
      <SuccessMessage :msg=successMsg @clear="successMsg = ''"/>

      <AppButton type="submit">UPDATE</AppButton>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users'
import ErrorMessage from './components/ErrorMessage.vue'
import SuccessMessage from './components/SuccessMessage.vue'

const usersStore = useUsersStore()

const originalEmail = ref(usersStore.userData.data.email)
const originalFName = ref(usersStore.userData.data.firstName)
const originalLName = ref(usersStore.userData.data.lastName)

const email = ref(originalEmail.value)
const fName = ref(originalFName.value)
const lName = ref(originalLName.value)

const successMsg = ref('')

const password = ref('')
const passwordConfirmation = ref('')
const currentPassword = ref('')

const showPassword = ref(false)
const showCurrentPassword = ref(false)

const error = ref<string | undefined>()
const submitting = ref(false)

const isEmailDirty = computed(() => email.value !== originalEmail.value)
const isFNameDirty = computed(() => fName.value !== originalFName.value)
const isLNameDirty = computed(() => lName.value !== originalLName.value)

const passwordMismatch = computed(() =>
  password.value !== '' && passwordConfirmation.value !== '' && password.value !== passwordConfirmation.value
)

async function updateProfile() {
  submitting.value = true
  error.value = undefined

  const payload: Record<string, string | null> = {}

  payload.id = `${usersStore.userData.data.id}`

  if (isFNameDirty.value) {
    payload.firstName = fName.value
  }

  if (isLNameDirty.value) {
    payload.lastName = lName.value
  }

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

    if (!currentPassword.value) {
      error.value = "Current password is required"
      submitting.value = false
      return
    }

    payload.password = password.value
    payload.currentPassword = currentPassword.value
  }

  if (Object.keys(payload).length === 0) {
    submitting.value = false
    return
  }

  const response = await usersStore.updateUser(payload)
  if (!response || !response.ok) {
    let errorVal = "An error occurred, try again later"
    if (response.body.error) {
      errorVal = response.body.error
    }

    error.value = errorVal;

    submitting.value = false
    return
  }

  successMsg.value = "Successfully updated profile!"

  password.value = ''
  passwordConfirmation.value = ''
  currentPassword.value = ''
  showPassword.value = false
  showCurrentPassword.value = false

  submitting.value = false
}
</script>

<style scoped>
.login-form {
  max-width: 500px;
}

.field-error {
  color: var(--danger);
  font-size: 0.85rem;
}
</style>

