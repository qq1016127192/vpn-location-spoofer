<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script>
export default { name: 'App' }
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
</style>
