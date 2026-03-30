<template>
  <div class="register-page">
    <div class="register-shell">
      <section class="register-brand">
        <span class="brand-kicker">{{ brand.productEnglishName }}</span>
        <BrandLogo class="register-logo" :width="320" />
        <h1>加入 {{ brand.productName }}</h1>
        <p class="brand-copy">
          现在注册，直接拥有一套更有品牌感的 AI 对话与识图体验。
        </p>

        <div class="brand-panels">
          <article>
            <span>气质</span>
            <strong>梅花冷艳</strong>
            <p>用深林绿、梅影红与暖金做整套视觉主色。</p>
          </article>
          <article>
            <span>空间</span>
            <strong>沉浸聊天</strong>
            <p>把对话、历史与交互节奏整理成更完整的聊天室体验。</p>
          </article>
          <article>
            <span>能力</span>
            <strong>聊天 + 识图</strong>
            <p>先把核心工作流落地，后续模块可以继续扩展。</p>
          </article>
        </div>

        <div class="signature-block">
          <span>产品名称</span>
          <strong>{{ brand.productName }}</strong>
          <small>一套更有氛围感的 AI 聊天室界面</small>
        </div>
      </section>

      <section class="register-card">
        <div class="register-card__head">
          <span class="panel-kicker">创建账号</span>
          <h2>注册新账号</h2>
          <p>通过邮箱验证码注册，随后即可登录进入聊天空间。</p>
        </div>

        <el-form
          ref="registerFormRef"
          :model="registerForm"
          :rules="registerRules"
          label-position="top"
          class="register-form"
        >
          <el-form-item label="邮箱" prop="email">
            <el-input
              v-model="registerForm.email"
              placeholder="输入常用邮箱"
              type="email"
            />
          </el-form-item>

          <el-form-item label="验证码" prop="captcha">
            <div class="captcha-row">
              <el-input
                v-model="registerForm.captcha"
                placeholder="输入邮箱验证码"
              />
              <el-button
                type="primary"
                class="captcha-btn"
                :loading="codeLoading"
                :disabled="countdown > 0"
                @click="sendCode"
              >
                {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
              </el-button>
            </div>
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="registerForm.password"
              placeholder="至少 6 位密码"
              type="password"
              show-password
            />
          </el-form-item>

          <el-form-item label="确认密码" prop="confirmPassword">
            <el-input
              v-model="registerForm.confirmPassword"
              placeholder="再次输入密码"
              type="password"
              show-password
              @keyup.enter="handleRegister"
            />
          </el-form-item>

          <el-button
            type="primary"
            class="submit-btn"
            :loading="loading"
            @click="handleRegister"
          >
            完成注册
          </el-button>

          <button
            type="button"
            class="text-link"
            @click="$router.push('/login')"
          >
            已有账号，返回登录
          </button>
        </el-form>
      </section>
    </div>
  </div>
</template>

<script>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import brand from '../constants/brand'
import api from '../utils/api'

export default {
  name: 'RegisterView',
  components: {
    BrandLogo
  },
  setup() {
    const router = useRouter()
    const registerFormRef = ref()
    const loading = ref(false)
    const codeLoading = ref(false)
    const countdown = ref(0)

    const registerForm = reactive({
      email: '',
      captcha: '',
      password: '',
      confirmPassword: ''
    })

    const validateConfirmPassword = (rule, value, callback) => {
      if (value !== registerForm.password) {
        callback(new Error('两次输入的密码不一致'))
      } else {
        callback()
      }
    }

    const registerRules = {
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      captcha: [
        { required: true, message: '请输入验证码', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 位', trigger: 'blur' }
      ],
      confirmPassword: [
        { required: true, message: '请再次输入密码', trigger: 'blur' },
        { validator: validateConfirmPassword, trigger: 'blur' }
      ]
    }

    const sendCode = async () => {
      if (!registerForm.email) {
        ElMessage.warning('请先输入邮箱')
        return
      }

      try {
        codeLoading.value = true
        const response = await api.post('/user/captcha', { email: registerForm.email })
        if (response.data.status_code === 1000) {
          ElMessage.success('验证码已发送')
          countdown.value = 60
          const timer = setInterval(() => {
            countdown.value -= 1
            if (countdown.value <= 0) {
              clearInterval(timer)
            }
          }, 1000)
        } else {
          ElMessage.error(response.data.status_msg || '验证码发送失败')
        }
      } catch (error) {
        console.error('Send code error:', error)
        ElMessage.error('验证码发送失败，请稍后重试')
      } finally {
        codeLoading.value = false
      }
    }

    const handleRegister = async () => {
      try {
        await registerFormRef.value.validate()
        loading.value = true

        const response = await api.post('/user/register', {
          email: registerForm.email,
          captcha: registerForm.captcha,
          password: registerForm.password
        })

        if (response.data.status_code === 1000) {
          ElMessage.success('注册成功，请登录')
          router.push('/login')
        } else {
          ElMessage.error(response.data.status_msg || '注册失败')
        }
      } catch (error) {
        console.error('Register error:', error)
        ElMessage.error('注册失败，请稍后重试')
      } finally {
        loading.value = false
      }
    }

    return {
      brand,
      registerFormRef,
      loading,
      codeLoading,
      countdown,
      registerForm,
      registerRules,
      sendCode,
      handleRegister
    }
  }
}
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  padding: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.register-shell {
  width: min(1180px, 100%);
  display: grid;
  grid-template-columns: 1.02fr 0.98fr;
  gap: 24px;
}

.register-brand,
.register-card {
  border-radius: var(--radius-xl);
  border: 1px solid var(--line-soft);
  box-shadow: var(--shadow-xl);
  overflow: hidden;
}

.register-brand {
  padding: 42px;
  color: var(--text-inverse);
  background:
    radial-gradient(circle at 15% 20%, rgba(211, 176, 123, 0.22), transparent 25%),
    radial-gradient(circle at 88% 18%, rgba(182, 79, 97, 0.3), transparent 26%),
    linear-gradient(155deg, rgba(12, 26, 21, 0.94), rgba(20, 57, 46, 0.92));
  display: flex;
  flex-direction: column;
}

.brand-kicker,
.panel-kicker {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 8px 14px;
  border-radius: 999px;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.brand-kicker {
  color: rgba(248, 242, 234, 0.78);
  background: rgba(255, 255, 255, 0.08);
}

.panel-kicker {
  color: var(--plum-deep);
  background: rgba(182, 79, 97, 0.08);
}

.register-brand h1 {
  margin: 24px 0 14px;
  font-size: clamp(38px, 5vw, 60px);
  line-height: 1.08;
}

.register-logo {
  margin-top: 28px;
}

.brand-copy {
  max-width: 520px;
  margin: 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.8);
}

.brand-panels {
  display: grid;
  gap: 14px;
  margin: 34px 0;
}

.brand-panels article {
  padding: 22px 24px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.brand-panels span {
  display: block;
  margin-bottom: 10px;
  color: rgba(248, 242, 234, 0.62);
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-size: 12px;
}

.brand-panels strong {
  display: block;
  font-size: 24px;
  color: var(--gold);
}

.brand-panels p {
  margin: 10px 0 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.78);
}

.signature-block {
  margin-top: auto;
  padding: 26px 28px;
  border-radius: 28px;
  background: linear-gradient(135deg, rgba(182, 79, 97, 0.18), rgba(211, 176, 123, 0.12));
}

.signature-block span,
.signature-block small {
  display: block;
  color: rgba(248, 242, 234, 0.74);
}

.signature-block strong {
  display: block;
  margin: 8px 0;
  font-size: 30px;
}

.register-card {
  padding: 42px;
  align-self: center;
  background: linear-gradient(180deg, rgba(247, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.register-card__head h2 {
  margin: 18px 0 10px;
  font-size: 34px;
}

.register-card__head p {
  margin: 0 0 28px;
  color: var(--text-soft);
  line-height: 1.7;
}

.register-form :deep(.el-form-item__label) {
  padding-bottom: 10px;
  color: var(--text-strong);
  font-weight: 600;
}

.register-form :deep(.el-input__wrapper) {
  min-height: 54px;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(24, 46, 39, 0.08);
  box-shadow: none;
}

.register-form :deep(.el-input__wrapper.is-focus) {
  border-color: rgba(182, 79, 97, 0.4);
  box-shadow: 0 0 0 4px rgba(182, 79, 97, 0.08);
}

.captcha-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
}

.captcha-btn,
.submit-btn {
  border: none;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
}

.captcha-btn {
  min-width: 136px;
  height: 54px;
}

.submit-btn {
  width: 100%;
  height: 54px;
  margin-top: 12px;
  box-shadow: 0 18px 36px rgba(182, 79, 97, 0.22);
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
  .register-page {
    padding: 18px;
  }

  .register-shell {
    grid-template-columns: 1fr;
  }

  .register-brand,
  .register-card {
    padding: 28px 22px;
  }

  .captcha-row {
    grid-template-columns: 1fr;
  }

  .captcha-btn {
    width: 100%;
  }
}
</style>
