<template>
  <div class="auth-page">
    <div class="auth-shell">
      <section class="auth-brand">
        <span class="brand-kicker">{{ brand.productEnglishName }}</span>
        <BrandLogo class="brand-logo-art" :width="328" />
        <h1>{{ brand.productName }}</h1>
        <p class="brand-copy">{{ brand.longTagline }}</p>

        <div class="brand-tags">
          <span v-for="item in brand.motifLabels" :key="item">{{ item }}</span>
        </div>

        <div class="brand-grid">
          <article>
            <strong>01</strong>
            <span>沉浸式智能对话</span>
          </article>
          <article>
            <strong>02</strong>
            <span>图像识别工作流</span>
          </article>
          <article>
            <strong>03</strong>
            <span>品牌化体验界面</span>
          </article>
        </div>

      </section>

      <section class="auth-card">
        <div class="auth-card__head">
          <span class="panel-kicker">欢迎回来</span>
          <h2>登录到 {{ brand.productName }}</h2>
          <p>支持用户名或邮箱登录，进入这套更有记忆点的 AI 聊天室。</p>
        </div>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="loginRules"
          label-position="top"
          class="auth-form"
        >
          <el-form-item label="用户名或邮箱" prop="account">
            <el-input
              v-model="loginForm.account"
              placeholder="输入用户名或邮箱"
            />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              show-password
              placeholder="输入密码"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-button
            type="primary"
            class="submit-btn"
            :loading="loading"
            @click="handleLogin"
          >
            进入聊天室
          </el-button>

          <button
            type="button"
            class="text-link"
            @click="$router.push('/register')"
          >
            还没有账号，去注册
          </button>
        </el-form>
      </section>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import brand from '../constants/brand'
import api from '../utils/api'

export default {
  name: 'LoginView',
  components: {
    BrandLogo
  },
  setup() {
    const router = useRouter()
    const loginFormRef = ref()
    const loading = ref(false)
    const loginForm = ref({
      account: '',
      password: ''
    })

    const loginRules = {
      account: [
        { required: true, message: '请输入用户名或邮箱', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 位', trigger: 'blur' }
      ]
    }

    const handleLogin = async () => {
      try {
        await loginFormRef.value.validate()
        loading.value = true

        const response = await api.post('/user/login', {
          account: loginForm.value.account,
          password: loginForm.value.password
        })

        if (response.data.status_code === 1000) {
          localStorage.setItem('token', response.data.token)
          ElMessage.success(`欢迎回来，已进入 ${brand.productName}`)
          router.push('/menu')
          return
        }

        ElMessage.error(response.data.status_msg || '登录失败')
      } catch (error) {
        console.error('Login error:', error)
        ElMessage.error('登录失败，请稍后重试')
      } finally {
        loading.value = false
      }
    }

    return {
      brand,
      handleLogin,
      loading,
      loginForm,
      loginFormRef,
      loginRules
    }
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  padding: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.auth-shell {
  width: min(1180px, 100%);
  min-height: 760px;
  display: grid;
  grid-template-columns: 1.08fr 0.92fr;
  gap: 24px;
}

.auth-brand,
.auth-card {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--line-soft);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
}

.auth-brand {
  padding: 44px;
  color: var(--text-inverse);
  background:
    radial-gradient(circle at top left, rgba(182, 79, 97, 0.42), transparent 26%),
    radial-gradient(circle at 85% 20%, rgba(211, 176, 123, 0.28), transparent 28%),
    linear-gradient(165deg, rgba(10, 25, 20, 0.92), rgba(19, 53, 42, 0.9));
  display: flex;
  flex-direction: column;
}

.auth-brand::before {
  content: '';
  position: absolute;
  inset: 22px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 28px;
}

.brand-kicker,
.panel-kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 999px;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.brand-kicker {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.78);
}

.panel-kicker {
  background: rgba(182, 79, 97, 0.08);
  color: var(--plum-deep);
}

.brand-logo-art {
  margin: 34px 0 24px;
}

.auth-brand h1 {
  margin: 0;
  font-size: clamp(40px, 5vw, 64px);
  line-height: 1.04;
}

.brand-copy {
  max-width: 520px;
  margin: 18px 0 0;
  font-size: 17px;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.8);
}

.brand-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 28px 0 34px;
}

.brand-tags span {
  padding: 10px 16px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.92);
}

.brand-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-top: auto;
}

.brand-grid article {
  min-height: 150px;
  padding: 22px 18px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.brand-grid strong {
  font-size: 28px;
  color: var(--gold);
}

.brand-grid span {
  line-height: 1.7;
  color: rgba(248, 242, 234, 0.78);
}

.auth-card {
  padding: 42px;
  align-self: center;
  background: linear-gradient(180deg, rgba(247, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.auth-card__head h2 {
  margin: 18px 0 10px;
  font-size: 34px;
}

.auth-card__head p {
  margin: 0 0 28px;
  color: var(--text-soft);
  line-height: 1.7;
}

.auth-form :deep(.el-form-item__label) {
  padding-bottom: 10px;
  color: var(--text-strong);
  font-weight: 600;
}

.auth-form :deep(.el-input__wrapper) {
  min-height: 54px;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(24, 46, 39, 0.08);
  box-shadow: none;
}

.auth-form :deep(.el-input__wrapper.is-focus) {
  border-color: rgba(182, 79, 97, 0.4);
  box-shadow: 0 0 0 4px rgba(182, 79, 97, 0.08);
}

.submit-btn {
  width: 100%;
  height: 54px;
  margin-top: 12px;
  border: none;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
  box-shadow: 0 18px 36px rgba(182, 79, 97, 0.22);
}

.submit-btn:hover {
  transform: translateY(-1px);
}

.text-link {
  width: 100%;
  margin-top: 18px;
  padding: 14px 18px;
  border: none;
  background: transparent;
  color: var(--plum-deep);
  cursor: pointer;
}

@media (max-width: 980px) {
  .auth-page {
    padding: 18px;
  }

  .auth-shell {
    min-height: auto;
    grid-template-columns: 1fr;
  }

  .auth-brand,
  .auth-card {
    padding: 28px 22px;
  }

  .brand-grid {
    grid-template-columns: 1fr;
  }
}
</style>
