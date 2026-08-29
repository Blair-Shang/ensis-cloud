<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { RsButton, RsInput } from '@/components/ui'
import { ApiError, loginOps } from '@/api'

const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const markSrc = `${import.meta.env.BASE_URL}favicon.svg`

const canSubmit = computed(
  () => email.value.trim().includes('@') && password.value.length >= 10 && !loading.value,
)

function mapError(e: unknown): string {
  if (e instanceof ApiError) {
    switch (e.code) {
      case 'invalid_credentials':
        return '邮箱或密码不正确'
      case 'account_disabled':
        return '账号已停用，请联系管理员'
      case 'rate_limited':
        return '尝试过于频繁，请稍后再试'
      default:
        return e.code
    }
  }
  if (e instanceof Error) return e.message
  return '登录失败'
}

async function submit() {
  if (!canSubmit.value) return
  error.value = ''
  loading.value = true
  try {
    await loginOps(email.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (e) {
    error.value = mapError(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="login__glow" aria-hidden="true" />
    <div class="login__panel">
      <aside class="login__brand">
        <img class="login__mark" :src="markSrc" width="40" height="40" alt="" />
        <p class="login__eyebrow">NiuMa Cloud</p>
        <h1 class="login__title">Ops</h1>
        <p class="login__lead">账户、反馈与产品运营控制台。仅限内部员工。</p>
      </aside>

      <form class="login__form" @submit.prevent="submit">
        <header class="login__form-head">
          <h2>员工登录</h2>
          <p>使用员工邮箱与密码进入后台</p>
        </header>

        <div class="login__field">
          <span class="login__label">邮箱</span>
          <RsInput
            v-model="email"
            type="email"
            placeholder="name@niuma.com"
            autocomplete="username"
            required
            :disabled="loading"
          />
        </div>

        <div class="login__field">
          <span class="login__label">密码</span>
          <RsInput
            v-model="password"
            type="password"
            placeholder="员工密码"
            autocomplete="current-password"
            required
            :disabled="loading"
          />
        </div>

        <p v-if="error" class="login__error" role="alert">{{ error }}</p>

        <RsButton
          type="submit"
          variant="primary"
          class="login__submit"
          :loading="loading"
          :disabled="!canSubmit"
        >
          {{ loading ? '登录中…' : '进入控制台' }}
        </RsButton>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login {
  position: relative;
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  overflow: hidden;
  background:
    radial-gradient(900px 480px at 12% 0%, color-mix(in srgb, #5ac8fa 28%, transparent), transparent 60%),
    radial-gradient(700px 420px at 88% 100%, color-mix(in srgb, #30d158 14%, transparent), transparent 55%),
    #eef2f7;
}

.login__glow {
  pointer-events: none;
  position: absolute;
  inset: 18% 28% auto;
  height: 36%;
  background: radial-gradient(closest-side, color-mix(in srgb, #64d2ff 16%, transparent), transparent);
  filter: blur(36px);
}

.login__panel {
  position: relative;
  z-index: 1;
  width: min(820px, 100%);
  display: grid;
  grid-template-columns: 1.05fr 1fr;
  border: 1px solid #e2e8f0;
  border-radius: 1.15rem;
  background: #fff;
  box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
}

.login__brand {
  display: grid;
  align-content: center;
  gap: 0.65rem;
  padding: 2.4rem 2rem;
  border-radius: 1.15rem 0 0 1.15rem;
  background:
    linear-gradient(160deg, color-mix(in srgb, #5ac8fa 18%, transparent), transparent 58%),
    linear-gradient(200deg, #f8fafc, #eef3f8);
  border-right: 1px solid #e2e8f0;
}

.login__mark {
  border-radius: 0.65rem;
  margin-bottom: 0.35rem;
}

.login__eyebrow {
  margin: 0;
  font-size: 0.78rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #64748b;
}

.login__title {
  margin: 0;
  font-size: 2.4rem;
  line-height: 1;
  letter-spacing: -0.04em;
  font-weight: 750;
  color: #0f172a;
}

.login__lead {
  margin: 0.35rem 0 0;
  max-width: 22ch;
  color: #64748b;
  line-height: 1.55;
  font-size: 0.95rem;
}

.login__form {
  display: grid;
  gap: 1rem;
  align-content: center;
  padding: 2.25rem 2rem;
  border-radius: 0 1.15rem 1.15rem 0;
  background: #fff;
}

.login__form-head h2 {
  margin: 0;
  font-size: 1.2rem;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.login__form-head p {
  margin: 0.35rem 0 0.15rem;
  color: #64748b;
  font-size: 0.9rem;
}

.login__field {
  display: grid;
  gap: 0.45rem;
  /* 避免父级裁切 RsInput 边框 / focus ring */
  overflow: visible;
}

.login__label {
  font-size: 0.82rem;
  color: #64748b;
}

/* 保证输入框组边框完整可见，不被外层样式压住 */
.login__field :deep(.rs-input-field) {
  width: 100%;
}

.login__field :deep(.rs-input-group) {
  width: 100%;
  background: #fff;
  border: 1px solid #d0d7e2;
  border-radius: 0.5rem;
  box-shadow: none;
  /* autofill 会画超大 inset 矩形，必须裁进圆角内，否则四角边框「断线」 */
  overflow: hidden;
}

.login__field :deep(.rs-input-group:hover:not(.rs-input-group--disabled)) {
  border-color: #b6c0cf;
}

.login__field :deep(.rs-input-group:focus-within) {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px color-mix(in srgb, #2563eb 18%, transparent);
}

.login__field :deep(.rs-input-group__control:-webkit-autofill),
.login__field :deep(.rs-input-group__control:-webkit-autofill:hover),
.login__field :deep(.rs-input-group__control:-webkit-autofill:focus) {
  -webkit-box-shadow: 0 0 0 1000px #fff inset;
  box-shadow: 0 0 0 1000px #fff inset;
  -webkit-text-fill-color: #151921;
  caret-color: #151921;
}

.login__error {
  margin: 0;
  padding: 0.55rem 0.7rem;
  border-radius: 0.55rem;
  background: #fef2f2;
  color: #b91c1c;
  font-size: 0.88rem;
  border: 1px solid #fecaca;
}

.login__submit {
  margin-top: 0.25rem;
  width: 100%;
}

@media (max-width: 720px) {
  .login__panel {
    grid-template-columns: 1fr;
  }

  .login__brand {
    border-right: none;
    border-bottom: 1px solid #e2e8f0;
    border-radius: 1.15rem 1.15rem 0 0;
    padding: 1.6rem 1.4rem;
  }

  .login__form {
    border-radius: 0 0 1.15rem 1.15rem;
    padding: 1.5rem 1.4rem 1.75rem;
  }

  .login__title {
    font-size: 1.8rem;
  }
}
</style>
