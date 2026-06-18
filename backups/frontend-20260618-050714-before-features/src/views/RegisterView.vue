<template>
  <div class="page">
    <div class="nav">
      <button class="back" @click="$router.back()">‹</button>
      <span class="nav-title">注册账号</span>
    </div>

    <div class="group form-group">
      <div class="field">
        <input v-model="email" type="email" placeholder="邮箱" @blur="vEmail" autocomplete="email" />
        <div class="err" v-if="eEmail">{{ eEmail }}</div>
      </div>
      <div class="field">
        <input v-model="password" type="password" placeholder="密码（至少 6 位）" @blur="vPass" />
        <div class="err" v-if="ePass">{{ ePass }}</div>
      </div>
      <div class="field">
        <input v-model="confirm" type="password" placeholder="确认密码" @blur="vConfirm" />
        <div class="err" v-if="eConfirm">{{ eConfirm }}</div>
      </div>
    </div>

    <button class="btn-primary" @click="submit" :disabled="loading">{{ loading ? '注册中…' : '注册' }}</button>
    <p class="tip">已有账号 <span class="link" @click="$router.push('/vpn-download')">去下载配置</span></p>
  </div>
</template>

<script>
import { vpnAPI } from '../api'
export default {
  name: 'RegisterView',
  data: () => ({ email:'', password:'', confirm:'', eEmail:'', ePass:'', eConfirm:'', loading:false }),
  methods: {
    vEmail() { this.eEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(this.email) ? '' : '邮箱格式错误' },
    vPass() { this.ePass = this.password.length >= 6 ? '' : '密码至少 6 位'; this.vConfirm() },
    vConfirm() { this.eConfirm = this.confirm === this.password ? '' : '两次密码不一致' },
    async submit() {
      this.vEmail(); this.vPass(); this.vConfirm()
      if (this.eEmail || this.ePass || this.eConfirm) return
      this.loading = true
      try { await vpnAPI.register({ email: this.email, password: this.password }); this.$router.push('/vpn-download') }
      catch { alert('注册失败') }
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
.form-group .field:first-child input { border-radius: 12px 12px 0 0; }
.form-group .field:last-child input { border-radius: 0 0 12px 12px; }
.tip { text-align: center; font-size: 14px; color: var(--text4); }
.link { color: var(--text3); cursor: pointer; text-decoration: underline; text-decoration-color: var(--border); }
</style>
