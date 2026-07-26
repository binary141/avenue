<template>
  <div class="file-usage">
    <div class="file-usage__row">
      <span>{{ humanUsed }} used</span>
      <span>{{ isUnlimited ? 'Unlimited' : `${percentage}%` }}</span>
    </div>

    <div v-if="isUnlimited" class="bar bar--unlimited" />
    <div v-else class="bar">
      <div
        class="bar-fill"
        :style="{ width: percentage + '%' }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  used: number;   // bytes
  quota: number; // bytes, 0 means unlimited
}

const props = defineProps<Props>();

const isUnlimited = computed(() => props.quota === 0);

const percentage = computed(() => {
  if (isUnlimited.value) return 0;
  return Math.min(
    100,
    Math.round((props.used / props.quota) * 100)
  );
});

function bytesToHuman(bytes: number): string {
  if (bytes === 0) return "0 B";

  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);

  return `${value.toFixed(2)} ${units[i]}`;
}

const humanUsed = computed(() => bytesToHuman(props.used));
</script>

<style scoped>
.file-usage {
  width: 100%;
}

.file-usage__row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.bar {
  height: 6px;
  border-radius: 4px;
  background-color: var(--gray-4);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: 4px;
  background: linear-gradient(90deg, var(--accent), var(--accent-hover));
  transition: width 0.3s ease;
}

.bar--unlimited {
  height: 100%;
  background-image: repeating-linear-gradient(
    135deg,
    var(--accent),
    var(--accent) 6px,
    var(--gray-4) 6px,
    var(--gray-4) 12px
  );
  opacity: 0.6;
}
</style>

