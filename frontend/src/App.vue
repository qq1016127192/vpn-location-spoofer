<template>
  <div id="app">
    <router-view />

    <!-- 免责声明弹窗，每次打开都显示 -->
    <div v-if="showDisclaimer" class="modal-mask" @click.self="confirm">
      <div class="modal">
        <div class="modal-hd">
          <span class="modal-title">使用须知</span>
        </div>
        <div class="modal-body">
          <p class="modal-text">
            本平台仅作<span class="highlight">技术测试、程序调试</span>使用，严禁用于各类违法违规活动；禁止擅自篡改、逆向各类 App 与网站程序，<span class="highlight">违规使用一切法律责任由使用者自行承担</span>。
          </p>
          <button class="btn-primary" @click="confirm">我已知晓，继续使用</button>
        </div>
      </div>
    </div>
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
  align-items: flex-end;
  justify-content: center;
  z-index: 9999;
}

.modal {
  width: 100%;
  background: var(--bg, #0a0a0a);
  border-radius: 16px 16px 0 0;
  padding: 0 16px max(env(safe-area-inset-bottom, 0px), 20px);
  border-top: 1px solid var(--border, #222);
}

.modal-hd {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px 0 14px;
  border-bottom: 1px solid var(--border, #222);
}

.modal-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text, #d0d0d0);
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 18px 0 4px;
}

.modal-text {
  font-size: 14px;
  line-height: 1.75;
  color: var(--text2, #a0a0a0);
}

.modal-text .highlight {
  color: var(--text, #d0d0d0);
  font-weight: 500;
}
</style>
