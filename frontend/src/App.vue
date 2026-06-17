<template>
  <div id="app">
    <router-view />

    <!-- 免责声明弹窗，首次访问显示 -->
    <div v-if="showDisclaimer" class="disclaimer-overlay">
      <div class="disclaimer-box">
        <div class="disclaimer-icon">⚠️</div>
        <h2>使用须知</h2>
        <p>
          本平台仅作<strong>技术测试、程序调试</strong>使用，严禁用于各类违法违规活动；
          禁止擅自篡改、逆向各类 App 与网站程序，
          <strong>违规使用一切法律责任由使用者自行承担</strong>。
        </p>
        <button @click="confirm">我已知晓，继续使用</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      showDisclaimer: false
    }
  },
  mounted() {
    if (!localStorage.getItem('disclaimer_confirmed')) {
      this.showDisclaimer = true
    }
  },
  methods: {
    confirm() {
      localStorage.setItem('disclaimer_confirmed', '1')
      this.showDisclaimer = false
    }
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

.disclaimer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 20px;
}

.disclaimer-box {
  background: #fff;
  border-radius: 16px;
  padding: 28px 24px;
  max-width: 360px;
  width: 100%;
  text-align: center;
  box-shadow: 0 8px 40px rgba(0,0,0,0.2);
}

.disclaimer-icon {
  font-size: 36px;
  margin-bottom: 12px;
}

.disclaimer-box h2 {
  font-size: 18px;
  font-weight: 600;
  color: #1c1c1e;
  margin-bottom: 14px;
}

.disclaimer-box p {
  font-size: 14px;
  line-height: 1.7;
  color: #555;
  margin-bottom: 22px;
}

.disclaimer-box strong {
  color: #1c1c1e;
}

.disclaimer-box button {
  width: 100%;
  padding: 13px;
  background: #007aff;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
}

.disclaimer-box button:active {
  opacity: 0.85;
}
</style>
