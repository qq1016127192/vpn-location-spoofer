<template>
  <div class="page">
    <div class="head">
      <div class="app-name">虚拟定位</div>
      <div class="app-sub">iOS 虚拟定位工具</div>
    </div>

    <!-- VPN 状态检测卡片 -->
    <div class="status-card" :class="{ connected: vpnConnected }" @click="checkVpn">
      <div class="status-dot" :class="{ active: vpnConnected, checking: checking }"></div>
      <div class="status-info">
        <div class="status-label">{{ statusLabel }}</div>
        <div class="status-desc">{{ statusDesc }}</div>
      </div>
      <div class="status-action" @click.stop="checkVpn">
        <span v-if="checking" class="loading-icon">⟳</span>
        <span v-else class="refresh-icon">↻</span>
      </div>
    </div>

    <div class="section-label">使用步骤</div>
    <div class="group">
      <div class="list-row" v-for="(s,i) in steps" :key="i" @click="toggleStep(i)">
        <span class="idx">{{ i+1 }}</span>
        <div class="row-content">
          <span class="row-text">{{ s.title }}<span v-if="s.important" class="tag">重要</span></span>
          <div class="row-desc" v-if="expandedSteps.includes(i)">{{ s.desc }}</div>
        </div>
        <span class="expand-icon">{{ expandedSteps.includes(i) ? '▼' : '▶' }}</span>
      </div>
    </div>

    <div class="btns">
      <button class="btn-primary" @click="$router.push('/map')">进入地图</button>
      <div class="row2">
        <button class="btn-ghost" @click="showDownload = true">下载配置</button>
        <button class="btn-ghost" @click="showHelp = true">使用教程</button>
      </div>
    </div>

    <!-- 下载配置弹窗 -->
    <div class="modal-mask" v-if="showDownload" @click="showDownload = false"></div>
    <transition name="modal">
      <div class="modal" v-if="showDownload">
        <div class="modal-hd">
          <span class="modal-title">下载配置</span>
          <button class="modal-close" @click="showDownload = false">✕</button>
        </div>
        <div class="modal-body">
          <!-- 未登录：登录/注册 -->
          <template v-if="!user">
            <div class="auth-tabs">
              <button :class="{ active: authTab === 'login' }" @click="authTab = 'login'">登录</button>
              <button :class="{ active: authTab === 'register' }" @click="authTab = 'register'">注册</button>
            </div>
            <div v-if="authTab === 'login'" class="auth-form">
              <input v-model="loginForm.username" placeholder="用户名" autocomplete="username" @keydown.enter="doLogin" />
              <input v-model="loginForm.password" type="password" placeholder="密码" autocomplete="current-password" @keydown.enter="doLogin" />
              <button class="btn-primary" @click="doLogin" :disabled="authLoading">{{ authLoading ? '登录中…' : '登录' }}</button>
            </div>
            <div v-else class="auth-form">
              <input v-model="regForm.username" placeholder="用户名（3位以上，字母数字下划线）" autocomplete="username" @keydown.enter="doRegister" />
              <input v-model="regForm.password" type="password" placeholder="密码（6位以上）" autocomplete="new-password" @keydown.enter="doRegister" />
              <button class="btn-primary" @click="doRegister" :disabled="authLoading">{{ authLoading ? '注册中…' : '注册' }}</button>
            </div>
            <div v-if="authError" class="auth-error">{{ authError }}</div>
          </template>

          <!-- 已登录：显示下载 -->
          <template v-else>
            <div class="user-bar">
              <span class="user-name">{{ user.username }}</span>
              <button class="logout-btn" @click="doLogout">退出</button>
            </div>
            <button class="btn-primary" @click="download" :disabled="loading">{{ loading ? '生成中…' : '下载网络配置文件' }}</button>
          </template>

          <div class="section-label">安装步骤</div>
          <div class="group">
            <div class="list-row" v-for="(s,i) in downloadSteps" :key="i">
              <span class="idx">{{ i+1 }}</span>
              <span class="row-text">{{ s }}</span>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- 使用教程弹窗 -->
    <div class="modal-mask" v-if="showHelp" @click="showHelp = false"></div>
    <transition name="modal">
      <div class="modal" v-if="showHelp">
        <div class="modal-hd">
          <span class="modal-title">使用教程</span>
          <button class="modal-close" @click="showHelp = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="section-label">网络配置安装</div>
          <div class="group">
            <div class="step-row" v-for="(s,i) in vpnSteps" :key="i">
              <span class="idx">{{ i+1 }}</span>
              <div class="body">
                <div class="st">{{ s.title }}<span v-if="s.important" class="tag">重要</span></div>
                <div class="sd">{{ s.desc }}</div>
              </div>
            </div>
          </div>
          <div class="section-label">虚拟定位</div>
          <div class="group">
            <div class="step-row" v-for="(s,i) in locSteps" :key="i">
              <span class="idx">{{ i+1 }}</span>
              <div class="body">
                <div class="st">{{ s.title }}<span v-if="s.important" class="tag">重要</span></div>
                <div class="sd">{{ s.desc }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- 注册弹窗已移除 -->
  </div>
</template>

<script>
import { vpnAPI, authAPI } from '../api'
export default {
  name: 'VPNView',
  data: () => ({
    showDownload: false,
    showHelp: false,
    loading: false,
    // 认证状态
    user: null,
    authTab: 'login',
    authLoading: false,
    authError: '',
    loginForm: { username: '', password: '' },
    regForm: { username: '', password: '' },
    // VPN 状态检测
    checking: false,
    vpnConnected: false,
    checkTimer: null,
    // 保存当前客户端IP，用于MapView查询自己的状态
    clientIP: '',
    vpnUser: '',
    steps: [
      { title: '下载配置', desc: '点击"下载配置"，Safari 弹出提示后点允许下载。' },
      { title: '安装描述文件', desc: '设置 → 通用 → VPN与设备管理，找到描述文件并安装。' },
      { title: '信任根证书', desc: '设置 → 通用 → 关于本机 → 证书信任设置，打开根证书信任。安装描述文件后才会出现此选项。', important: true },
      { title: '连接代理', desc: '设置 → VPN，选择 wzgo.cc 虚拟定位连接。代理会按需自动重连。' },
      { title: '锁定目标位置', desc: '启用代理后进入地图，准星对准目标，点击锁定。' },
      { title: '刷新定位', desc: '设置 → 隐私 → 定位服务，关闭后重新打开。', important: true },
      { title: '验证效果', desc: '打开地图 App 查看当前位置是否为锁定位置。' },
      { title: '恢复位置', desc: '断开代理，关闭再打开定位服务即可恢复。' }
    ],
    expandedSteps: [],
    downloadSteps: ['Safari 打开本页，点下载后允许下载描述文件', '设置 → 通用 → VPN与设备管理 → 安装描述文件', '设置 → 关于本机 → 证书信任设置 → 开启根证书信任', '设置 → VPN → 连接 wzgo.cc 虚拟定位'],
    vpnSteps: [
      { title: '下载并安装描述文件', desc: '点击"下载配置"，Safari 会提示"此网站正尝试下载一个配置描述文件"，点允许，然后去 设置 → 通用 → VPN与设备管理 找到描述文件并安装。' },
      { title: '信任根证书', desc: '设置 → 通用 → 关于本机 → 证书信任设置，找到根证书并打开信任开关。安装描述文件后才会出现此选项。', important: true },
      { title: '连接代理', desc: '设置 → VPN，选择 wzgo.cc 虚拟定位，点击连接。代理已配置按需自动连接，断网恢复后会自动重连。' }
    ],
    locSteps: [
      { title: '锁定目标位置', desc: '启用代理后进入地图，准星对准目标，点击锁定。' },
      { title: '刷新定位', desc: '设置 → 隐私 → 定位服务，关闭后重新打开。', important: true },
      { title: '验证效果', desc: '打开地图 App 查看当前位置是否为锁定位置。' },
      { title: '恢复位置', desc: '断开代理，关闭再打开定位服务即可恢复。' }
    ]
  }),
  mounted() {
    this.checkLogin()
    this.checkVpn()
    // 每 15 秒自动检测一次
    this.checkTimer = setInterval(() => this.checkVpn(true), 15000)
  },
  beforeUnmount() {
    if (this.checkTimer) clearInterval(this.checkTimer)
  },
  computed: {
    statusLabel() {
      if (this.checking) return '检测中…'
      return this.vpnConnected ? 'VPN 已连接' : 'VPN 未连接'
    },
    statusDesc() {
      if (this.checking) return '正在检测连接状态'
      if (this.vpnConnected) return '可以设置虚拟定位了'
      return '请先在 iOS 设置中连接 VPN'
    }
  },
  methods: {
    toggleStep(index) {
      const idx = this.expandedSteps.indexOf(index)
      if (idx > -1) { this.expandedSteps.splice(idx, 1) } else { this.expandedSteps.push(index) }
    },
    async checkVpn(silent = false) {
      if (this.checking) return
      this.checking = true
      try {
        // 检查连接时，先传递当前保存的clientIP（如果有）
        const res = await vpnAPI.checkConnection(this.clientIP)
        this.vpnConnected = res.data.connected
        this.clientIP = res.data.client_ip
        if (this.clientIP) {
          localStorage.setItem('vpn_client_ip', this.clientIP)
        }
        // 核心：保存 vpn_user 到 localStorage，供 MapView 使用
        // vpn_user 是多用户数据隔离的关键，比 clientIP 更稳定
        if (res.data.vpn_user) {
          this.vpnUser = res.data.vpn_user
          localStorage.setItem('vpn_user', this.vpnUser)
        }
      } catch (e) {
        if (!silent) {
          this.vpnConnected = false
          // 不要清除clientIP和vpnUser，保留上次的
        }
      } finally {
        this.checking = false
      }
    },
    async download() {
      this.loading = true
      try {
        const res = await vpnAPI.downloadConfig()
        const url = URL.createObjectURL(new Blob([res.data], { type: 'application/x-apple-aspen-config' }))
        const a = document.createElement('a'); a.href = url; a.download = 'vpn.mobileconfig'
        document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url)
      } catch { alert('下载失败，请稍后重试') }
      finally { this.loading = false }
    },
    async checkLogin() {
      try {
        const res = await authAPI.getProfile()
        this.user = res.data
      } catch { this.user = null }
    },
    async doLogin() {
      this.authError = ''
      if (!this.loginForm.username || !this.loginForm.password) { this.authError = '请填写用户名和密码'; return }
      this.authLoading = true
      try {
        const res = await authAPI.login(this.loginForm)
        this.user = res.data
      } catch (e) { this.authError = e.response?.data?.error || '登录失败' }
      finally { this.authLoading = false }
    },
    async doRegister() {
      this.authError = ''
      if (!this.regForm.username || !this.regForm.password) { this.authError = '请填写用户名和密码'; return }
      this.authLoading = true
      try {
        const res = await authAPI.register(this.regForm)
        this.user = res.data
      } catch (e) { this.authError = e.response?.data?.error || '注册失败' }
      finally { this.authLoading = false }
    },
    async doLogout() {
      try { await authAPI.logout() } catch {}
      this.user = null
    }
  }
}
</script>

<style scoped>
.page {
  min-height: 100vh;
  padding: calc(var(--safe-top) + 48px) 16px calc(var(--safe-bottom) + 32px);
  display: flex; flex-direction: column; gap: 20px;
}
.head { padding: 8px 0 4px; }
.app-name { font-size: 26px; font-weight: 700; color: var(--text); letter-spacing: -.3px; }
.app-sub { font-size: 13px; color: var(--text4); margin-top: 4px; }

/* VPN 状态卡片 */
.status-card {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 16px;
  background: var(--bg2);
  border: 1px solid var(--border);
  border-radius: 12px;
  cursor: pointer;
  transition: border-color .2s ease, transform .15s ease;
}
.status-card:active { transform: scale(.99); }
.status-card.connected { border-color: #34c759; }
.status-dot {
  width: 10px; height: 10px; border-radius: 50%;
  background: var(--text4);
  flex-shrink: 0;
  transition: background .3s ease;
}
.status-dot.active {
  background: #34c759;
  box-shadow: 0 0 0 4px rgba(52,199,89,.2);
  animation: pulse-green 2s ease-in-out infinite;
}
.status-dot.checking {
  background: var(--text3);
  animation: pulse-gray 1s ease-in-out infinite;
}
@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 0 4px rgba(52,199,89,.2); }
  50% { box-shadow: 0 0 0 8px rgba(52,199,89,.1); }
}
@keyframes pulse-gray {
  0%, 100% { opacity: 1; }
  50% { opacity: .4; }
}
.status-info { flex: 1; min-width: 0; }
.status-label { font-size: 15px; font-weight: 600; color: var(--text); }
.status-desc { font-size: 12px; color: var(--text3); margin-top: 2px; }
.status-action { flex-shrink: 0; padding: 4px; }
.refresh-icon, .loading-icon {
  font-size: 18px; color: var(--text3);
  display: inline-block;
}
.loading-icon { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.section-label { font-size: 11px; color: var(--text4); text-transform: uppercase; letter-spacing: .8px; padding: 0 2px; }
.list-row {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 13px 14px;
  border-bottom: 1px solid var(--border);
  font-size: 14px; color: var(--text2);
  cursor: pointer;
}
.list-row:last-child { border-bottom: none; }
.list-row:active { background: var(--bg3); }
.idx {
  width: 20px; height: 20px; border-radius: 50%;
  background: var(--bg3); border: 1px solid var(--border);
  color: var(--text3); font-size: 11px; font-weight: 600;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  margin-top: 2px;
}
.row-content { flex: 1; }
.row-text { font-size: 14px; color: var(--text2); display: flex; align-items: center; gap: 6px; }
.row-desc { font-size: 13px; color: var(--text4); margin-top: 6px; line-height: 1.5; }
.expand-icon { color: var(--text4); font-size: 10px; flex-shrink: 0; margin-top: 4px; }
.list-row .tag { background: var(--bg2); color: var(--text3); font-size: 10px; font-weight: 600; padding: 1px 5px; border-radius: 3px; border: 1px solid var(--border); }
.btns { display: flex; flex-direction: column; gap: 8px; margin-top: auto; }
.row2 { display: flex; gap: 8px; }
.row2 .btn-ghost { flex: 1; }

/* 弹窗样式 */
.modal-mask {
  position: fixed; inset: 0;
  background: rgba(0,0,0,.45);
  z-index: 1000;
}
.modal {
  position: fixed;
  left: 0;
  right: 0;
  margin: 0 auto;
  width: 100%;
  max-width: 420px;
  bottom: 0;
  max-height: 85vh;
  background: var(--bg);
  border-radius: 16px 16px 0 0;
  z-index: 1001;
  overflow-y: auto;
  padding: 0 16px max(var(--safe-bottom), 16px);
  display: flex; flex-direction: column;
}
.modal-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 0 12px;
  position: sticky; top: 0;
  background: var(--bg);
  z-index: 10;
}
.modal-title { font-size: 17px; font-weight: 600; color: var(--text); }
.modal-close { background: none; border: none; font-size: 18px; color: var(--text4); cursor: pointer; padding: 4px 8px; }
.modal-body { display: flex; flex-direction: column; gap: 14px; padding-bottom: 8px; }
.form-group .field:first-child input { border-radius: 12px 12px 0 0; }
.form-group .field:last-child input { border-radius: 0 0 12px 12px; }
.step-row { display: flex; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border); }
.step-row:last-child { border-bottom: none; }
.step-row .idx { color: var(--text4); font-weight: 700; font-size: 12px; flex-shrink: 0; margin-top: 2px; width: 14px; height: auto; background: none; border: none; }
.step-row .body { flex: 1; }
.step-row .st { font-size: 14px; font-weight: 500; color: var(--text2); margin-bottom: 3px; display: flex; align-items: center; gap: 6px; }
.step-row .sd { font-size: 13px; color: var(--text4); line-height: 1.5; }
.step-row .tag { background: var(--bg2); color: var(--text3); font-size: 10px; font-weight: 600; padding: 1px 5px; border-radius: 3px; border: 1px solid var(--border); }

.modal-enter-active, .modal-leave-active { transition: transform .25s ease; }
.modal-enter-from, .modal-leave-to { transform: translateY(100%); }
.tip { text-align: center; font-size: 14px; color: var(--text4); }
.link { color: var(--text3); cursor: pointer; text-decoration: underline; text-decoration-color: var(--border); }

/* 认证相关 */
.auth-tabs { display: flex; gap: 0; margin-bottom: 12px; }
.auth-tabs button { flex: 1; padding: 10px; background: none; border: 1px solid var(--border); color: var(--text4); font-size: 14px; cursor: pointer; transition: all .2s; }
.auth-tabs button:first-child { border-radius: 8px 0 0 8px; }
.auth-tabs button:last-child { border-radius: 0 8px 8px 0; }
.auth-tabs button.active { background: var(--text2); color: var(--bg); border-color: var(--text2); }
.auth-form { display: flex; flex-direction: column; gap: 10px; }
.auth-form input { padding: 12px 14px; background: var(--bg3); border: 1px solid var(--border); border-radius: 8px; font-size: 15px; color: var(--text); outline: none; }
.auth-form input::placeholder { color: var(--text5); }
.auth-error { margin-top: 8px; padding: 10px; background: rgba(255,59,48,.1); border-radius: 8px; font-size: 13px; color: #ff3b30; text-align: center; }
.user-bar { display: flex; align-items: center; gap: 10px; padding: 12px 14px; background: var(--bg2); border: 1px solid var(--border); border-radius: 10px; margin-bottom: 8px; }
.user-name { flex: 1; font-size: 15px; font-weight: 500; color: var(--text); }
.logout-btn { background: none; border: 1px solid var(--border); padding: 6px 12px; border-radius: 6px; font-size: 12px; color: var(--text4); cursor: pointer; }
</style>
