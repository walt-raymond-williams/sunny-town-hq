<script setup>
defineProps({
  boxes: {
    type: Array,
    required: true,
  },
})
</script>

<template>
  <div class="review-grid">
    <section
      v-for="box in boxes"
      :key="box.title"
      class="review-box"
      :class="{ submitted: box.submitted }"
    >
      <h3>{{ box.title }}</h3>
      <p>{{ box.text || box.emptyText || '' }}</p>
    </section>
  </div>
</template>
