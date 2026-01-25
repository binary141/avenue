<template>
  <div class="page gap-5">
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
        <input v-model="password" type="password" autocomplete="new-password" />
      </div>

      <div class="flex flex-col gap-3">
        <label>Password Confirmation</label>
        <input v-model="passwordConfirmation" type="password" autocomplete="new-password" />
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

const error = ref<string | undefined>()
const submitting = ref(false)

const isEmailDirty = computed(() => email.value !== originalEmail.value)
const isFNameDirty = computed(() => fName.value !== originalFName.value)
const isLNameDirty = computed(() => lName.value !== originalLName.value)

async function updateProfile() {
  submitting.value = true
  error.value = undefined

  const payload: Record<string, string | null> = {}

  payload.id = usersStore.userData.data.id

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

    payload.password = password.value
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

  submitting.value = false
}
</script>

<style scoped>
.login-form {
  max-width: 500px;
}
</style>

