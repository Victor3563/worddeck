<template>
  <div class="landing">
    <section class="container hero">
      <div class="hero-text fade-up">
        <span class="badge violet"><Icon name="sparkles" :size="14" /> Карточки для английского</span>
        <h1>Учите слова <span class="gradient-text">в контексте</span>, а не списком</h1>
        <p class="lead">
          Собирайте слова в области и папки, добавляйте примеры употребления и тренируйтесь тремя способами:
          свайпы, диктант и сопоставление. Делитесь наборами по ссылке.
        </p>
        <div class="row wrap">
          <RouterLink to="/register" class="btn primary lg">Начать бесплатно <Icon name="arrow-right" /></RouterLink>
          <RouterLink to="/login" class="btn lg">У меня есть аккаунт</RouterLink>
        </div>
        <p class="muted small try-hint"><Icon name="hand" :size="14" /> Попробуйте карточку справа: нажмите — перевернётся, удерживайте — появится пример.</p>
      </div>
      <div class="hero-demo fade-up">
        <div class="demo-stack">
          <div class="demo-shadow s2"></div>
          <div class="demo-shadow s1"></div>
          <FlipCard :card="demo" side="en" size="study" class="demo-card" />
        </div>
      </div>
    </section>

    <section class="container features">
      <div v-for="f in features" :key="f.title" class="feature panel">
        <div class="feature-icon"><Icon :name="f.icon" :size="22" /></div>
        <h3>{{ f.title }}</h3>
        <p class="muted">{{ f.text }}</p>
      </div>
    </section>

    <div class="container">
    <section class="how panel">
      <h2>Как это устроено</h2>
      <ol class="steps">
        <li><span>1</span><div><b>Создайте область</b><p class="muted">Например, «Английский B2» или «IT-лексика».</p></div></li>
        <li><span>2</span><div><b>Разложите слова по папкам</b><p class="muted">По одной через форму или сразу списком из CSV/TXT/JSON-файла.</p></div></li>
        <li><span>3</span><div><b>Тренируйтесь</b><p class="muted">Свайп влево — знаю, вправо — повторить. Непонятые карточки вернутся, пока все не будут выучены.</p></div></li>
      </ol>
      <RouterLink to="/register" class="btn primary">Создать аккаунт</RouterLink>
    </section>
    </div>

    <footer class="container foot muted small">
      WordDeck · учебный проект · Go + PostgreSQL + Vue
    </footer>
  </div>
</template>

<script setup>
import FlipCard from '../components/FlipCard.vue'

const demo = {
  ru_word: 'настойчивый',
  ru_example: 'Он был настойчив и в итоге добился своего.',
  en_word: 'persistent',
  en_example: 'He was persistent and finally got what he wanted.',
}
const features = [
  { icon: 'layers', title: 'Области и папки', text: 'Своё пространство для каждой цели: экзамены, работа, путешествия.' },
  { icon: 'rotate', title: 'Переворот и контекст', text: 'Видно только слово. Касание переворачивает, долгое нажатие показывает пример.' },
  { icon: 'hand', title: 'Учить свайпами', text: 'Влево — знаю, вправо — ещё раз. Карточки перемешиваются, пока все не выучены.' },
  { icon: 'keyboard', title: 'Диктант', text: 'Пишите перевод без ошибок. Ошибочные слова придётся написать заново.' },
  { icon: 'match', title: 'Сопоставление', text: 'Перетаскивайте слова к их переводам на время.' },
  { icon: 'link', title: 'Ссылки и импорт', text: 'Делитесь областью, папкой или карточкой по ссылке. Импорт и экспорт в CSV/JSON.' },
]
</script>

<style>
.hero { display: grid; grid-template-columns: 1.1fr 0.9fr; gap: 48px; align-items: center; padding-top: 64px; padding-bottom: 56px; }
.hero-text { display: flex; flex-direction: column; gap: 20px; align-items: flex-start; }
.hero-text h1 { font-size: clamp(34px, 5vw, 56px); line-height: 1.05; }
.lead { font-size: 17px; color: var(--text-2); max-width: 560px; }
.try-hint { display: flex; align-items: center; gap: 6px; }
.hero-demo { display: flex; justify-content: center; animation-delay: 0.1s; }
.demo-stack { position: relative; width: min(380px, 100%); height: 260px; }
.demo-card { position: relative; z-index: 2; }
.demo-shadow { position: absolute; inset: 0; border-radius: 20px; border: 1px solid var(--border); }
.demo-shadow.s1 { transform: rotate(4deg) translate(10px, 8px); background: rgba(99, 102, 241, 0.25); }
.demo-shadow.s2 { transform: rotate(-5deg) translate(-12px, 12px); background: rgba(59, 130, 246, 0.16); }
.features { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; }
.feature { display: flex; flex-direction: column; gap: 8px; }
.feature-icon { width: 44px; height: 44px; border-radius: 14px; display: grid; place-items: center; background: var(--grad-soft); color: #c4b5fd; margin-bottom: 4px; }
.how { margin-top: 32px; display: flex; flex-direction: column; gap: 18px; align-items: flex-start; }
.steps { list-style: none; padding: 0; margin: 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 18px; width: 100%; }
.steps li { display: flex; gap: 14px; }
.steps li > span { flex-shrink: 0; width: 34px; height: 34px; border-radius: 50%; display: grid; place-items: center; background: var(--grad); font-weight: 800; }
.foot { padding: 40px 20px 50px; text-align: center; }
@media (max-width: 860px) {
  .hero { grid-template-columns: 1fr; padding-top: 36px; gap: 36px; }
}
</style>
