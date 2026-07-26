<template>
  <div
    v-if="msg"
    class="error-message flex items-center justify-between"
  >
    <p>{{ msg }}</p>

    <button
      class="close-btn"
      aria-label="Dismiss message"
      @click="emit('clear')"
    >
      ✕
    </button>
  </div>
</template>

<script setup lang="ts">
import { watch } from "vue";

interface Props {
  msg?: string
}

const props = withDefaults(defineProps<Props>(), {
  msg: '',
})

const emit = defineEmits<{
  (e: "clear"): void
}>()

watch(
  () => props.msg,
  (newMsg) => {
    if (!newMsg) return
    setTimeout(() => emit("clear"), 3000)
  }
)
</script>

<style scoped>
.error-message {
  padding: var(--space-2) var(--space-3);
  background-color: var(--danger);
  border-radius: 8px;
  width: 100%;
  gap: var(--space-2);
  box-shadow: var(--shadow-sm);
}

.close-btn {
  background: transparent;
  border: none;
  color: white;
  font-size: 16px;
  cursor: pointer;
  line-height: 1;
  padding: 4px;
}

.close-btn:hover {
  opacity: 0.8;
}
</style>

