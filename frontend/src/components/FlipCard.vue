<template>
  <div class="flip-card" :class="[`size-${size}`, { flipped: isFlipped, 'with-example': isExample }]"
       :tabindex="interactive ? 0 : -1" role="button"
       :aria-label="`Карточка: ${frontWord}. Нажмите, чтобы перевернуть; удерживайте, чтобы увидеть пример`"
       v-bind="interactive ? press.handlers : {}">
    <div class="flip-inner">
      <div v-for="face in faces" :key="face.key" class="flip-face" :class="[face.key, `lang-${face.lang}`]">
        <div class="face-top">
          <span class="lang-chip">{{ face.lang === 'ru' ? 'RU' : 'EN' }}</span>
          <div class="face-actions" data-no-press @pointerdown.stop @click.stop><slot name="actions" /></div>
        </div>
        <div class="face-main">
          <div class="face-word" :class="{ long: face.word.length > 18 }">{{ face.word }}</div>
          <Transition name="example">
            <div v-if="isExample && face.example" class="face-example">{{ face.example }}</div>
          </Transition>
          <div v-if="isExample && !face.example" class="face-example empty">Пример не добавлен</div>
        </div>
        <div class="face-hint">
          <template v-if="face.example && !isExample"><Icon name="hand" :size="13" /> удерживайте — пример</template>
          <template v-else-if="isExample">удерживайте — скрыть пример</template>
          <template v-else>нажмите — перевернуть</template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { usePress } from '../composables/usePress.js'

const props = defineProps({
  card: { type: Object, required: true },
  side: { type: String, default: 'ru' }, // язык лицевой стороны
  flipped: { type: Boolean, default: null },
  showExample: { type: Boolean, default: null },
  interactive: { type: Boolean, default: true },
  size: { type: String, default: 'grid' }, // grid | study
})
const emit = defineEmits(['update:flipped', 'update:showExample'])

const innerFlipped = ref(false)
const innerExample = ref(false)

const isFlipped = computed(() => (props.flipped === null ? innerFlipped.value : props.flipped))
const isExample = computed(() => (props.showExample === null ? innerExample.value : props.showExample))

const toggleFlip = () => {
  innerFlipped.value = !isFlipped.value
  emit('update:flipped', !isFlipped.value)
}
const toggleExample = () => {
  innerExample.value = !isExample.value
  emit('update:showExample', !isExample.value)
}
const press = usePress({ onTap: toggleFlip, onLongPress: toggleExample })

const faceOf = (lang) => ({
  lang,
  word: lang === 'ru' ? props.card.ru_word : props.card.en_word,
  example: lang === 'ru' ? props.card.ru_example : props.card.en_example,
})
const faces = computed(() => {
  const other = props.side === 'ru' ? 'en' : 'ru'
  return [{ key: 'front', ...faceOf(props.side) }, { key: 'back', ...faceOf(other) }]
})
const frontWord = computed(() => faces.value[0].word)
</script>

<style>
.flip-card {
  perspective: 1400px;
  cursor: pointer;
  user-select: none;
  -webkit-user-select: none;
  -webkit-touch-callout: none;
  -webkit-tap-highlight-color: transparent;
  touch-action: manipulation;
  outline: none;
  border-radius: 20px;
}
.flip-card.size-grid { height: 190px; }
.flip-card.size-study { width: 100%; height: 100%; }
.flip-card:focus-visible .flip-face { box-shadow: 0 0 0 2px var(--violet-2), var(--shadow); }
.flip-inner {
  position: relative; width: 100%; height: 100%;
  transform-style: preserve-3d;
  transition: transform 0.55s cubic-bezier(0.3, 1.25, 0.5, 1);
}
.flip-card.flipped .flip-inner { transform: rotateY(180deg); }
.flip-face {
  position: absolute; inset: 0;
  display: flex; flex-direction: column;
  padding: 14px 16px;
  border-radius: 20px;
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  border: 1px solid rgba(196, 181, 253, 0.22);
  box-shadow: 0 8px 28px rgba(3, 2, 20, 0.45);
  overflow: hidden;
}
.flip-face::before {
  content: ''; position: absolute; inset: 0; pointer-events: none;
  background: radial-gradient(120% 80% at 100% 0%, rgba(255, 255, 255, 0.12), transparent 55%);
}
.flip-face.lang-ru { background: linear-gradient(145deg, #3a2a8a 0%, #2a1f6b 45%, #1c1650 100%); }
.flip-face.lang-en { background: linear-gradient(145deg, #1f3f9a 0%, #1b2f78 45%, #141f55 100%); border-color: rgba(147, 197, 253, 0.25); }
.flip-face.back { transform: rotateY(180deg); }
.face-top { display: flex; align-items: center; justify-content: space-between; min-height: 28px; position: relative; z-index: 1; }
.face-actions { display: flex; gap: 2px; opacity: 0; transition: opacity 0.2s; }
.flip-card:hover .face-actions, .flip-card:focus-within .face-actions { opacity: 1; }
@media (hover: none) { .face-actions { opacity: 1; } }
.face-actions .btn { height: 28px; width: 28px; padding: 0; border-radius: 8px; background: rgba(0, 0, 0, 0.25); border-color: transparent; }
.face-actions .btn:hover { background: rgba(0, 0, 0, 0.45); }
.face-main { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 10px; position: relative; z-index: 1; min-height: 0; }
.face-word { font-size: 24px; font-weight: 800; letter-spacing: -0.01em; color: #fff; word-break: break-word; line-height: 1.2; }
.face-word.long { font-size: 19px; }
.size-study .face-word { font-size: clamp(30px, 5vw, 42px); }
.size-study .face-word.long { font-size: clamp(24px, 4vw, 32px); }
.face-example { font-size: 13.5px; line-height: 1.45; color: rgba(236, 235, 255, 0.85); font-style: italic; max-height: 4.4em; overflow: hidden; padding: 0 4px; }
.size-study .face-example { font-size: 17px; max-height: none; max-width: 90%; }
.face-example.empty { opacity: 0.55; }
.face-hint { position: relative; z-index: 1; font-size: 11px; color: rgba(236, 235, 255, 0.5); display: flex; align-items: center; justify-content: center; gap: 5px; min-height: 16px; }
.size-study .face-hint { font-size: 12.5px; }
.example-enter-active, .example-leave-active { transition: opacity 0.25s, transform 0.25s; }
.example-enter-from, .example-leave-to { opacity: 0; transform: translateY(4px); }
</style>
