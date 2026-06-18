<template>
  <div class="page">
    <div class="nav">
      <button class="back" @click="$router.back()">‹</button>
      <span class="nav-title">下载配置</span>
    </div>

    <!-- 未登录：显示登录/注册表单 -->
    <div v-if="!user" class="auth-card">
      <div class="auth-tabs">
        <button :class="{ active: tab === 'login' }" @click="tab = 'login'">登录</button>
        <button :class="{ active: tab === 'register' }" @click="tab = 'register'">注册</button>
      </div>

      <div v-if="tab === 'login'" class="auth-form">
        <input v-model="loginForm.username" placeholder="用户名" autocomplete="username" @keydown.enter="doLogin" />
        <input v-model="loginForm.password" type="password" placeholder="密码" autocomplete="current-password" @keydown.enter="doLogin" />
        <button class="btn-primary" @click="doLogin" :disabled="authLoading">
          {{ authLoading ? '登录中…' : '登录' }}
        </button>
      </div>

      <div v-else class="auth-form">
        <input v-model="regForm.username" placeholder="用户名（3位以上，字母数字下划线）" autocomplete="username" @keydown.enter="doRegister" />
        <input v-model="regForm.password" type="password" placeholder="密码（6位以上）" autocomplete="new-password" @keydown.enter="doRegister" />
        <button class="btn-primary" @click="doRegister" :disabled="authLoading">
          {{ authLoading ? '注册中…' : '注册' }}
        </button>
      </div>

      <div v-if="authError" class="auth-error">{{ authError }}</div>
    </div>

    <!-- 已登录：显示下载按钮 -->
    <div v-else>
      <div class="user-info">
        <span class="user-label">当前账号</span>
        <span class="user-name">{{ user.username }}</span>
        <button class="logout-btn" @click="doLogout">退出</button>
      </div>

      <button class="btn-primary" @click="download" :disabled="loading">
        {{ loading ? '生成中…' : '下载网络配置文件' }}
      </button>
    </div>

    <div class="section-label">安装步骤</div>
    <div class="group">
      <div class="list-row" v-for="(s,i) in steps" :key="i">
        <span class="idx">{{ i+1 }}</span>
        <span class="row-text">{{ s }}</span>
      </div>
    </div>
  </div>
</template>

<script>
import { authAPI, vpnAPI } from '../api'
export default {
  name: 'VPNDownloadView',
  data: () => ({
    user: null,
    tab: 'login',
    authLoading: false,
    authError: '',
    loginForm: { username: '', password: '' },
    regForm: { username: '', password: '' },
    loading: false,
    steps: ['设置 → 通用 → VPN与设备管理 → 安装描述文件', '设置 → 关于本机 → 证书信任设置 → 开启根证书信任', '设置 → 代理 → 连接']
  }),
  mounted() {
    this.checkLogin()
  },
  methods: {
    async checkLogin() {
      try {
        const res = await authAPI.getProfile()
        this.user = res.data
      } catch {
        this.user = null
      }
    },
    async doLogin() {
      this.authError = ''
      if (!this.loginForm.username || !this.loginForm.password) {
        this.authError = '请填写用户名和密码'
        return
      }
      this.authLoading = true
      try {
        const res = await authAPI.login(this.loginForm)
        this.user = res.data
      } catch (e) {
        this.authError = e.response?.data?.error || '登录失败'
      } finally {
        this.authLoading = false
      }
    },
    async doRegister() {
      this.authError = ''
      if (!this.regForm.username || !this.regForm.password) {
        this.authError = '请填写用户名和密码'
        return
      }
      this.authLoading = true
      try {
        const res = await authAPI.register(this.regForm)
        this.user = res.data
      } catch (e) {
        this.authError = e.response?.data?.error || '注册失败'
      } finally {
        this.authLoading = false
      }
    },
    async doLogout() {
      try { await authAPI.logout() } catch {}
      this.user = null
    },
    async download() {
      this.loading = true
      try {
        const res = await vpnAPI.downloadConfig()
        const url = URL.createObjectURL(new Blob([res.data], { type: 'application/x-apple-aspen-config' }))
        const a = document.createElement('a')
        a.href = url; a.download = 'vpn.mobileconfig'
        document.body.appendChild(a); a.click()
        document.body.removeChild(a); URL.revokeObjectURL(url)
      } catch { alert('下载失败，请稍后重试') }
      finally { this.loading = false }
    }
  }
}
</script>

<style scoped>
.page { min-height: 100vh; padding: 0 16px calc(var(--safe-bottom) + 32px); display: flex; flex-direction: column; gap: 14px; }
.nav { display: flex; align-items: center; gap: 8px; padding: calc(var(--safe-top) + 12px) 0 16px; }
.back { background: none; border: none; font-size: 32px; color: var(--text3); cursor: pointer; line-height: 1; padding: 0; }
.nav-title { font-size: 17px; font-weight: 600; color: var(--text); }
.section-label { font-size: 11px; color: var(--text4); text-transform: uppercase; letter-spacing: .8px; padding: 4px 2px 0; }
.list-row { display: flex; align-items: flex-start; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border); font-size: 13px; color: var(--text3); line-height: 1.5; }
.list-row:last-child { border-bottom: none; }
.idx { color: var(--text4); font-weight: 700; font-size: 12px; flex-shrink: 0; margin-top: 1px; }
.row-text { flex: 1; }

/* 认证卡片 */
.auth-card { background: var(--bg2); border: 1px solid var(--border); border-radius: 12px; padding: 16px; }
.auth-tabs { display: flex; gap: 0; margin-bottom: 16px; }
.auth-tabs button { flex: 1; padding: 10px; background: none; border: 1px solid var(--border); color: var(--text4); font-size: 14px; cursor: pointer; transition: all .2s; }
.auth-tabs button:first-child { border-radius: 8px 0 0 8px; }
.auth-tabs button:last-child { border-radius: 0 8px 8px 0; }
.auth-tabs button.active { background: var(--text2); color: var(--bg); border-color: var(--text2); }
.auth-form { display: flex; flex-direction: column; gap: 10px; }
.auth-form input { padding: 12px 14px; background: var(--bg3); border: 1px solid var(--border); border-radius: 8px; font-size: 15px; color: var(--text); outline: none; }
.auth-form input::placeholder { color: var(--text5); }
.auth-error { margin-top: 8px; padding: 10px; background: rgba(255,59,48,.1); border-radius: 8px; font-size: 13px; color: #ff3b30; text-align: center; }

/* 用户信息 */
.user-info { display: flex; align-items: center; gap: 10px; padding: 14px; background: var(--bg2); border: 1px solid var(--border); border-radius: 12px; }
.user-label { font-size: 12px; color: var(--text4); }
.user-name { flex: 1; font-size: 15px; font-weight: 500; color: var(--text); }
.logout-btn { background: none; border: 1px solid var(--border); padding: 6px 12px; border-radius: 6px; font-size: 12px; color: var(--text4); cursor: pointer; }
</style>