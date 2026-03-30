<template>
  <div id="app">
    <router-view v-slot="{ Component }">
      <transition name="page" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
:root {
  --bg-night: #081410;
  --bg-forest: #12352a;
  --bg-moss: #28483d;
  --bg-paper: #f4efe6;
  --bg-paper-strong: rgba(247, 242, 234, 0.9);
  --panel-dark: rgba(10, 25, 20, 0.56);
  --panel-soft: rgba(255, 248, 241, 0.14);
  --line-soft: rgba(255, 255, 255, 0.1);
  --line-strong: rgba(24, 46, 39, 0.1);
  --text-strong: #17241e;
  --text-soft: #5f6d67;
  --text-inverse: #f8f2ea;
  --plum: #b64f61;
  --plum-deep: #8b3146;
  --gold: #d3b07b;
  --moss: #86a277;
  --shadow-xl: 0 28px 80px rgba(5, 13, 11, 0.28);
  --shadow-lg: 0 18px 48px rgba(10, 22, 18, 0.18);
  --radius-xl: 32px;
  --radius-lg: 24px;
  --radius-md: 18px;
  --radius-sm: 12px;
}

* {
  box-sizing: border-box;
}

html,
body,
#app {
  min-height: 100%;
}

html {
  background:
    radial-gradient(circle at top left, rgba(182, 79, 97, 0.16), transparent 28%),
    radial-gradient(circle at 85% 10%, rgba(211, 176, 123, 0.14), transparent 24%),
    linear-gradient(145deg, #07110d 0%, #10241d 42%, #17342b 100%);
}

body {
  margin: 0;
  color: var(--text-strong);
  font-family: 'PingFang SC', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  overflow-x: hidden;
}

body::before {
  content: '';
  position: fixed;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(circle at 12% 18%, rgba(255, 255, 255, 0.06), transparent 0 24%),
    radial-gradient(circle at 78% 26%, rgba(255, 255, 255, 0.05), transparent 0 18%),
    radial-gradient(circle at 22% 78%, rgba(182, 79, 97, 0.08), transparent 0 20%);
  opacity: 0.9;
}

button,
input,
textarea,
select {
  font: inherit;
}

h1,
h2,
h3,
h4 {
  font-family: 'Baskerville', 'Times New Roman', 'Source Han Serif SC', 'Noto Serif SC', serif;
  letter-spacing: 0.02em;
}

a {
  color: inherit;
  text-decoration: none;
}

#app {
  position: relative;
}

.page-enter-active,
.page-leave-active {
  transition: opacity 0.35s ease, transform 0.35s ease;
}

.page-enter-from,
.page-leave-to {
  opacity: 0;
  transform: translateY(18px);
}

.el-overlay {
  backdrop-filter: blur(10px);
}

.el-button {
  border-radius: 999px;
  font-weight: 600;
}

.el-input__wrapper,
.el-textarea__inner {
  border-radius: 16px;
  box-shadow: none;
}

.el-card,
.el-message,
.el-message-box {
  border-radius: 20px;
}

::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.04);
}

::-webkit-scrollbar-thumb {
  background: rgba(211, 176, 123, 0.35);
  border-radius: 999px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(211, 176, 123, 0.55);
}

@media (max-width: 768px) {
  .page-enter-active,
  .page-leave-active {
    transition: opacity 0.25s ease;
  }

  .page-enter-from,
  .page-leave-to {
    transform: none;
  }
}
</style>
