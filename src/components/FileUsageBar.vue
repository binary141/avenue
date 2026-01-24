<template>
  <div class="file-usage">
    <div class="flex justify-between mb-1 text-sm">
      <span>Storage Usage</span>
      <span>{{ percentage }}%</span>
    </div>

    <div class="bar">
      <div
        class="bar-fill"
        :style="{ width: percentage + '%' }"
      />
    </div>

    <div class="mt-2 text-xs text-gray-600">
      {{ humanUsed }} used of {{ humanQuota }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  used: number;   // bytes
  quota: number; // bytes
}

const props = defineProps<Props>();

const percentage = computed(() => {
  if (props.quota === 0) return 0;
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
const humanQuota = computed(() => bytesToHuman(props.quota));
</script>

<style scoped>
.file-usage {
  width: 100%;
}

.bar {
  height: 10px;
  background-color: #e5e7eb;
  border-radius: 9999px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  background-color: #3b82f6;
  transition: width 0.3s ease;
}
</style>

