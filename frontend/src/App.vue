<template>
  <div id="app">
    <router-view />

    <!-- 免责声明弹窗，每次打开都显示 -->
    <transition name="fade">
      <div v-if="showDisclaimer" class="modal-mask" @click.self="confirm">
        <transition name="pop">
          <div class="modal" v-if="showDisclaimer">
            <div class="modal-hd">
              <div class="modal-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                  <line x1="12" y1="9" x2="12" y2="13"/>
                  <line x1="12" y1="17" x2="12.01" y2="17"/>
                </svg>
              </div>
              <span class="modal-title">使用须知</span>
            </div>
            <div class="modal-body">
              <div class="notice-section warn">
                <ul class="notice-list">
                  <li>本工具仅供<span class="em">技术开发、调试与测试</span>使用</li>
                  <li>禁止用于打卡、考勤等违反制度的行为</li>
                  <li>禁止用于游戏外挂、虚假位置上报</li>
                  <li>禁止用于欺诈、诈骗等违法犯罪活动</li>
                  <li>禁止用于绕过基于位置的风控系统</li>
                </ul>
              </div>
              <p class="notice-footer">
                <span class="highlight">违规使用产生的一切法律责任由使用者自行承担</span>，本平台不承担任何连带责任。
              </p>
              <button class="btn-primary" @click="confirm">我已知晓，继续使用</button>
            </div>
          </div>
        </transition>
      </div>
    </transition>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return { showDisclaimer: true }
  },
  methods: {
    confirm() { this.showDisclaimer = false }
  }
}
</script>

<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue", sans-serif;
  background: #f2f2f7;
  color: #1c1c1e;
  -webkit-tap-highlight-color: transparent;
}
#app { min-height: 100vh; }

.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 0 16px;
}

.modal {
  width: 100%;
  max-width: 360px;
  background: var(--bg, #1c1c1e);
  border-radius: 14px;
  padding: 20px 18px 18px;
  border: 1px solid var(--border, #38383a);
  margin: 0 auto;
  box-shadow: 0 16px 48px rgba(0,0,0,.5);
}

.modal-hd {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border, #38383a);
  margin-bottom: 14px;
}

.modal-icon {
  width: 32px; height: 32px;
  border-radius: 8px;
  background: rgba(255,159,28,.12);
  color: #ff9f1c;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text, #fff);
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.notice-section {
  padding: 12px 14px;
  background: var(--bg2, #2c2c2e);
  border-radius: 10px;
  border: 1px solid var(--border, #38383a);
}
.notice-section.warn {
  border-color: rgba(255,159,28,.25);
  background: rgba(255,159,28,.05);
}

.notice-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.notice-list li {
  font-size: 13px;
  line-height: 1.8;
  color: var(--text3, #98989f);
  padding-left: 14px;
  position: relative;
}
.notice-list li::before {
  content: "·";
  position: absolute;
  left: 4px;
  color: var(--text4, #636366);
  font-weight: bold;
  font-size: 16px;
  line-height: 1.6;
}

.notice-list .em {
  color: var(--text, #fff);
  font-weight: 500;
}

.notice-footer {
  font-size: 12px;
  line-height: 1.7;
  color: var(--text3, #98989f);
  text-align: justify;
  padding: 0 2px;
}

.notice-footer .highlight {
  color: var(--text, #fff);
  font-weight: 500;
}

/* 过渡动画 */
.fade-enter-active, .fade-leave-active { transition: opacity .25s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.pop-enter-active { transition: transform .3s cubic-bezier(.32,.72,0,1), opacity .25s ease; }
.pop-leave-active { transition: transform .2s ease, opacity .15s ease; }
.pop-enter-from { transform: scale(.92); opacity: 0; }
.pop-leave-to { transform: scale(.96); opacity: 0; }
</style>
