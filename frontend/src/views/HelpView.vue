<template>
  <div class="page">
    <div class="nav">
      <button class="back" @click="$router.back()">‹</button>
      <span class="nav-title">使用教程</span>
    </div>
    <div class="section-label">VPN 安装</div>
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
</template>

<script>
export default {
  name: 'HelpView',
  data: () => ({
    vpnSteps: [
      { title: '下载并安装描述文件', desc: '点击"下载配置"，Safari 会提示"此网站正尝试下载一个配置描述文件"，点允许，然后去 设置 → 通用 → VPN与设备管理 找到描述文件并安装。' },
      { title: '信任根证书', desc: '设置 → 通用 → 关于本机 → 证书信任设置，找到 wzgo.cc Virtual Location Root CA 并打开信任开关。安装描述文件后才会出现此选项。', important: true },
      { title: '连接 VPN', desc: '设置 → VPN，选择 wzgo.cc 虚拟定位，点击连接。VPN 已配置按需自动连接，断网恢复后会自动重连。' }
    ],
    locSteps: [
      { title: '锁定目标位置', desc: '连接 VPN 后进入地图，准星对准目标，点击锁定。' },
      { title: '刷新定位', desc: '设置 → 隐私 → 定位服务，关闭后重新打开。', important: true },
      { title: '验证效果', desc: '打开地图 App 查看当前位置是否为锁定位置。' },
      { title: '恢复位置', desc: '断开 VPN，关闭再打开定位服务即可恢复。' }
    ]
  })
}
</script>

<style scoped>
.page { min-height: 100vh; padding: 0 16px calc(var(--safe-bottom) + 32px); display: flex; flex-direction: column; gap: 10px; }
.nav { display: flex; align-items: center; gap: 8px; padding: calc(var(--safe-top) + 12px) 0 8px; }
.back { background: none; border: none; font-size: 32px; color: var(--text3); cursor: pointer; line-height: 1; padding: 0; }
.nav-title { font-size: 17px; font-weight: 600; color: var(--text); }
.section-label { font-size: 11px; color: var(--text4); text-transform: uppercase; letter-spacing: .8px; padding: 4px 2px 0; }
.step-row { display: flex; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border); }
.step-row:last-child { border-bottom: none; }
.idx { color: var(--text4); font-weight: 700; font-size: 12px; flex-shrink: 0; margin-top: 2px; width: 14px; }
.body { flex: 1; }
.st { font-size: 14px; font-weight: 500; color: var(--text2); margin-bottom: 3px; display: flex; align-items: center; gap: 6px; }
.sd { font-size: 13px; color: var(--text4); line-height: 1.5; }
.tag { background: var(--bg2); color: var(--text3); font-size: 10px; font-weight: 600; padding: 1px 5px; border-radius: 3px; border: 1px solid var(--border); }
</style>
