<template>
  <div class="page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Admin</h1>

      <AppButton
        @click="show = true"
        class="px-4 py-2 bg-blue-600 text-white rounded"
      >
        Create User
      </AppButton>
    </div>

    <!-- create user form -->
    <div
      v-if="show"
      class="fixed inset-0 flex items-center justify-center"
    >
      <div class="bg-white p-6 rounded-lg w-96">
        <h2 class="text-lg font-semibold mb-4 text-gray-700 ">Create User</h2>

        <form
          @submit.prevent="submitForm"
          class="space-y-4 w-80 bg-white text-gray-700 p-6 rounded-lg shadow"
        >
          <div>
            <label class="block text-sm font-medium mb-1 text-gray-600">First Name</label>
            <input
              v-model="firstName"
              type="text"
              autocomplete="off"
              class="w-full border border-gray-300 rounded px-3 py-2 text-gray-700 bg-white"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1 text-gray-600">Last Name</label>
            <input
              v-model="lastName"
              type="text"
              autocomplete="off"
              class="w-full border border-gray-300 rounded px-3 py-2 text-gray-700 bg-white"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1 text-gray-600">Email</label>
            <input
              v-model="email"
              type="email"
              autocomplete="new-email"
              class="w-full border border-gray-300 rounded px-3 py-2 text-gray-700 bg-white"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1 text-gray-600">Password</label>
            <input
              v-model="password"
              type="password"
              autocomplete="new-password"
              class="w-full border border-gray-300 rounded px-3 py-2 text-gray-700 bg-white"
            />
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <AppButton
              @click="show = false"
              class="px-3 py-2 bg-gray-200 text-gray-700 rounded"
            >
              Cancel
            </AppButton>

            <AppButton
              type="submit"
              @click="createUser"
              class="px-3 py-2 bg-blue-600 text-white rounded"
            >
              Save
            </AppButton>
          </div>
        </form>


      </div>
    </div>

    <h1>Users</h1>

    <table class="min-w-full border border-gray-300 rounded-lg overflow-hidden">
      <thead class="bg-gray-100 border-b border-gray-300">
        <tr>
          <th class="px-4 py-2 text-left font-semibold text-gray-700 border-r">ID</th>
          <th class="px-4 py-2 text-left font-semibold text-gray-700 border-r">First Name</th>
          <th class="px-4 py-2 text-left font-semibold text-gray-700 border-r">Last Name</th>
          <th class="px-4 py-2 text-left font-semibold text-gray-700">Email</th>
        </tr>
      </thead>

      <tbody>
        <tr
          v-for="user in usersList"
          :key="user.id"
          class="odd:bg-white even:bg-gray-50 border-b border-gray-200"
        >
          <td class="px-4 py-2 border-r text-gray-700">{{ user.id }}</td>
          <td class="px-4 py-2 border-r text-gray-700">{{ user.firstName }}</td>
          <td class="px-4 py-2 border-r text-gray-700">{{ user.lastName }}</td>
          <td class="px-4 py-2 text-gray-700">{{ user.email }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import AppButton from './components/AppButton.vue'
import { useUsersStore } from '@/stores/users';
import type { User } from '@/types/users';

const usersStore = useUsersStore();

onMounted(() => {
  getUsers();
})

const submitForm = () => {}

const usersList = ref<User[]>([]);
const show = ref(false);

const firstName = ref('');
const lastName = ref('');
const email = ref('');
const password = ref('');

async function createUser() {
  if (email.value == '') {
    return;
  }

  if (password.value == '') {
    return;
  }

  const response = await usersStore.createUser({
    email: email.value,
    password: password.value,
    firstName: firstName.value,
    lastName: lastName.value,
  });

  if (!response.ok) {
    return
  }

  firstName.value = '';
  lastName.value = '';
  email.value = '';

  show.value = false;

  usersList.value = response.body
}

async function getUsers() {
  const response = await usersStore.getUsers()

  if (!response.ok) {
    return
  }

  usersList.value = response.body
}

</script>

<style scoped>
</style>

