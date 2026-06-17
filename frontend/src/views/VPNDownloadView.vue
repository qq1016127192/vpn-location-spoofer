<template>
  <div class="page">
    <div class="nav">
      <button class="back" @click="$router.back()">‹</button>
      <span class="nav-title">下载配置</span>
    </div>

    <button class="btn-primary" @click="download" :disabled="loading">
      {{ loading ? '生成中…' : '下载 VPN 配置文件' }}
    </button>

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
import { vpnAPI } from '../api'
export default {
  name: 'VPNDownloadView',
  data: () => ({
    loading: false,
    steps: ['设置 → 通用 → VPN与设备管理 → 安装描述文件', '设置 → 关于本机 → 证书信任设置 → 开启信任', '设置 → VPN → 连接']
  }),
  methods: {
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
</style>
