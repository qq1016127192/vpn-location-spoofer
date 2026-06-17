<template>
  <div class="map-view">
    <div id="map" ref="mapEl"></div>

    <!-- 准星：固定在视口正中 -->
    <div class="crosshair" aria-hidden="true">+</div>

    <!-- 顶部栏（考虑刘海屏） -->
    <div class="top-bar">
      <button class="nav-back" @click="$router.back()">‹</button>
      <div class="search-wrap" :class="{ focus: focused }">
        <input
          ref="inp"
          v-model="query"
          placeholder="搜索地点或输入坐标"
          @focus="focused = true"
          @blur="focused = false"
          @input="onInput"
          @keydown.enter.prevent="doSearch"
          autocomplete="off"
        />
        <button v-if="query" class="search-clear" @mousedown.prevent @click="query='';results=[]">✕</button>
      </div>
      <button class="nav-fav" @click="showPanel = true">☆</button>
    </div>

    <!-- 搜索结果列表 -->
    <div class="search-results" v-if="results.length">
      <div
        class="sr-row"
        v-for="(r,i) in results" :key="i"
        @mousedown.prevent
        @click="pick(r)"
      >
        <span class="sr-icon">▸</span>
        <div class="sr-info">
          <div class="sr-name">{{ r.name }}</div>
          <div class="sr-addr">{{ r.address }}</div>
        </div>
      </div>
    </div>

    <!-- 底部操作栏（考虑 Home 指示条） -->
    <div class="bottom-bar">
      <div class="coord-line">
        <span class="coord-val">{{ lat.toFixed(6) }},{{ lng.toFixed(6) }}</span>
        <button class="fav-toggle" @click="addFav">{{ isFaved ? '★' : '☆' }}</button>
      </div>
      <button
        class="lock-btn"
        :class="{ locked }"
        @click="lockLoc"
        :disabled="locking"
      >{{ locking ? '锁定中' : locked ? '已锁定' : '锁定此位置' }}</button>
    </div>

    <!-- 收藏/历史抽屉 -->
    <div class="drawer-mask" v-if="showPanel" @click="showPanel = false"></div>
    <transition name="drawer">
      <div class="drawer" v-if="showPanel">
        <div class="drawer-hd">
          <span>收藏 / 历史</span>
          <button class="drawer-close" @click="showPanel = false">✕</button>
        </div>

        <template v-if="favs.length">
          <div class="drawer-label">收藏</div>
          <div class="drawer-group">
            <div class="drawer-row" v-for="(f,i) in favs" :key="'f'+i" @click="jump(f)">
              <div class="dr-info">
                <div class="dr-name">{{ f.name }}</div>
                <div class="dr-coord">{{ f.lat.toFixed(5) }}, {{ f.lng.toFixed(5) }}</div>
              </div>
              <button class="dr-del" @click.stop="delFav(i)">✕</button>
            </div>
          </div>
        </template>

        <template v-if="hist.length">
          <div class="drawer-label">历史记录</div>
          <div class="drawer-group">
            <div class="drawer-row" v-for="(h,i) in hist" :key="'h'+i" @click="jump(h)">
              <div class="dr-info">
                <div class="dr-name">{{ h.name }}</div>
                <div class="dr-coord">{{ h.lat.toFixed(5) }}, {{ h.lng.toFixed(5) }}</div>
              </div>
              <button class="dr-del" @click.stop="delHist(i)">✕</button>
            </div>
          </div>
          <button class="drawer-clear" @click="clearHist">清除历史记录</button>
        </template>

        <div class="drawer-empty" v-if="!favs.length && !hist.length">暂无记录</div>
      </div>
    </transition>
  </div>
</template>

<script>
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { locationAPI } from '../api'

export default {
  name: 'MapView',
  data: () => ({
    map: null,
    lat: 39.90923, lng: 116.397428,
    locked: false, locking: false,
    query: '', results: [], focused: false, timer: null,
    showPanel: false,
    mapLoaded: false,
    favs: JSON.parse(localStorage.getItem('map_favs') || '[]'),
    hist: JSON.parse(localStorage.getItem('map_hist') || '[]')
  }),
  computed: {
    isFaved() {
      return this.favs.some(f => Math.abs(f.lat - this.lat) < 0.0001 && Math.abs(f.lng - this.lng) < 0.0001)
    }
  },
  mounted() { this.initMap() },
  beforeUnmount() { this.map?.remove() },
  methods: {
    initMap() {
      try {
        this.map = L.map(this.$refs.mapEl, {
          center: [this.lat, this.lng], zoom: 15, zoomControl: false
        })
        // 使用高德地图瓦片
        L.tileLayer('https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}', {
          subdomains: ['1', '2', '3', '4'],
          attribution: '© 高德地图',
          maxZoom: 18
        }).addTo(this.map)
        L.control.zoom({ position: 'bottomright' }).addTo(this.map)
        this.map.on('move', () => {
          const c = this.map.getCenter()
          this.lat = c.lat; this.lng = c.lng; this.locked = false
        })
        this.mapLoaded = true
      } catch (e) {
        console.error('地图初始化失败:', e)
        this.mapLoaded = false
      }
    },
    onInput() {
      clearTimeout(this.timer)
      if (!this.query.trim()) { this.results = []; return }
      this.timer = setTimeout(this.doSearch, 500)
    },
    async doSearch() {
      if (!this.query.trim()) return

      // 支持直接输入坐标格式：纬度,经度（如 39.9092,116.3975）
      const coordMatch = this.query.trim().match(/^(-?\d+\.?\d*)[,，\s]+(-?\d+\.?\d*)$/)
      if (coordMatch) {
        const lat = parseFloat(coordMatch[1])
        const lng = parseFloat(coordMatch[2])
        if (lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180) {
          this.results = [{ name: `${lat.toFixed(5)}, ${lng.toFixed(5)}`, address: '直接跳转到该坐标', lat, lng }]
          return
        }
      }

      try {
        const res = await fetch(`/search?q=${encodeURIComponent(this.query)}`)
        const data = await res.json()
        // Photon 返回 GeoJSON FeatureCollection
        if (data.features) {
          this.results = data.features.slice(0, 6).map(f => {
            const p = f.properties
            const name = p.name || p.city || p.street || this.query
            const parts = [p.city, p.state, p.country].filter(Boolean)
            return {
              name,
              address: parts.join(', ') || name,
              lat: f.geometry.coordinates[1],
              lng: f.geometry.coordinates[0]
            }
          })
        } else {
          this.results = []
        }
      } catch { this.results = [] }
    },
    pick(r) {
      this.query = r.name; this.results = []; this.focused = false
      this.map.setView([r.lat, r.lng], 15)
    },
    async lockLoc() {
      this.locking = true
      try {
        await locationAPI.setSpoofLocation({ latitude: this.lat, longitude: this.lng })
        this.locked = true
        this.pushHist({ lat: this.lat, lng: this.lng, name: `${this.lat.toFixed(5)},${this.lng.toFixed(5)}` })
      } catch { alert('锁定失败，请确认 VPN 已连接') }
      finally { this.locking = false }
    },
    addFav() {
      if (this.isFaved) return
      const name = prompt('收藏名称', `${this.lat.toFixed(4)},${this.lng.toFixed(4)}`)
      if (!name) return
      this.favs.unshift({ lat: this.lat, lng: this.lng, name })
      localStorage.setItem('map_favs', JSON.stringify(this.favs))
    },
    delFav(i) { this.favs.splice(i, 1); localStorage.setItem('map_favs', JSON.stringify(this.favs)) },
    pushHist(item) {
      this.hist = [item, ...this.hist.filter(h => Math.abs(h.lat - item.lat) > 0.0001)].slice(0, 20)
      localStorage.setItem('map_hist', JSON.stringify(this.hist))
    },
    delHist(i) { this.hist.splice(i, 1); localStorage.setItem('map_hist', JSON.stringify(this.hist)) },
    clearHist() { this.hist = []; localStorage.removeItem('map_hist') },
    jump(item) { this.map.setView([item.lat, item.lng], 15); this.showPanel = false }
  }
}
</script>

<style scoped>
/* ─── 容器 ────────────────────────────────────────── */
.map-view { position: fixed; inset: 0; background: var(--bg); }
#map { position: absolute; inset: 0; z-index: 1; }

/* 深色模式地图滤镜 */
@media (prefers-color-scheme: dark) {
  #map :deep(.leaflet-tile-pane) {
    filter: invert(1) hue-rotate(180deg) brightness(.82) contrast(.88) saturate(.15);
  }
}
#map :deep(.leaflet-control-zoom a) {
  background: var(--bg2) !important; color: var(--text3) !important; border-color: var(--border) !important;
}
#map :deep(.leaflet-attribution-flag) { display: none; }
#map :deep(.leaflet-control-attribution) { background: transparent !important; color: var(--text5) !important; font-size: 9px !important; }

/* 准星 */
.crosshair {
  position: absolute; left: 50%; top: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none; z-index: 500;
  font-size: 32px; font-weight: 700; line-height: 1;
  font-family: monospace;
  color: #ff3b30;
  text-shadow: 0 0 8px rgba(255,59,48,0.5);
}

/* 顶部栏 */
.top-bar {
  position: absolute; top: 0; left: 0; right: 0;
  padding-top: max(var(--safe-top), 10px);
  padding-bottom: 10px; padding-left: 12px; padding-right: 12px;
  background: var(--bar-bg, rgba(10,10,10,.93));
  border-bottom: 1px solid var(--border);
  display: flex; align-items: center; gap: 8px;
  z-index: 600;
  backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
}
@media (prefers-color-scheme: light) {
  .top-bar { --bar-bg: rgba(242,242,247,.93); }
}

.nav-back { background: none; border: none; font-size: 32px; color: var(--text3); cursor: pointer; line-height: 1; flex-shrink: 0; padding: 0; }
.nav-fav { background: none; border: none; font-size: 18px; color: var(--text4); cursor: pointer; flex-shrink: 0; padding: 2px 4px; }

.search-wrap {
  flex: 1; display: flex; align-items: center;
  background: var(--bg3); border: 1px solid var(--border);
  border-radius: 8px; padding: 8px 10px;
  transition: border-color .2s;
}
.search-wrap.focus { border-color: var(--text4); }
.search-wrap input {
  flex: 1; border: none; background: transparent;
  font-size: 16px; color: var(--text); outline: none;
}
.search-wrap input::placeholder { color: var(--text5); }
.search-clear { background: none; border: none; font-size: 11px; color: var(--text4); cursor: pointer; padding: 0 2px; }

/* 搜索结果 */
.search-results {
  position: absolute;
  top: calc(max(var(--safe-top), 10px) + 54px);
  left: 12px; right: 12px;
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: 10px; z-index: 700;
  max-height: 44vh; overflow-y: auto;
}
.sr-row { display: flex; align-items: flex-start; gap: 10px; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; }
.sr-row:last-child { border-bottom: none; }
.sr-row:active { background: var(--bg3); }
.sr-icon { font-size: 10px; color: var(--text4); flex-shrink: 0; margin-top: 3px; }
.sr-name { font-size: 14px; color: var(--text2); font-weight: 500; }
.sr-addr { font-size: 12px; color: var(--text4); margin-top: 2px; line-height: 1.4; }

/* 底部栏 */
.bottom-bar {
  position: absolute; bottom: 0; left: 0; right: 0;
  padding-bottom: max(var(--safe-bottom), 10px);
  padding-top: 12px; padding-left: 14px; padding-right: 14px;
  background: var(--bar-bg, rgba(10,10,10,.93));
  border-top: 1px solid var(--border);
  display: flex; flex-direction: column; gap: 10px;
  z-index: 600;
  backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
}
@media (prefers-color-scheme: light) {
  .bottom-bar { --bar-bg: rgba(242,242,247,.95); }
}

.coord-line { display: flex; align-items: center; }
.coord-val { flex: 1; font-size: 12px; font-family: "SF Mono","Menlo",monospace; color: var(--text4); letter-spacing: .3px; }
.fav-toggle { background: none; border: none; font-size: 16px; color: var(--text4); cursor: pointer; padding: 2px 4px; }

.lock-btn {
  width: 100%; padding: 13px;
  background: transparent;
  color: var(--text2);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 15px; font-weight: 500; cursor: pointer;
  letter-spacing: .2px;
}
.lock-btn:active:not(:disabled) { background: var(--bg2); }
.lock-btn.locked {
  background: var(--bg2);
  color: var(--text3);
  border-color: var(--border);
}
.lock-btn:disabled { color: var(--text5); border-color: var(--border2); cursor: not-allowed; }

/* 抽屉 */
.drawer-mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 800; }
.drawer {
  position: fixed; left: 0; right: 0; bottom: 0;
  max-height: 70vh; background: var(--bg);
  border-top: 1px solid var(--border);
  border-radius: 14px 14px 0 0;
  z-index: 900; overflow-y: auto;
  padding: 0 14px max(var(--safe-bottom), 16px);
}
.drawer-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 0 10px; position: sticky; top: 0;
  background: var(--bg); border-bottom: 1px solid var(--border);
  font-size: 15px; font-weight: 600; color: var(--text2);
}
.drawer-close { background: none; border: none; font-size: 13px; color: var(--text4); cursor: pointer; padding: 4px 6px; }
.drawer-label { font-size: 10px; color: var(--text5); text-transform: uppercase; letter-spacing: .8px; padding: 12px 0 6px; }
.drawer-group { background: var(--bg2); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; margin-bottom: 4px; }
.drawer-row { display: flex; align-items: center; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; }
.drawer-row:last-child { border-bottom: none; }
.drawer-row:active { background: var(--bg3); }
.dr-info { flex: 1; }
.dr-name { font-size: 14px; color: var(--text2); }
.dr-coord { font-size: 11px; color: var(--text4); font-family: monospace; margin-top: 2px; }
.dr-del { background: none; border: none; font-size: 12px; color: var(--text5); cursor: pointer; padding: 4px 6px; }
.drawer-clear { width: 100%; padding: 12px; background: none; border: none; color: var(--text4); font-size: 13px; cursor: pointer; text-align: center; }
.drawer-empty { text-align: center; color: var(--text5); font-size: 13px; padding: 32px 0; }

.drawer-enter-active, .drawer-leave-active { transition: transform .25s ease; }
.drawer-enter-from, .drawer-leave-to { transform: translateY(100%); }
</style>
