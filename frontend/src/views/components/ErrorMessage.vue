<template>
    <div class="error-message flex flex-col align-center content-center">
        <slot></slot>
    </div>
</template>

<style scoped>
.error-message {
    padding: 8px;
    background-color: rgb(240, 87, 87);
    border-radius: 8px;
    width: 100%;
}
</style>
