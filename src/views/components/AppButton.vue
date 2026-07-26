<template>
    <button :type="type" class="app-button" :class="[`app-button--${variant}`, `app-button--${size}`]">
        <slot></slot>
    </button>
</template>

<script setup lang="ts">
import type { PropType } from 'vue';

defineProps({
    type: {
        type: String as PropType<"button" | "submit" | "reset" | undefined>,
        default: 'button'
    },
    variant: {
        type: String as PropType<"primary" | "secondary" | "danger" | "danger-subtle">,
        default: 'primary'
    },
    size: {
        type: String as PropType<"md" | "sm">,
        default: 'md'
    }
})
</script>

<style scoped>
.app-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: auto;
    font-weight: 600;
    border-radius: 8px;
    white-space: nowrap;
    transition: background-color 0.15s ease, transform 0.1s ease, box-shadow 0.15s ease;
}

.app-button:active {
    transform: translateY(1px);
}

.app-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
}

/* Sizes */
.app-button--md {
    padding: var(--space-3) var(--space-6);
    font-size: 15px;
}

.app-button--sm {
    padding: var(--space-2) var(--space-3);
    font-size: 13px;
    border-radius: 6px;
}

/* Variants */
.app-button--primary {
    background-color: var(--primary);
    color: var(--text);
    box-shadow: var(--shadow-sm);
}
.app-button--primary:hover {
    background-color: var(--primary-hover);
}
.app-button--primary:active {
    background-color: var(--primary-active);
}

.app-button--secondary {
    background-color: var(--gray-4);
    color: var(--text);
}
.app-button--secondary:hover {
    background-color: var(--gray-5);
}

.app-button--danger {
    background-color: var(--danger-active);
    color: var(--text);
}
.app-button--danger:hover {
    background-color: var(--danger-hover);
}

.app-button--danger-subtle {
    background-color: var(--danger-bg);
    color: var(--danger);
}
.app-button--danger-subtle:hover {
    background-color: var(--danger-bg);
    filter: brightness(1.3);
}
</style>
