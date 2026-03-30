<template>
  <div class="menu-page">
    <header class="menu-toolbar">
      <span class="toolbar-kicker">AI Chatroom</span>
      <el-button class="logout-btn" @click="handleLogout">退出登录</el-button>
    </header>

    <main class="menu-main">
      <section class="hero-panel">
        <div class="hero-copy">
          <span class="section-kicker">空间概念</span>
          <h2>做成一套更有记忆点的 AI 聊天室。</h2>
          <p>
            不再只做一个普通工具页，而是把对话、历史、灵感和视觉氛围整理成一个更完整的聊天空间。
          </p>

          <div class="hero-tags">
            <span v-for="item in brand.motifLabels" :key="item">{{ item }}</span>
          </div>
        </div>

        <div class="hero-stats">
          <article>
            <strong>2</strong>
            <span>当前功能入口</span>
          </article>
          <article>
            <strong>AI</strong>
            <span>对话优先的主界面</span>
          </article>
          <article>
            <strong>Flow</strong>
            <span>更沉浸的使用节奏</span>
          </article>
        </div>
      </section>

      <section class="feature-grid">
        <article class="feature-card feature-card-chat">
          <div class="feature-index">01</div>
          <h3>智能对话</h3>
          <p>管理多轮会话、切换模型、保留上下文，并在需要时开启联网搜索。</p>
          <button type="button" class="feature-btn" @click="$router.push('/ai-chat')">
            进入聊天室
          </button>
        </article>

        <article class="feature-card feature-card-image">
          <div class="feature-index">02</div>
          <h3>图像识别</h3>
          <p>上传图片后快速返回识别结果，适合作为聊天空间之外的辅助视觉能力。</p>
          <button type="button" class="feature-btn" @click="$router.push('/image-recognition')">
            进入识图台
          </button>
        </article>
      </section>
    </main>
  </div>
</template>

<script>
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import brand from '../constants/brand'

export default {
  name: 'MenuView',
  setup() {
    const router = useRouter()

    const handleLogout = async () => {
      try {
        await ElMessageBox.confirm('确认退出当前账号吗？', '退出登录', {
          confirmButtonText: '退出',
          cancelButtonText: '取消',
          type: 'warning'
        })
        localStorage.removeItem('token')
        ElMessage.success('已退出登录')
        router.push('/login')
      } catch {
        // 用户取消
      }
    }

    return {
      brand,
      handleLogout
    }
  }
}
</script>

<style scoped>
.menu-page {
  min-height: 100vh;
  padding: 24px;
  color: var(--text-inverse);
}

.menu-toolbar,
.hero-panel,
.feature-card {
  border: 1px solid var(--line-soft);
  box-shadow: var(--shadow-lg);
}

.menu-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 16px 22px;
  border-radius: 999px;
  background: rgba(10, 25, 20, 0.48);
  backdrop-filter: blur(12px);
}

.toolbar-kicker,
.section-kicker {
  display: inline-flex;
  width: fit-content;
  padding: 14px 24px;
  border-radius: 999px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  font-size: 18px;
  font-weight: 700;
}

.toolbar-kicker {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.82);
}

.logout-btn {
  height: 44px;
  padding: 0 20px;
  border: none;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
}

.menu-main {
  margin-top: 24px;
  display: grid;
  gap: 24px;
}

.hero-panel {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 22px;
  padding: 30px;
  border-radius: var(--radius-xl);
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.98), rgba(244, 239, 230, 0.92));
  color: var(--text-strong);
}

.section-kicker {
  background: rgba(182, 79, 97, 0.08);
  color: var(--plum-deep);
}

.hero-copy h2 {
  margin: 18px 0 14px;
  font-size: clamp(30px, 4vw, 48px);
  line-height: 1.16;
}

.hero-copy p {
  margin: 0;
  line-height: 1.9;
  color: var(--text-soft);
}

.hero-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 24px;
}

.hero-tags span {
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(18, 53, 42, 0.06);
  color: var(--text-strong);
}

.hero-stats {
  display: grid;
  gap: 14px;
}

.hero-stats article {
  padding: 24px;
  border-radius: 24px;
  background: rgba(18, 53, 42, 0.05);
}

.hero-stats strong {
  display: block;
  margin-bottom: 10px;
  font-size: 34px;
  color: var(--plum-deep);
}

.hero-stats span {
  color: var(--text-soft);
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
}

.feature-card {
  min-height: 320px;
  padding: 28px;
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.9));
  color: var(--text-strong);
}

.feature-card-chat {
  background:
    radial-gradient(circle at top right, rgba(182, 79, 97, 0.12), transparent 24%),
    linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.9));
}

.feature-card-image {
  background:
    radial-gradient(circle at top right, rgba(134, 162, 119, 0.18), transparent 26%),
    linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.9));
}

.feature-index {
  width: fit-content;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(18, 53, 42, 0.07);
  color: var(--plum-deep);
  font-size: 13px;
  letter-spacing: 0.14em;
}

.feature-card h3 {
  margin: 22px 0 12px;
  font-size: 30px;
}

.feature-card p {
  margin: 0;
  line-height: 1.8;
  color: var(--text-soft);
}

.feature-btn {
  width: fit-content;
  margin-top: auto;
  padding: 14px 22px;
  border: none;
  border-radius: 999px;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
  cursor: pointer;
}

@media (max-width: 1040px) {
  .menu-page {
    padding: 16px;
  }

  .hero-panel,
  .feature-grid {
    grid-template-columns: 1fr;
  }

  .menu-toolbar {
    border-radius: 24px;
  }
}
</style>
