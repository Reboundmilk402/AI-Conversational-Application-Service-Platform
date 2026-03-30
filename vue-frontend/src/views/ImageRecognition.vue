<template>
  <div class="vision-page">
    <aside class="vision-sidebar">
      <button type="button" class="back-btn" @click="$router.push('/menu')">返回工作台</button>

      <div class="sidebar-copy">
        <span class="sidebar-kicker">{{ brand.productName }}</span>
        <BrandLogo class="sidebar-logo" :width="250" />
        <h1>识图台</h1>
        <p>上传一张图片，让 AI 返回识别结果。整个页面继续沿用梅花与鹿角的品牌语汇。</p>
      </div>

      <div class="sidebar-cards">
        <article>
          <strong>即传即识别</strong>
          <span>不需要额外参数，适合快速验证图片内容。</span>
        </article>
        <article>
          <strong>结果回显</strong>
          <span>上传记录和识别结果会一起保留在当前页面。</span>
        </article>
      </div>
    </aside>

    <main class="vision-workspace">
      <header class="vision-header">
        <div>
          <span class="header-kicker">Vision Console</span>
          <h2>图像识别助手</h2>
          <p>支持常见图片格式，上传后将直接返回识别结果。</p>
        </div>
      </header>

      <section ref="chatContainerRef" class="vision-feed">
        <div v-if="messages.length === 0" class="feed-empty">
          <span class="empty-badge">上传一张图片开始识别</span>
          <h3>把图片拖进来，或点击下方区域选择文件</h3>
          <p>这里会展示你上传的图片和 AI 返回的识别结果。</p>
        </div>

        <article
          v-for="(message, index) in messages"
          :key="index"
          :class="['feed-message', message.role === 'user' ? 'user-message' : 'assistant-message']"
        >
          <div class="message-head">
            <strong>{{ message.role === 'user' ? '你' : brand.productName }}</strong>
          </div>
          <div class="message-content">
            <span>{{ message.content }}</span>
            <img v-if="message.imageUrl" :src="message.imageUrl" alt="上传图片预览">
          </div>
        </article>
      </section>

      <footer class="upload-panel">
        <form class="upload-form" @submit.prevent="handleSubmit">
          <label class="upload-dropzone" for="vision-upload-input">
            <input
              id="vision-upload-input"
              ref="fileInputRef"
              type="file"
              accept="image/*"
              @change="handleFileSelect"
            >

            <template v-if="selectedPreviewUrl">
              <img :src="selectedPreviewUrl" alt="当前选择图片">
            </template>

            <template v-else>
              <span class="dropzone-title">点击选择图片或拖入这里</span>
              <small>支持 JPG / PNG / WEBP 等常见格式</small>
            </template>
          </label>

          <div class="upload-actions">
            <div class="file-meta">
              <span>当前文件</span>
              <strong>{{ selectedFile ? selectedFile.name : '暂未选择图片' }}</strong>
            </div>

            <button type="submit" class="submit-btn" :disabled="!selectedFile">
              开始识别
            </button>
          </div>
        </form>
      </footer>
    </main>
  </div>
</template>

<script>
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import brand from '../constants/brand'
import api from '../utils/api'

export default {
  name: 'ImageRecognition',
  components: {
    BrandLogo
  },
  setup() {
    const messages = ref([])
    const selectedFile = ref(null)
    const selectedPreviewUrl = ref('')
    const fileInputRef = ref()
    const chatContainerRef = ref()
    const uploadedImageUrls = []

    const revokeSelectedPreview = () => {
      if (selectedPreviewUrl.value) {
        URL.revokeObjectURL(selectedPreviewUrl.value)
        selectedPreviewUrl.value = ''
      }
    }

    const resetPicker = () => {
      selectedFile.value = null
      revokeSelectedPreview()
      if (fileInputRef.value) {
        fileInputRef.value.value = ''
      }
    }

    const handleFileSelect = (event) => {
      const file = event.target.files[0]
      selectedFile.value = file || null
      revokeSelectedPreview()

      if (file) {
        selectedPreviewUrl.value = URL.createObjectURL(file)
      }
    }

    const scrollToBottom = () => {
      if (chatContainerRef.value) {
        chatContainerRef.value.scrollTop = chatContainerRef.value.scrollHeight
      }
    }

    const handleSubmit = async () => {
      if (!selectedFile.value) {
        ElMessage.warning('请先选择一张图片')
        return
      }

      const file = selectedFile.value
      const messageImageUrl = URL.createObjectURL(file)
      uploadedImageUrls.push(messageImageUrl)

      messages.value.push({
        role: 'user',
        content: `已上传图片: ${file.name}`,
        imageUrl: messageImageUrl
      })

      await nextTick()
      scrollToBottom()

      const formData = new FormData()
      formData.append('image', file)

      try {
        const response = await api.post('/image/recognize', formData, {
          headers: {
            'Content-Type': 'multipart/form-data'
          }
        })

        if (response.data && response.data.class_name) {
          messages.value.push({
            role: 'assistant',
            content: `识别结果: ${response.data.class_name}`
          })
        } else {
          messages.value.push({
            role: 'assistant',
            content: `[错误] ${response.data.status_msg || '识别失败'}`
          })
        }
      } catch (error) {
        console.error('Upload error:', error)
        messages.value.push({
          role: 'assistant',
          content: `[错误] 无法连接到服务器或上传失败: ${error.message}`
        })
      } finally {
        resetPicker()
        await nextTick()
        scrollToBottom()
      }
    }

    onBeforeUnmount(() => {
      revokeSelectedPreview()
      uploadedImageUrls.forEach((url) => URL.revokeObjectURL(url))
    })

    return {
      brand,
      chatContainerRef,
      fileInputRef,
      handleFileSelect,
      handleSubmit,
      messages,
      selectedFile,
      selectedPreviewUrl
    }
  }
}
</script>

<style scoped>
.vision-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 20px;
  padding: 20px;
}

.vision-sidebar,
.vision-header,
.upload-panel {
  border: 1px solid var(--line-soft);
  box-shadow: var(--shadow-lg);
}

.vision-sidebar {
  padding: 22px;
  border-radius: var(--radius-xl);
  color: var(--text-inverse);
  background:
    radial-gradient(circle at top left, rgba(211, 176, 123, 0.18), transparent 24%),
    linear-gradient(180deg, rgba(10, 25, 20, 0.92), rgba(19, 53, 42, 0.84));
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.back-btn,
.submit-btn {
  border: none;
  border-radius: 999px;
  cursor: pointer;
}

.back-btn {
  width: fit-content;
  padding: 12px 18px;
  color: var(--text-inverse);
  background: rgba(255, 255, 255, 0.08);
}

.sidebar-kicker,
.header-kicker,
.empty-badge {
  display: inline-flex;
  width: fit-content;
  padding: 8px 14px;
  border-radius: 999px;
  font-size: 12px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.sidebar-kicker {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.78);
}

.sidebar-copy h1 {
  margin: 16px 0 10px;
  font-size: 42px;
}

.sidebar-logo {
  margin-top: 18px;
}

.sidebar-copy p {
  margin: 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.76);
}

.sidebar-cards {
  display: grid;
  gap: 14px;
}

.sidebar-cards article {
  padding: 18px;
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.08);
}

.sidebar-cards strong {
  display: block;
  margin-bottom: 10px;
}

.sidebar-cards span {
  color: rgba(248, 242, 234, 0.72);
  line-height: 1.7;
}

.vision-workspace {
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 18px;
}

.vision-header {
  padding: 24px 26px;
  border-radius: var(--radius-xl);
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.header-kicker {
  background: rgba(18, 53, 42, 0.06);
  color: var(--plum-deep);
}

.vision-header h2 {
  margin: 14px 0 10px;
  font-size: clamp(30px, 4vw, 42px);
}

.vision-header p {
  margin: 0;
  color: var(--text-soft);
}

.vision-feed {
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 8px 8px 8px 12px;
}

.feed-empty {
  margin: auto;
  max-width: 620px;
  padding: 42px 38px;
  border-radius: 32px;
  text-align: center;
  color: var(--text-inverse);
  background:
    radial-gradient(circle at top left, rgba(211, 176, 123, 0.22), transparent 28%),
    linear-gradient(160deg, rgba(10, 25, 20, 0.84), rgba(19, 53, 42, 0.76));
  box-shadow: var(--shadow-lg);
}

.empty-badge {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.82);
}

.feed-empty h3 {
  margin: 18px 0 12px;
  font-size: 36px;
}

.feed-empty p {
  margin: 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.78);
}

.feed-message {
  max-width: min(760px, 82%);
  padding: 18px 20px;
  border-radius: 26px;
  box-shadow: var(--shadow-lg);
}

.user-message {
  align-self: flex-end;
  color: var(--text-inverse);
  background: linear-gradient(135deg, rgba(182, 79, 97, 0.96), rgba(184, 113, 88, 0.88));
}

.assistant-message {
  align-self: flex-start;
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.98), rgba(244, 239, 230, 0.92));
}

.message-head {
  margin-bottom: 10px;
}

.message-content {
  display: grid;
  gap: 12px;
  line-height: 1.8;
}

.message-content img {
  max-width: min(320px, 100%);
  border-radius: 18px;
  box-shadow: var(--shadow-lg);
}

.upload-panel {
  padding: 20px;
  border-radius: var(--radius-xl);
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.upload-form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  gap: 16px;
}

.upload-dropzone {
  min-height: 180px;
  padding: 18px;
  border: 1.5px dashed rgba(24, 46, 39, 0.18);
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.68);
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  cursor: pointer;
  overflow: hidden;
}

.upload-dropzone input {
  display: none;
}

.upload-dropzone img {
  width: 100%;
  max-height: 280px;
  object-fit: contain;
  border-radius: 18px;
}

.dropzone-title {
  display: block;
  font-size: 20px;
  font-weight: 700;
}

.upload-dropzone small {
  display: block;
  margin-top: 10px;
  color: var(--text-soft);
}

.upload-actions {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 14px;
}

.file-meta {
  padding: 20px;
  border-radius: 24px;
  background: rgba(18, 53, 42, 0.06);
}

.file-meta span {
  display: block;
  margin-bottom: 8px;
  color: var(--text-soft);
}

.file-meta strong {
  display: block;
  line-height: 1.7;
  word-break: break-word;
}

.submit-btn {
  height: 54px;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
}

.submit-btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

@media (max-width: 1100px) {
  .vision-page {
    grid-template-columns: 1fr;
  }

  .upload-form {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .vision-page {
    padding: 12px;
  }

  .feed-message {
    max-width: 100%;
  }
}
</style>
