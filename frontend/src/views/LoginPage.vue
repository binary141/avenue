<template>
  <div class="page">
    <h2>Login</h2>

    <form @submit.prevent="handleLogin">
      <div class="field">
        <label>Email</label>
        <input v-model="email" type="Email" required />
      </div>

      <div class="field">
        <label>Password</label>
        <input v-model="password" type="password" required />
      </div>

      <button type="submit">LOGIN</button>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const email = ref('')
const password = ref('')

function handleLogin() {
  console.log("Logging in with:", email.value, password.value)
  alert(`Email: ${email.value}\nPassword: ${password.value}`)
}
</script>

<style scoped>
.login-container {
  max-width: 350px;
  margin: auto;
  padding: 2rem;
  border: 1px solid #ccc;
  border-radius: 12px;
  font-family: sans-serif;
}

.field {
  margin-bottom: 1rem;
}

label {
  display: block;
  margin-bottom: 0.25rem;
}

input {
  width: 100%;
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid #aaa;
}

button {
  width: 100%;
  padding: 0.75rem;
  border: none;
  background: #42b883;
  color: white;
  font-weight: bold;
  border-radius: 6px;
  cursor: pointer;
}
</style>
