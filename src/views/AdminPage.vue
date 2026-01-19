<template>
  <div class="page gap-5">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-2xl font-bold">Admin</h1>

      <AppButton
        @click="openCreateModal"
        class="px-4 py-2 bg-blue-600 text-grey rounded"
      >
        Create User
      </AppButton>
    </div>

    <!-- Users Table -->
    <table class="min-w-full border border-grey-300 rounded-lg overflow-hidden">
      <thead class="bg-grey-100 border-b border-grey-300">
        <tr>
          <th class="px-4 py-2 text-left">ID</th>
          <th class="px-4 py-2 text-left">First Name</th>
          <th class="px-4 py-2 text-left">Last Name</th>
          <th class="px-4 py-2 text-left">Email</th>
          <th class="px-4 py-2 text-left">Admin</th>
          <th class="px-4 py-2 text-left">Actions</th>
        </tr>
      </thead>

      <tbody>
        <tr
          v-for="user in usersList"
          :key="user.id"
          class="odd:bg-grey-100 even:bg-grey-50 border-b"
        >
          <td class="px-4 py-2">{{ user.id }}</td>
          <td class="px-4 py-2">{{ user.firstName }}</td>
          <td class="px-4 py-2">{{ user.lastName }}</td>
          <td class="px-4 py-2">{{ user.email }}</td>
          <td class="px-4 py-2">
            <span
              v-if="user.isAdmin"
              class="px-2 py-1 text-xs rounded bg-green-600 text-white"
            >
              Admin
            </span>
            <span v-else class="opacity-60 text-sm">User</span>
          </td>
          <td class="px-4 py-2">
            <AppButton
              class="px-3 py-1 bg-grey-200 rounded"
              @click="() => {$emit('close-menu'); openEditModal(user)}"
            >
              Edit
            </AppButton>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- User Modal (Create + Edit) -->
    <div
      v-if="showModal"
      class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
    >
      <div class="bg-[#3A3F78] p-6 rounded-lg w-96">
        <h2 class="text-lg font-semibold mb-4 text-grey-700">
          {{ modalMode === 'create' ? 'Create User' : 'Edit User' }}
        </h2>

        <form @submit.prevent="submitUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">First Name</label>
            <input v-model="form.firstName" type="text" class="input" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Last Name</label>
            <input v-model="form.lastName" type="text" class="input" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Email</label>
            <input
              v-model="form.email"
              type="email"
              autocomplete="new-email"
              class="input"
              required
            />
          </div>

          <!-- Admin Checkbox -->
          <div class="flex items-center gap-2">
            <input
              id="isAdmin"
              type="checkbox"
              v-model="form.isAdmin"
              class="h-4 w-4 rounded border-grey-300"
            />
            <label for="isAdmin" class="text-sm font-medium">
              Administrator
            </label>
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">
              Password
              <span v-if="modalMode === 'edit'" class="text-xs opacity-60">
                (leave blank to keep unchanged)
              </span>
            </label>
            <input
              v-model="form.password"
              type="password"
              class="input"
              autocomplete="new-password"
              :required="modalMode === 'create'"
            />
          </div>

          <ErrorMessage :msg=error @clear="error = ''"/>

          <div class="flex justify-end gap-3 pt-4">
            <AppButton
              type="button"
              class="px-3 py-2 bg-grey-200 rounded"
              @click="closeModal"
            >
              Cancel
            </AppButton>

            <AppButton
              type="submit"
              class="px-3 py-2 bg-blue-600 text-grey rounded"
            >
              Save
            </AppButton>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, reactive } from 'vue'
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users'
import ErrorMessage from './components/ErrorMessage.vue';
import type { User } from '@/types/users'

const usersStore = useUsersStore()

const emit = defineEmits(['close-menu']);

const usersList = ref<User[]>([])
const showModal = ref(false)
const modalMode = ref<'create' | 'edit'>('create')
const editingUserId = ref<number | null>(null)
const error = ref<string>('')

interface UserForm {
  firstName: string
  lastName: string
  email: string
  password: string
  isAdmin: boolean
}

const form = reactive<UserForm>({
  firstName: '',
  lastName: '',
  email: '',
  password: '',
  isAdmin: false,
})

onMounted(fetchUsers)

/* ---------------- Actions ---------------- */

function openCreateModal() {
  modalMode.value = 'create'
  editingUserId.value = null
  resetForm()
  showModal.value = true
}

function openEditModal(user: User) {
  modalMode.value = 'edit'
  editingUserId.value = user.id

  form.firstName = user.firstName ?? ''
  form.lastName = user.lastName ?? ''
  form.email = user.email
  form.isAdmin = user.isAdmin
  form.password = ''

  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

/* ---------------- API ---------------- */

async function submitUser() {
  if (modalMode.value === 'create') {
    const res = await usersStore.createUser({ ...form })
    if (!res.ok) return
  } else if (editingUserId.value !== null) {
    const req = {
      id: editingUserId.value,
      firstName: form.firstName,
      lastName: form.lastName,
      email: form.email,
      isAdmin: form.isAdmin,
      ...(form.password ? { password: form.password } : {}),
    }

    const res = await usersStore.updateUser(req)
    if (!res.ok) {
      error.value = res.body.error
      return
    }
  }

  await fetchUsers()

  closeModal()
}

async function fetchUsers() {
  const res = await usersStore.getUsers()
  if (!res.ok) return
  usersList.value = res.body
}

function resetForm() {
  form.firstName = ''
  form.lastName = ''
  form.email = ''
  form.password = ''
  form.isAdmin = false
}
</script>

<style scoped>
.input {
  width: 100%;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
}
</style>

