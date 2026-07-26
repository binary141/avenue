<template>
  <Transition name="sidebar-backdrop">
    <div
      v-if="open"
      class="sidebar-backdrop fixed inset-0 z-40"
      @click="emit('close')"
    />
  </Transition>

  <aside class="sidebar flex flex-col" :class="{ 'sidebar--open': open }">
    <nav class="sidebar-nav flex flex-col gap-1">
      <RouterLink to="/" class="sidebar-link" active-class="sidebar-link--active" @click="emit('navigate')">
        <span class="sidebar-icon" v-html="icons.drive" />
        Drive
      </RouterLink>

      <RouterLink
        v-if="sharingEnabled"
        to="/shares"
        class="sidebar-link"
        active-class="sidebar-link--active"
        @click="emit('navigate')"
      >
        <span class="sidebar-icon" v-html="icons.share" />
        Shared Links
      </RouterLink>

      <RouterLink to="/trash" class="sidebar-link" active-class="sidebar-link--active" @click="emit('navigate')">
        <span class="sidebar-icon" v-html="icons.trash" />
        Trash
      </RouterLink>

      <RouterLink
        v-if="isAdmin"
        to="/admin"
        class="sidebar-link"
        active-class="sidebar-link--active"
        @click="emit('navigate')"
      >
        <span class="sidebar-icon" v-html="icons.admin" />
        Admin
      </RouterLink>
    </nav>

    <div class="sidebar-spacer" />

    <div class="sidebar-divider" />

    <nav class="sidebar-nav flex flex-col gap-1">
      <RouterLink to="/profile" class="sidebar-link" active-class="sidebar-link--active" @click="emit('navigate')">
        <span class="sidebar-icon" v-html="icons.user" />
        Profile Settings
      </RouterLink>

      <RouterLink to="/sessions" class="sidebar-link" active-class="sidebar-link--active" @click="emit('navigate')">
        <span class="sidebar-icon" v-html="icons.sessions" />
        Sessions
      </RouterLink>

      <button class="sidebar-link sidebar-link--danger" @click="emit('logout')">
        <span class="sidebar-icon" v-html="icons.logout" />
        Logout
      </button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean
  isAdmin: boolean
  sharingEnabled: boolean
}>()

const emit = defineEmits<{ close: []; navigate: []; logout: [] }>()

const icons = {
  drive: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6z"/></svg>',
  share: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="2.5"/><circle cx="6" cy="12" r="2.5"/><circle cx="18" cy="19" r="2.5"/><path d="M8.3 10.7 15.7 6.3M8.3 13.3l7.4 4.4"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7h16"/><path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/><path d="M6 7l1 13a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-13"/></svg>',
  admin: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6l7-3z"/></svg>',
  user: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="3.2"/><path d="M5 20c1-3.5 4-5.5 7-5.5s6 2 7 5.5"/></svg>',
  sessions: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="12" rx="1.5"/><path d="M8 20h8M12 16v4"/></svg>',
  logout: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h3"/><path d="M15 8l4 4-4 4M19 12H9"/></svg>',
}
</script>

<style scoped>
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background-color: var(--gray-2);
  border-radius: 10px;
  padding: var(--space-4);
  align-self: flex-start;
  position: sticky;
  top: var(--space-4);
  box-shadow: var(--shadow-sm);
}

.sidebar-nav {
  width: 100%;
}

.sidebar-spacer {
  flex: 1;
  min-height: 12px;
}

.sidebar-divider {
  height: 1px;
  background-color: var(--gray-4);
  margin: var(--space-1) 0 var(--space-2);
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
  text-align: left;
  transition: background-color 0.15s, color 0.15s;
}

.sidebar-link:hover {
  background-color: var(--gray-4);
  color: var(--text);
}

.sidebar-link--active {
  background-color: var(--primary);
  color: var(--text);
}

.sidebar-link--danger {
  color: var(--danger);
}

.sidebar-link--danger:hover {
  background-color: var(--danger-bg);
  color: var(--danger);
}

.sidebar-icon {
  display: inline-flex;
  width: 1.15em;
  height: 1.15em;
  flex-shrink: 0;
}

.sidebar-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.sidebar-backdrop {
  background-color: rgba(0, 0, 0, 0.5);
  display: none;
}

.sidebar-backdrop-enter-active,
.sidebar-backdrop-leave-active {
  transition: opacity 0.15s ease;
}
.sidebar-backdrop-enter-from,
.sidebar-backdrop-leave-to {
  opacity: 0;
}

@media (max-width: 860px) {
  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    width: 250px;
    z-index: 50;
    border-radius: 0;
    transform: translateX(-100%);
    transition: transform 0.2s ease;
    overflow-y: auto;
  }

  .sidebar--open {
    transform: translateX(0);
  }

  .sidebar-backdrop {
    display: block;
  }
}
</style>
