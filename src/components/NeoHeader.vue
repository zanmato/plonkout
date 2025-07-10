<template>
  <header
    class="dark:bg-header dark:text-white bg-white border-b-4 border-black flex gap-8 items-center justify-between p-4 safe-area-top"
  >
    <!-- Left Section -->
    <div class="flex flex-1 gap-3 items-center justify-start">
      <slot name="left" />
    </div>

    <!-- Middle Section -->
    <div class="flex-1 flex items-center justify-center">
      <slot name="middle">
        <h1>{{ title }}</h1>
      </slot>
    </div>

    <!-- Right Section -->
    <div class="flex flex-1 gap-3 items-center justify-end">
      <slot name="right" />
    </div>
  </header>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: "",
  },
});
</script>
