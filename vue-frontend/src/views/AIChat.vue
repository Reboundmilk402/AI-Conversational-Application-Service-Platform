<template>
  <div class="chat-page">
    <aside class="chat-sidebar">
      <div class="sidebar-brand">
        <span class="sidebar-kicker">{{ brand.productName }}</span>
        <BrandLogo class="sidebar-logo" :width="250" />
        <h1>对话台</h1>
        <p>管理多轮会话、模型切换和联网辅助，把思路都收进同一个工作区。</p>
      </div>

      <button type="button" class="create-btn" @click="createNewSession">+ 新建会话</button>

      <div class="sidebar-summary">
        <article>
          <strong>{{ sessionList.length }}</strong>
          <span>历史会话</span>
        </article>
        <article>
          <strong>{{ selectedModelLabel }}</strong>
          <span>当前模型</span>
        </article>
      </div>

      <div v-if="sessionList.length === 0" class="sidebar-empty">
        还没有历史会话。现在就发起一段新的提问。
      </div>

      <ul v-else class="session-list">
        <li
          v-for="session in sessionList"
          :key="session.id"
          :class="['session-item', { active: currentSessionId === session.id }]"
          @click="switchSession(session.id)"
        >
          <div class="session-item__body">
            <span class="session-title">{{ session.name }}</span>
            <small>{{ session.modelType === '2' ? 'Qwen2.5' : '豆包' }}</small>
          </div>

          <button
            type="button"
            class="icon-btn"
            title="删除会话"
            @click.stop="deleteSession(session.id)"
          >
            ×
          </button>
        </li>
      </ul>
    </aside>

    <main class="chat-workspace">
      <header class="workspace-header">
        <div class="workspace-title">
          <button type="button" class="ghost-btn" @click="$router.push('/menu')">返回首页</button>
          <span class="workspace-kicker">当前会话 · {{ currentSessionName }}</span>
          <h2>{{ isTempSession ? '新的灵感对话' : currentSessionName }}</h2>
          <p>
            当前模型: {{ selectedModelLabel }}
            · {{ enableWebSearch ? '联网搜索已开启' : '当前仅使用基础问答' }}
          </p>
        </div>

        <div class="workspace-actions">
          <button
            type="button"
            class="ghost-btn"
            :disabled="!currentSessionId || isTempSession || loading"
            @click="syncHistory"
          >
            同步历史
          </button>

          <label class="control-card">
            <span>模型</span>
            <select
              v-model="selectedModel"
              class="select-input"
              :disabled="loading || (!isTempSession && !!currentSessionId)"
            >
              <option
                v-for="option in modelOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </label>

          <label class="toggle-card">
            <input v-model="enableWebSearch" type="checkbox">
            <span>联网搜索</span>
          </label>

          <label class="toggle-card">
            <input v-model="isStreaming" type="checkbox">
            <span>流式响应</span>
          </label>
        </div>
      </header>

      <section ref="messagesRef" class="messages">
        <div v-if="currentMessages.length === 0" class="empty-state">
          <span class="empty-badge">Immersive AI Chatroom</span>
          <h3>从这里开始一段新的问题</h3>
          <p>{{ brand.tagline }}</p>
          <div class="empty-tags">
            <span>对话</span>
            <span>会话管理</span>
            <span>识图联动</span>
          </div>
        </div>

        <article
          v-for="(message, index) in currentMessages"
          :key="index"
          :class="['message-card', message.role === 'user' ? 'user-card' : 'assistant-card']"
        >
          <div class="message-role">{{ message.role === 'user' ? '你' : brand.productName }}</div>
          <div class="message-body" v-html="renderMarkdown(message.content)" />
          <div v-if="message.meta?.status === 'streaming'" class="message-status">正在生成中...</div>
          <div v-else-if="message.meta?.status === 'error'" class="message-status error">生成失败</div>
        </article>
      </section>

      <footer class="composer-shell">
        <div class="composer-head">
          <span>输入提示</span>
          <p>按 Enter 发送，Shift + Enter 换行。</p>
        </div>

        <div class="composer">
          <textarea
            ref="messageInput"
            v-model="inputMessage"
            :disabled="loading"
            rows="1"
            :placeholder="`向 ${brand.productName} 输入你的问题...`"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <button
            type="button"
            class="send-btn"
            :disabled="loading || !inputMessage.trim()"
            @click="sendMessage"
          >
            {{ loading ? '生成中...' : '发送' }}
          </button>
        </div>
      </footer>
    </main>
  </div>
</template>

<script>
import { computed, nextTick, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import brand from '../constants/brand'
import api from '../utils/api'

const modelOptions = [
  { value: '1', label: '豆包' },
  { value: '2', label: 'Qwen2.5 (Ollama)' }
]

const cloneMessages = (messages = []) => messages.map((message) => ({
  role: message.role,
  content: message.content,
  meta: message.meta ? { ...message.meta } : undefined
}))

const escapeHtml = (text) => String(text ?? '')
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const isCorruptedText = (text) => {
  const value = String(text ?? '').trim()
  if (!value) {
    return false
  }

  const questionMarkCount = (value.match(/\?/g) || []).length
  return /\uFFFD/.test(value) || /\?{3,}/.test(value) || (questionMarkCount >= 4 && questionMarkCount / value.length > 0.28)
}

const formatSessionName = (name, sessionId) => {
  const value = String(name ?? '').trim()
  if (value && !isCorruptedText(value)) {
    return value
  }
  return `历史会话 ${String(sessionId).slice(0, 6)}`
}

const normalizeHistoryContent = (content, isUser) => {
  const value = String(content ?? '')
  if (!isCorruptedText(value)) {
    return value
  }

  if (isUser) {
    return '该历史提问包含异常字符，原始内容可能在更早之前写入数据库时已经损坏。'
  }

  return value
}

export default {
  name: 'AIChat',
  components: {
    BrandLogo
  },
  setup() {
    const sessions = ref({})
    const currentSessionId = ref('')
    const currentMessages = ref([])
    const inputMessage = ref('')
    const loading = ref(false)
    const isTempSession = ref(true)
    const selectedModel = ref('1')
    const enableWebSearch = ref(false)
    const isStreaming = ref(false)
    const messagesRef = ref(null)
    const messageInput = ref(null)

    const sessionList = computed(() => Object.values(sessions.value))
    const currentSessionName = computed(() => {
      if (isTempSession.value) {
        return '未保存新会话'
      }
      return sessions.value[currentSessionId.value]?.name || '对话会话'
    })
    const selectedModelLabel = computed(() => {
      return modelOptions.find((option) => option.value === selectedModel.value)?.label || '未选择模型'
    })

    const renderMarkdown = (text) => {
      const safeText = escapeHtml(text)
      return safeText
        .replace(/`([^`]+)`/g, '<code>$1</code>')
        .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
        .replace(/\*([^*]+)\*/g, '<em>$1</em>')
        .replace(/\n/g, '<br>')
    }

    const focusInput = async () => {
      await nextTick()
      if (messageInput.value) {
        messageInput.value.focus()
      }
    }

    const scrollToBottom = async () => {
      await nextTick()
      if (messagesRef.value) {
        messagesRef.value.scrollTop = messagesRef.value.scrollHeight
      }
    }

    const syncCurrentToCache = () => {
      if (!isTempSession.value && currentSessionId.value && sessions.value[currentSessionId.value]) {
        sessions.value[currentSessionId.value].messages = cloneMessages(currentMessages.value)
      }
    }

    const setCurrentMessages = async (messages) => {
      currentMessages.value = cloneMessages(messages)
      syncCurrentToCache()
      await scrollToBottom()
    }

    const createNewSession = async () => {
      currentSessionId.value = ''
      currentMessages.value = []
      inputMessage.value = ''
      isTempSession.value = true
      await focusInput()
    }

    const loadSessions = async () => {
      try {
        const response = await api.get('/AI/chat/sessions')
        if (!response.data || response.data.status_code !== 1000 || !Array.isArray(response.data.sessions)) {
          ElMessage.error(response.data?.status_msg || '加载会话列表失败')
          return
        }

        const sessionMap = {}
        response.data.sessions.forEach((session) => {
          const sessionId = String(session.sessionId)
          sessionMap[sessionId] = {
            id: sessionId,
            name: formatSessionName(session.name, sessionId),
            messages: null,
            modelType: String(session.modelType || '1')
          }
        })
        sessions.value = sessionMap

        const ids = Object.keys(sessionMap)
        if (ids.length > 0) {
          await switchSession(ids[0])
        } else {
          await createNewSession()
        }
      } catch (error) {
        console.error('loadSessions error:', error)
        ElMessage.error('加载会话列表失败')
      }
    }

    const fetchHistory = async (sessionId) => {
      const response = await api.post('/AI/chat/history', { sessionId })
      if (!response.data || response.data.status_code !== 1000 || !Array.isArray(response.data.history)) {
        throw new Error(response.data?.status_msg || '加载历史消息失败')
      }

      return response.data.history.map((item) => ({
        role: item.is_user ? 'user' : 'assistant',
        content: normalizeHistoryContent(item.content, item.is_user)
      }))
    }

    const switchSession = async (sessionId) => {
      const targetId = String(sessionId)
      const targetSession = sessions.value[targetId]
      if (!targetSession) {
        return
      }

      currentSessionId.value = targetId
      isTempSession.value = false
      selectedModel.value = targetSession.modelType || '1'

      try {
        if (!Array.isArray(targetSession.messages)) {
          targetSession.messages = await fetchHistory(targetId)
        }
        await setCurrentMessages(targetSession.messages)
      } catch (error) {
        console.error('switchSession error:', error)
        ElMessage.error(error.message || '加载会话失败')
      }
    }

    const syncHistory = async () => {
      if (!currentSessionId.value || isTempSession.value) {
        ElMessage.warning('请先选择一个已保存的会话')
        return
      }

      try {
        const messages = await fetchHistory(currentSessionId.value)
        sessions.value[currentSessionId.value].messages = messages
        await setCurrentMessages(messages)
        ElMessage.success('历史消息已同步')
      } catch (error) {
        console.error('syncHistory error:', error)
        ElMessage.error(error.message || '同步历史消息失败')
      }
    }

    const deleteSession = async (sessionId) => {
      const targetId = String(sessionId)
      const targetSession = sessions.value[targetId]
      if (!targetSession) {
        return
      }

      try {
        await ElMessageBox.confirm(
          `确认删除“${targetSession.name}”吗？删除后无法恢复。`,
          '删除会话',
          {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
      } catch {
        return
      }

      try {
        const response = await api.post('/AI/chat/delete', { sessionId: targetId })
        if (!response.data || response.data.status_code !== 1000) {
          ElMessage.error(response.data?.status_msg || '删除会话失败')
          return
        }

        delete sessions.value[targetId]

        if (currentSessionId.value === targetId) {
          const remainingIds = Object.keys(sessions.value)
          if (remainingIds.length > 0) {
            await switchSession(remainingIds[0])
          } else {
            await createNewSession()
          }
        }

        ElMessage.success('会话已删除')
      } catch (error) {
        console.error('deleteSession error:', error)
        ElMessage.error('删除会话失败')
      }
    }

    const createSessionEntry = (sessionId, firstQuestion) => {
      const name = firstQuestion.trim().slice(0, 24) || `会话 ${sessionId}`
      const newEntry = {
        id: sessionId,
        name,
        messages: cloneMessages(currentMessages.value),
        modelType: selectedModel.value
      }
      sessions.value = {
        [sessionId]: newEntry,
        ...sessions.value
      }
    }

    const finalizeStreamingMessage = (assistantIndex, status) => {
      if (currentMessages.value[assistantIndex]) {
        currentMessages.value[assistantIndex].meta = { status }
        syncCurrentToCache()
      }
    }

    const streamRequest = async (question) => {
      const assistantIndex = currentMessages.value.length
      currentMessages.value.push({
        role: 'assistant',
        content: '',
        meta: { status: 'streaming' }
      })
      syncCurrentToCache()
      await scrollToBottom()

      const endpoint = isTempSession.value
        ? '/AI/chat/send-stream-new-session'
        : '/AI/chat/send-stream'

      const response = await fetch(`${api.defaults.baseURL}${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token') || ''}`
        },
        body: JSON.stringify(
          isTempSession.value
            ? { question, modelType: selectedModel.value, enableWebSearch: enableWebSearch.value }
            : {
                question,
                modelType: selectedModel.value,
                sessionId: currentSessionId.value,
                enableWebSearch: enableWebSearch.value
              }
        )
      })

      if (!response.ok || !response.body) {
        throw new Error('流式请求失败')
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let streamDone = false

      while (!streamDone) {
        const { value, done } = await reader.read()
        streamDone = done
        if (streamDone) {
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const rawLine of lines) {
          const line = rawLine.trim()
          if (!line.startsWith('data:')) {
            continue
          }

          const payload = line.slice(5).trim()
          if (!payload) {
            continue
          }

          if (payload === '[DONE]') {
            finalizeStreamingMessage(assistantIndex, 'done')
            continue
          }

          if (payload.startsWith('{')) {
            try {
              const parsed = JSON.parse(payload)
              if (parsed.sessionId) {
                const newSessionId = String(parsed.sessionId)
                if (isTempSession.value) {
                  currentSessionId.value = newSessionId
                  isTempSession.value = false
                  createSessionEntry(newSessionId, question)
                }
                continue
              }
            } catch (error) {
              console.error('parse stream json error:', error)
            }
          }

          if (currentMessages.value[assistantIndex]) {
            currentMessages.value[assistantIndex].content += payload
            syncCurrentToCache()
          }
          await scrollToBottom()
        }
      }

      finalizeStreamingMessage(assistantIndex, 'done')
    }

    const normalRequest = async (question) => {
      if (isTempSession.value) {
        const response = await api.post('/AI/chat/send-new-session', {
          question,
          modelType: selectedModel.value,
          enableWebSearch: enableWebSearch.value
        })

        if (!response.data || response.data.status_code !== 1000) {
          throw new Error(response.data?.status_msg || '发送失败')
        }

        const newSessionId = String(response.data.sessionId)
        currentSessionId.value = newSessionId
        isTempSession.value = false
        currentMessages.value.push({
          role: 'assistant',
          content: response.data.Information || ''
        })
        createSessionEntry(newSessionId, question)
        syncCurrentToCache()
        return
      }

      const response = await api.post('/AI/chat/send', {
        question,
        modelType: selectedModel.value,
        sessionId: currentSessionId.value,
        enableWebSearch: enableWebSearch.value
      })

      if (!response.data || response.data.status_code !== 1000) {
        throw new Error(response.data?.status_msg || '发送失败')
      }

      currentMessages.value.push({
        role: 'assistant',
        content: response.data.Information || ''
      })
      syncCurrentToCache()
    }

    const sendMessage = async () => {
      const question = inputMessage.value.trim()
      if (!question || loading.value) {
        if (!question) {
          ElMessage.warning('请输入消息内容')
        }
        return
      }

      loading.value = true
      inputMessage.value = ''
      currentMessages.value.push({ role: 'user', content: question })
      syncCurrentToCache()
      await scrollToBottom()

      try {
        if (isStreaming.value) {
          await streamRequest(question)
        } else {
          await normalRequest(question)
        }
      } catch (error) {
        console.error('sendMessage error:', error)
        currentMessages.value.pop()
        syncCurrentToCache()
        ElMessage.error(error.message || '发送失败')
      } finally {
        loading.value = false
        await scrollToBottom()
      }
    }

    onMounted(() => {
      loadSessions()
    })

    return {
      brand,
      currentMessages,
      currentSessionId,
      currentSessionName,
      createNewSession,
      deleteSession,
      enableWebSearch,
      inputMessage,
      isStreaming,
      isTempSession,
      loading,
      messageInput,
      messagesRef,
      modelOptions,
      renderMarkdown,
      selectedModel,
      selectedModelLabel,
      sendMessage,
      sessionList,
      switchSession,
      syncHistory
    }
  }
}
</script>

<style scoped>
.chat-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 20px;
  padding: 20px;
}

.chat-sidebar,
.workspace-header,
.composer-shell {
  border: 1px solid var(--line-soft);
  box-shadow: var(--shadow-lg);
}

.chat-sidebar {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 22px;
  border-radius: var(--radius-xl);
  color: var(--text-inverse);
  background:
    radial-gradient(circle at top left, rgba(182, 79, 97, 0.22), transparent 24%),
    linear-gradient(180deg, rgba(10, 25, 20, 0.92), rgba(19, 53, 42, 0.84));
}

.sidebar-brand h1 {
  margin: 16px 0 10px;
  font-size: 40px;
}

.sidebar-logo {
  margin-top: 18px;
}

.sidebar-brand p {
  margin: 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.76);
}

.sidebar-kicker,
.workspace-kicker,
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
  color: rgba(248, 242, 234, 0.82);
}

.create-btn {
  height: 48px;
  border: none;
  border-radius: 999px;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
  cursor: pointer;
}

.sidebar-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.sidebar-summary article {
  padding: 16px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.08);
}

.sidebar-summary strong {
  display: block;
  margin-bottom: 8px;
  font-size: 24px;
}

.sidebar-summary span {
  color: rgba(248, 242, 234, 0.68);
  font-size: 13px;
}

.sidebar-empty {
  padding: 18px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.76);
  line-height: 1.7;
}

.session-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 14px 14px 16px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.06);
  cursor: pointer;
  transition: transform 0.18s ease, background 0.18s ease;
}

.session-item:hover {
  transform: translateX(4px);
  background: rgba(255, 255, 255, 0.1);
}

.session-item.active {
  background: linear-gradient(135deg, rgba(182, 79, 97, 0.96), rgba(184, 113, 88, 0.88));
}

.session-item__body {
  flex: 1;
  min-width: 0;
}

.session-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.session-item small {
  color: rgba(248, 242, 234, 0.72);
}

.icon-btn {
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 999px;
  background: rgba(10, 25, 20, 0.22);
  color: inherit;
  cursor: pointer;
}

.chat-workspace {
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 18px;
}

.workspace-header {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  padding: 24px 26px;
  border-radius: var(--radius-xl);
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.workspace-title h2 {
  margin: 14px 0 10px;
  font-size: clamp(30px, 4vw, 42px);
}

.workspace-title p {
  margin: 0;
  color: var(--text-soft);
}

.workspace-kicker {
  margin-top: 16px;
  background: rgba(18, 53, 42, 0.06);
  color: var(--plum-deep);
}

.workspace-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: flex-end;
  gap: 12px;
}

.ghost-btn,
.send-btn {
  border: none;
  border-radius: 999px;
  cursor: pointer;
}

.ghost-btn {
  padding: 12px 18px;
  color: var(--text-strong);
  background: rgba(18, 53, 42, 0.06);
}

.control-card,
.toggle-card {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 0 16px;
  border-radius: 999px;
  background: rgba(18, 53, 42, 0.06);
  color: var(--text-strong);
}

.control-card span,
.toggle-card span {
  white-space: nowrap;
}

.select-input {
  border: none;
  background: transparent;
  color: inherit;
  outline: none;
}

.messages {
  min-height: 0;
  overflow-y: auto;
  padding: 12px 8px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.empty-state {
  margin: auto;
  max-width: 620px;
  padding: 42px 38px;
  border-radius: 32px;
  text-align: center;
  color: var(--text-inverse);
  background:
    radial-gradient(circle at top left, rgba(182, 79, 97, 0.28), transparent 28%),
    linear-gradient(160deg, rgba(10, 25, 20, 0.84), rgba(19, 53, 42, 0.76));
  box-shadow: var(--shadow-lg);
}

.empty-badge {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(248, 242, 234, 0.82);
}

.empty-state h3 {
  margin: 18px 0 12px;
  font-size: 38px;
}

.empty-state p {
  margin: 0;
  line-height: 1.8;
  color: rgba(248, 242, 234, 0.78);
}

.empty-tags {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 24px;
}

.empty-tags span {
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
}

.message-card {
  max-width: min(780px, 82%);
  padding: 18px 20px;
  border-radius: 26px;
  box-shadow: var(--shadow-lg);
}

.user-card {
  align-self: flex-end;
  color: var(--text-inverse);
  background: linear-gradient(135deg, rgba(182, 79, 97, 0.96), rgba(184, 113, 88, 0.88));
}

.assistant-card {
  align-self: flex-start;
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.98), rgba(244, 239, 230, 0.92));
}

.message-role {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.message-body {
  line-height: 1.8;
  word-break: break-word;
}

.message-body code {
  padding: 2px 8px;
  border-radius: 8px;
  background: rgba(18, 53, 42, 0.08);
}

.message-status {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-soft);
}

.message-status.error {
  color: var(--plum-deep);
}

.composer-shell {
  padding: 20px;
  border-radius: var(--radius-xl);
  background: linear-gradient(180deg, rgba(248, 242, 234, 0.97), rgba(244, 239, 230, 0.92));
}

.composer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.composer-head span {
  font-weight: 700;
}

.composer-head p {
  margin: 0;
  color: var(--text-soft);
}

.composer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 14px;
}

.composer textarea {
  min-height: 120px;
  resize: vertical;
  padding: 18px 20px;
  border: 1px solid rgba(24, 46, 39, 0.1);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.76);
  color: var(--text-strong);
}

.composer textarea:focus {
  outline: none;
  border-color: rgba(182, 79, 97, 0.35);
  box-shadow: 0 0 0 4px rgba(182, 79, 97, 0.08);
}

.send-btn {
  min-width: 110px;
  padding: 0 24px;
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--plum), #b87158);
}

.ghost-btn:disabled,
.send-btn:disabled,
.select-input:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

@media (max-width: 1100px) {
  .chat-page {
    grid-template-columns: 1fr;
  }

  .workspace-header {
    flex-direction: column;
  }

  .workspace-actions {
    justify-content: flex-start;
  }
}

@media (max-width: 760px) {
  .chat-page {
    padding: 12px;
  }

  .message-card {
    max-width: 100%;
  }

  .composer {
    grid-template-columns: 1fr;
  }

  .send-btn {
    height: 50px;
  }

  .composer-head {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
