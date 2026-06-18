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
      <button class="nav-fav" @click="showPanel = true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2Z"/>
        </svg>
      </button>
      <button class="nav-locate" @click="locateUser" title="定位">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <circle cx="12" cy="12" r="3"/>
          <path d="M12 2v4M12 18v4M2 12h4M18 12h4"/>
        </svg>
      </button>
      <button class="nav-return" v-if="lastLockedLocation" @click="returnToLastLocked" title="返回上一次锁定">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 10h10a5 5 0 0 1 5 5v2"/>
          <path d="M3 10l6-6M3 10l6 6"/>
        </svg>
      </button>
    </div>

    <!-- 搜索结果列表 -->
    <div class="search-results" v-if="results.length || searching || (focused && !query && (searchHistory.length || hotPlaces.length))">
      <div v-if="searching" class="sr-loading">搜索中...</div>
      
      <!-- 热门地点 -->
      <template v-if="!query && focused && hotPlaces.length">
        <div class="sr-section-label">热门地点</div>
        <div
          class="sr-row"
          v-for="(p,i) in hotPlaces" :key="'hot'+i"
          @mousedown.prevent
          @click="pick(p)"
        >
          <span class="sr-icon">🔥</span>
          <div class="sr-info">
            <div class="sr-name">{{ p.name }}</div>
            <div class="sr-addr">{{ p.address }}</div>
          </div>
        </div>
      </template>
      
      <!-- 搜索历史 -->
      <template v-if="!query && focused && searchHistory.length">
        <div class="sr-section-label">搜索历史</div>
        <div
          class="sr-row"
          v-for="(h,i) in searchHistory" :key="'hist'+i"
          @mousedown.prevent
          @click="pick(h)"
        >
          <span class="sr-icon">🕐</span>
          <div class="sr-info">
            <div class="sr-name">{{ h.name }}</div>
            <div class="sr-addr">{{ h.address }}</div>
          </div>
        </div>
      </template>
      
      <!-- 搜索结果 -->
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
      
      <!-- 无结果提示 -->
      <div v-if="!searching && query && !results.length" class="sr-empty">
        未找到相关地点
      </div>
    </div>

    <!-- 地图图层切换 -->
    <div class="map-layer-control">
      <button class="layer-btn" :class="{ active: mapLayer === 'standard' }" @click="switchLayer('standard')">标准</button>
      <button class="layer-btn" :class="{ active: mapLayer === 'satellite' }" @click="switchLayer('satellite')">卫星</button>
    </div>
    <div v-if="mapError" class="map-error">
      <div class="error-content">
        <div class="error-icon">⚠️</div>
        <div class="error-text">地图加载失败，请检查网络连接</div>
        <button class="error-retry" @click="initMap">重试</button>
      </div>
    </div>

    <!-- 底部操作栏（考虑 Home 指示条） -->
    <div class="bottom-bar">
      <div class="coord-line">
        <span class="coord-val" @click="copyCoord" title="点击复制">{{ lat.toFixed(6) }},{{ lng.toFixed(6) }}</span>
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

        <div class="drawer-empty" v-if="!favs.length && !hist.length">
          <div class="empty-icon">📍</div>
          <div class="empty-text">暂无记录</div>
          <div class="empty-hint">搜索或锁定位置后自动保存</div>
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
      </div>
    </transition>

    <!-- 收藏弹窗 -->
    <div class="modal-mask" v-if="showFavModal" @click="showFavModal = false"></div>
    <transition name="modal">
      <div class="modal" v-if="showFavModal">
        <div class="modal-hd">
          <span class="modal-title">添加收藏</span>
          <button class="modal-close" @click="showFavModal = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <input 
              v-model="favName" 
              placeholder="请输入收藏名称（如：公司、家等）" 
              class="modal-input"
              @keydown.enter="confirmAddFav"
              ref="favInput"
            />
          </div>
          <button class="btn-primary" @click="confirmAddFav">确认收藏</button>
        </div>
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
    marker: null,
    tileLayer: null,
    mapLayer: 'standard',
    lastLockedLocation: null,
    currentLocationName: null,
    lat: 39.90923, lng: 116.397428,
    locked: false, locking: false,
    query: '', results: [], focused: false, timer: null,
    showPanel: false,
    showFavModal: false,
    favName: '',
    mapLoaded: false,
    mapError: false,
    searching: false,
    searchHistory: JSON.parse(localStorage.getItem('search_history') || '[]'),
    hotPlaces: [
      { name: '北京天安门', address: '北京市东城区', lat: 39.908722, lng: 116.397499 },
      { name: '上海外滩', address: '上海市黄浦区', lat: 31.239708, lng: 121.490817 },
      { name: '广州塔', address: '广州市海珠区', lat: 23.106480, lng: 113.324520 },
      { name: '深圳市民中心', address: '深圳市福田区', lat: 22.547778, lng: 114.085947 }
    ],
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
          this.lat = c.lat; this.lng = c.lng
          // 移动时清除当前地点名称
          this.currentLocationName = null
          // 移动时取消锁定状态
          if (this.locked) {
            this.locked = false
          }
          // 移动时移除标记
          if (this.marker) {
            this.map.removeLayer(this.marker)
            this.marker = null
          }
        }, { passive: true })
        this.mapLoaded = true
      } catch (e) {
        console.error('地图初始化失败:', e)
        this.mapLoaded = false
        this.mapError = true
      }
    },
    onInput() {
      clearTimeout(this.timer)
      if (!this.query.trim()) { this.results = []; return }
      this.searching = true
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
      finally { this.searching = false }
    },
    pick(r) {
      this.query = r.name; this.results = []; this.focused = false; this.searching = false
      this.map.setView([r.lat, r.lng], 15)
      // 保存当前地点名称
      this.currentLocationName = r.name
      // 添加到搜索历史，使用搜索结果中的名称
      this.addToSearchHistory(r)
    },
    async lockLoc() {
      this.locking = true
      try {
        await locationAPI.setSpoofLocation({ latitude: this.lat, longitude: this.lng })
        this.locked = true
        // 记录上一次锁定的位置
        this.lastLockedLocation = { lat: this.lat, lng: this.lng }
        
        // 使用当前地点名称或默认名称
        const locationName = this.currentLocationName || `位置 ${this.lat.toFixed(4)},${this.lng.toFixed(4)}`
        this.pushHist({ lat: this.lat, lng: this.lng, name: locationName })
        
        // 添加锁定标记
        if (this.marker) {
          this.map.removeLayer(this.marker)
        }
        this.marker = L.marker([this.lat, this.lng], {
          icon: L.divIcon({
            className: 'lock-marker',
            html: '📍',
            iconSize: [32, 32],
            iconAnchor: [16, 32]
          })
        }).addTo(this.map)
      } catch { alert('锁定失败，请确认 VPN 已连接') }
      finally { this.locking = false }
    },
    addFav() {
      if (this.isFaved) return
      // 使用当前地点名称作为默认值
      this.favName = this.currentLocationName || `位置 ${this.lat.toFixed(4)},${this.lng.toFixed(4)}`
      this.showFavModal = true
      this.$nextTick(() => {
        this.$refs.favInput?.focus()
      })
    },
    confirmAddFav() {
      if (!this.favName.trim()) return
      this.favs.unshift({ lat: this.lat, lng: this.lng, name: this.favName.trim() })
      localStorage.setItem('map_favs', JSON.stringify(this.favs))
      this.showFavModal = false
      this.favName = ''
    },
    delFav(i) {
      if (confirm('确定删除此收藏？')) {
        this.favs.splice(i, 1)
        localStorage.setItem('map_favs', JSON.stringify(this.favs))
      }
    },
    pushHist(item) {
      this.hist = [item, ...this.hist.filter(h => Math.abs(h.lat - item.lat) > 0.0001)].slice(0, 20)
      localStorage.setItem('map_hist', JSON.stringify(this.hist))
    },
    delHist(i) {
      if (confirm('确定删除此历史记录？')) {
        this.hist.splice(i, 1)
        localStorage.setItem('map_hist', JSON.stringify(this.hist))
      }
    },
    clearHist() { this.hist = []; localStorage.removeItem('map_hist') },
    copyCoord() {
      const coord = `${this.lat.toFixed(6)},${this.lng.toFixed(6)}`
      navigator.clipboard.writeText(coord).then(() => {
        // 显示复制成功提示
        const originalText = this.$el.querySelector('.coord-val').textContent
        this.$el.querySelector('.coord-val').textContent = '已复制'
        setTimeout(() => {
          this.$el.querySelector('.coord-val').textContent = originalText
        }, 1000)
      }).catch(() => {
        alert('复制失败')
      })
    },
    addToSearchHistory(item) {
      const historyItem = { name: item.name, address: item.address, lat: item.lat, lng: item.lng }
      this.searchHistory = [historyItem, ...this.searchHistory.filter(h => h.name !== item.name)].slice(0, 10)
      localStorage.setItem('search_history', JSON.stringify(this.searchHistory))
    },
    locateUser() {
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (position) => {
            const { latitude, longitude } = position.coords
            this.map.setView([latitude, longitude], 15)
            this.lat = latitude
            this.lng = longitude
          },
          (error) => {
            let errorMsg = '定位失败'
            if (error.code === 1) {
              errorMsg = '定位权限被拒绝，请在设置中开启定位权限'
            } else if (error.code === 2) {
              errorMsg = '无法获取位置信息，请检查GPS或网络连接'
            } else if (error.code === 3) {
              errorMsg = '定位超时，请重试'
            }
            alert(errorMsg)
          },
          { timeout: 10000, enableHighAccuracy: true }
        )
      } else {
        alert('您的浏览器不支持定位功能')
      }
    },
    switchLayer(layer) {
      this.mapLayer = layer
      if (this.tileLayer) {
        this.map.removeLayer(this.tileLayer)
      }
      
      if (layer === 'standard') {
        this.tileLayer = L.tileLayer('https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}', {
          subdomains: ['1', '2', '3', '4'],
          attribution: '© 高德地图',
          maxZoom: 18
        })
      } else {
        this.tileLayer = L.tileLayer('https://webst0{s}.is.autonavi.com/appmaptile?style=6&x={x}&y={y}&z={z}', {
          subdomains: ['1', '2', '3', '4'],
          attribution: '© 高德地图',
          maxZoom: 18
        })
      }
      this.tileLayer.addTo(this.map)
    },
    returnToLastLocked() {
      if (this.lastLockedLocation) {
        this.map.setView([this.lastLockedLocation.lat, this.lastLockedLocation.lng], 15)
        this.lat = this.lastLockedLocation.lat
        this.lng = this.lastLockedLocation.lng
        this.locked = true
        // 重新添加标记
        if (this.marker) {
          this.map.removeLayer(this.marker)
        }
        this.marker = L.marker([this.lastLockedLocation.lat, this.lastLockedLocation.lng], {
          icon: L.divIcon({
            className: 'lock-marker',
            html: '📍',
            iconSize: [32, 32],
            iconAnchor: [16, 32]
          })
        }).addTo(this.map)
      } else {
        alert('没有上一次的锁定位置记录')
      }
    },
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
  font-size: 36px; font-weight: 700; line-height: 1;
  font-family: monospace;
  color: #ff3b30;
  text-shadow: 0 0 12px rgba(255,59,48,0.6);
  animation: crosshair-pulse 2s ease-in-out infinite;
  letter-spacing: 8px;
}
@keyframes crosshair-pulse {
  0%, 100% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
  50% { opacity: 0.7; transform: translate(-50%, -50%) scale(1.1); }
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

.nav-back { background: none; border: none; font-size: 32px; color: var(--text3); cursor: pointer; line-height: 1; flex-shrink: 0; padding: 0; transition: transform .1s; }
.nav-back:active { transform: scale(0.9); }
.nav-fav { background: none; border: none; color: var(--text4); cursor: pointer; flex-shrink: 0; padding: 2px 4px; transition: transform .1s; display: flex; align-items: center; justify-content: center; }
.nav-fav:active { transform: scale(0.9); }
.nav-locate { background: none; border: none; color: var(--text4); cursor: pointer; flex-shrink: 0; padding: 2px 4px; transition: transform .1s; display: flex; align-items: center; justify-content: center; }
.nav-locate:active { transform: scale(0.9); }
.nav-return { background: none; border: none; color: var(--text4); cursor: pointer; flex-shrink: 0; padding: 2px 4px; transition: transform .1s; display: flex; align-items: center; justify-content: center; }
.nav-return:active { transform: scale(0.9); }

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
  left: 0; right: 0;
  margin: 0 12px;
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: 10px; z-index: 700;
  max-height: 44vh; overflow-y: auto;
}
.sr-row { display: flex; align-items: flex-start; gap: 10px; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; transition: background .15s; position: relative; overflow: hidden; }
.sr-row:last-child { border-bottom: none; }
.sr-row:active { background: var(--bg3); transform: scale(0.98); }
.sr-icon { font-size: 10px; color: var(--text4); flex-shrink: 0; margin-top: 3px; }
.sr-name { font-size: 14px; color: var(--text2); font-weight: 500; }
.sr-addr { font-size: 12px; color: var(--text4); margin-top: 2px; line-height: 1.4; }
.sr-loading { padding: 20px; text-align: center; color: var(--text4); font-size: 13px; }
.sr-section-label { font-size: 11px; color: var(--text5); text-transform: uppercase; letter-spacing: .8px; padding: 12px 13px 8px; }
.sr-empty { padding: 20px; text-align: center; color: var(--text5); font-size: 13px; }

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
.coord-val { flex: 1; font-size: 12px; font-family: "SF Mono","Menlo",monospace; color: var(--text4); letter-spacing: .3px; cursor: pointer; padding: 4px 8px; border-radius: 4px; transition: background .2s; }
.coord-val:active { background: var(--bg3); }
.fav-toggle { background: none; border: none; font-size: 16px; color: var(--text4); cursor: pointer; padding: 2px 4px; transition: transform .1s; }
.fav-toggle:active { transform: scale(0.9); }

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
  background: #34c759;
  color: white;
  border-color: #34c759;
  font-weight: 600;
}
.lock-btn:disabled { color: var(--text5); border-color: var(--border2); cursor: not-allowed; }

/* 抽屉 */
.drawer-mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 800; }
.drawer {
  position: fixed; left: 0; right: 0; bottom: 0;
  margin: 0 auto;
  width: 100%;
  max-width: 420px;
  max-height: 70vh; background: var(--bg);
  border-top: 1px solid var(--border);
  border-radius: 14px 14px 0 0;
  z-index: 900; overflow-y: auto;
  padding: 0 14px max(var(--safe-bottom), 16px);
}
/* 优化滚动条样式 */
.drawer::-webkit-scrollbar {
  width: 4px;
}
.drawer::-webkit-scrollbar-track {
  background: transparent;
}
.drawer::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}
.drawer::-webkit-scrollbar-thumb:hover {
  background: var(--text5);
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
.drawer-row { display: flex; align-items: center; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; transition: background .15s, transform .1s; }
.drawer-row:last-child { border-bottom: none; }
.drawer-row:active { background: var(--bg3); transform: scale(0.98); }
.dr-info { flex: 1; }
.dr-name { font-size: 14px; color: var(--text2); }
.dr-coord { font-size: 11px; color: var(--text4); font-family: monospace; margin-top: 2px; }
.dr-del { background: none; border: none; font-size: 12px; color: var(--text5); cursor: pointer; padding: 4px 6px; }
.drawer-clear { width: 100%; padding: 12px; background: none; border: none; color: var(--text4); font-size: 13px; cursor: pointer; text-align: center; transition: color .2s; }
.drawer-clear:active { color: var(--text3); }
.drawer-empty { text-align: center; color: var(--text5); font-size: 13px; padding: 32px 0; }
.empty-icon { font-size: 32px; margin-bottom: 12px; }
.empty-text { font-size: 14px; color: var(--text3); margin-bottom: 8px; }
.empty-hint { font-size: 12px; color: var(--text5); line-height: 1.5; }

.drawer-enter-active, .drawer-leave-active { transition: transform .25s ease; }
.drawer-enter-from, .drawer-leave-to { transform: translateY(100%); }

/* 地图层切换控件 */
.map-layer-control {
  position: absolute;
  right: 12px;
  top: calc(max(var(--safe-top), 10px) + 60px);
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 600;
}
.layer-btn {
  padding: 8px 12px;
  background: var(--bar-bg, rgba(10,10,10,.93));
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text3);
  font-size: 12px;
  cursor: pointer;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  transition: all .2s, transform .1s;
}
.layer-btn:active { background: var(--bg3); transform: scale(0.95); }
.layer-btn.active {
  background: var(--text2);
  color: var(--bg);
  border-color: var(--text2);
}
@media (prefers-color-scheme: light) {
  .map-layer-control .layer-btn { --bar-bg: rgba(242,242,247,.93); }
}

/* 地图错误提示 */
.map-error {
  position: fixed;
  inset: 0;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.error-content {
  text-align: center;
  padding: 32px;
}
.error-icon {
  font-size: 48px;
  margin-bottom: 16px;
}
.error-text {
  font-size: 15px;
  color: var(--text3);
  margin-bottom: 20px;
  line-height: 1.5;
}
.error-retry {
  padding: 12px 24px;
  background: var(--text2);
  color: var(--bg);
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: transform .1s, opacity .2s;
}
.error-retry:active { transform: scale(0.95); opacity: 0.8; }

/* 弹窗样式 */
.modal-mask {
  position: fixed; inset: 0;
  background: rgba(0,0,0,.45);
  z-index: 1000;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 0 16px;
}
.modal {
  position: relative;
  width: 100%;
  max-width: 360px;
  background: var(--bg);
  border-radius: 16px;
  z-index: 1001;
  padding: 0 16px 20px;
  border: 1px solid var(--border);
  margin-bottom: env(safe-area-inset-bottom, 20px);
  transition: transform 0.3s ease;
}
.modal-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 0 12px;
  border-bottom: 1px solid var(--border);
}
.modal-title { font-size: 17px; font-weight: 600; color: var(--text); }
.modal-close { background: none; border: none; font-size: 18px; color: var(--text4); cursor: pointer; padding: 4px 8px; }
.modal-body { display: flex; flex-direction: column; gap: 14px; padding: 18px 0 4px; }
.form-group { display: flex; flex-direction: column; gap: 8px; }
.modal-input {
  width: 100%;
  padding: 12px 14px;
  background: var(--bg3);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 15px;
  color: var(--text);
  outline: none;
  transition: border-color .2s;
}
.modal-input:focus { border-color: var(--text4); }
.modal-input::placeholder { color: var(--text5); }

.modal-enter-active, .modal-leave-active { transition: opacity .2s ease, transform .2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: translateY(20px); }

/* 锁定标记 */
.lock-marker {
  font-size: 28px;
  text-shadow: 0 2px 8px rgba(0,0,0,0.3);
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2));
}
</style>
