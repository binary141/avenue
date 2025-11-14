<template>
    <button class="app-button flex flex-row content-center align-center self-center p-5">
        <slot></slot>
    </button>
</template>

<style scoped>
.app-button {
    background-color: var(--primary);
    color: var(--text);
    border-radius: 8px;
    box-shadow: 2px 2px 4px rgba(53, 53, 53, 0.13);
}
.app-button:hover {
    background-color: var(--primary-hover);
}
.app-button:active {
    background-color: var(--primary-active);
}
</style>