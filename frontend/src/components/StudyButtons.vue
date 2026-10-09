<template>
  <div class="row wrap study-buttons">
    <RouterLink v-for="(m, i) in MODES" :key="m.mode"
                :to="{ name: 'study', params: { mode: m.mode }, query }"
                class="btn" :class="{ primary: i === 0, sm: small, disabled: count < m.minCards }">
      <Icon :name="m.icon" :size="15" /> {{ m.label }}
    </RouterLink>
  </div>
</template>

<script setup>
defineProps({
  query: { type: Object, required: true },
  count: { type: Number, required: true },
  small: Boolean,
})

const MODES = [
  { mode: 'learn', label: 'Учить', icon: 'hand', minCards: 1 },
  { mode: 'dictation', label: 'Диктант', icon: 'keyboard', minCards: 1 },
  { mode: 'match', label: 'Сопоставление', icon: 'match', minCards: 2 },
]
</script>
