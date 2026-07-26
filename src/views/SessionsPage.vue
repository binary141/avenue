<template>
  <div class="app-page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Sessions</h1>
    </div>

    <div v-if="loading" class="flex flex-col items-center gap-3">
      <SpinnerView />
      <p>Loading sessions…</p>
    </div>

    <ErrorMessage :msg="error" @clear="error = ''" />
    <SuccessMessage :msg="successMsg" @clear="successMsg = ''" />

    <div v-if="!loading" class="w-full flex flex-col gap-6" style="max-width: 1000px;">
      <div class="flex items-center justify-between w-full">
        <p class="text-sm" style="color: var(--text-secondary);">
          Devices and browsers currently signed in to your account.
        </p>
        <AppButton
          v-if="sessions.length > 1"
          @click="revokeOthers"
          variant="danger-subtle"
          size="sm"
          class="shrink-0"
        >
          Log Out Other Sessions
        </AppButton>
      </div>

      <div class="flex flex-col gap-3">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">💻</span>

          <span
            v-if="session.isCurrent"
            class="text-xs font-medium px-1.5 py-0.5 rounded shrink-0"
            style="background: #e8f5e9; color: #2e7d32;"
          >This device</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">
              {{ session.userAgent || 'Unknown device' }}
            </p>
            <p class="font-mono text-xs truncate" style="color: var(--text-secondary);">
              {{ session.ipAddress || 'Unknown IP' }}
            </p>
          </div>

          <div class="text-xs shrink-0 text-right" style="color: var(--text-secondary);">
            <p>Signed in {{ formatDate(session.createdAt) }}</p>
            <p>Expires {{ formatDate(session.expiresAt) }}</p>
          </div>

          <AppButton
            v-if="!session.isCurrent"
            @click="revoke(session.id)"
            variant="danger-subtle"
          size="sm"
          class="shrink-0"
          >
            Log Out
          </AppButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import api from '@/utils/api';
import AppButton from './components/AppButton.vue';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import SuccessMessage from './components/SuccessMessage.vue';

interface SessionInfo {
  id: number;
  createdAt: string;
  expiresAt: string;
  userAgent: string;
  ipAddress: string;
  isCurrent: boolean;
}

const loading = ref(true);
const error = ref('');
const successMsg = ref('');
const sessions = ref<SessionInfo[]>([]);

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' });
}

async function loadSessions() {
  loading.value = true;
  const response = await api({ url: 'v1/user/sessions', method: 'GET' });
  loading.value = false;
  if (response.ok && Array.isArray(response.body)) {
    sessions.value = response.body;
  } else {
    error.value = 'Failed to load sessions';
  }
}

async function revoke(id: number) {
  const response = await api({ url: `v1/user/sessions/${id}`, method: 'DELETE' });
  if (response.ok) {
    sessions.value = sessions.value.filter(s => s.id !== id);
    successMsg.value = 'Session logged out';
  } else {
    error.value = response.body?.error || 'Failed to log out session';
  }
}

async function revokeOthers() {
  const response = await api({ url: 'v1/user/sessions', method: 'DELETE' });
  if (response.ok) {
    sessions.value = sessions.value.filter(s => s.isCurrent);
    successMsg.value = 'Logged out all other sessions';
  } else {
    error.value = 'Failed to log out other sessions';
  }
}

onMounted(loadSessions);
</script>
