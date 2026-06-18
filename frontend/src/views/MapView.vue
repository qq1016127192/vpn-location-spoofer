<template>
  <div class="map-view">
    <div id="map" ref="mapEl"></div>

    <!-- 准星：SVG 圆环+十字 -->
    <div class="crosshair" aria-hidden="true">
      <svg width="44" height="44" viewBox="0 0 44 44">
        <circle cx="22" cy="22" r="14" fill="none" stroke="#ff3b30" stroke-width="1.5" opacity="0.9"/>
        <circle cx="22" cy="22" r="2" fill="#ff3b30"/>
        <line x1="22" y1="2" x2="22" y2="12" stroke="#ff3b30" stroke-width="1.5"/>
        <line x1="22" y1="32" x2="22" y2="42" stroke="#ff3b30" stroke-width="1.5"/>
        <line x1="2" y1="22" x2="12" y2="22" stroke="#ff3b30" stroke-width="1.5"/>
        <line x1="32" y1="22" x2="42" y2="22" stroke="#ff3b30" stroke-width="1.5"/>
      </svg>
    </div>

    <!-- 顶部栏 -->
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
      <button class="nav-fav" @click="showPanel = true" title="收藏/历史">
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
      <button class="nav-return" v-if="lastLockedLocation" @click="returnToLastLocked" title="返回上次锁定">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 10h10a5 5 0 0 1 5 5v2"/>
          <path d="M3 10l6-6M3 10l6 6"/>
        </svg>
      </button>
    </div>

    <!-- 搜索结果列表 -->
    <div class="search-results" v-if="results.length || searching || (focused && !query && (searchHistory.length || hotPlaces.length))">
      <div v-if="searching" class="sr-loading">搜索中...</div>

      <template v-if="!query && focused && hotPlaces.length">
        <div class="sr-section-label">热门地点</div>
        <div class="sr-row" v-for="(p,i) in hotPlaces" :key="'hot'+i" @mousedown.prevent @click="pick(p)">
          <span class="sr-icon">🔥</span>
          <div class="sr-info">
            <div class="sr-name">{{ p.name }}</div>
            <div class="sr-addr">{{ p.address }}</div>
          </div>
        </div>
      </template>

      <template v-if="!query && focused && searchHistory.length">
        <div class="sr-section-label">搜索历史</div>
        <div class="sr-row" v-for="(h,i) in searchHistory" :key="'hist'+i" @mousedown.prevent @click="pick(h)">
          <span class="sr-icon">🕐</span>
          <div class="sr-info">
            <div class="sr-name">{{ h.name }}</div>
            <div class="sr-addr">{{ h.address }}</div>
          </div>
        </div>
      </template>

      <div class="sr-row" v-for="(r,i) in results" :key="i" @mousedown.prevent @click="pick(r)">
        <span class="sr-icon">{{ typeIcon(r.type) }}</span>
        <div class="sr-info">
          <div class="sr-name">{{ r.name }}</div>
          <div class="sr-addr">
            {{ r.address }}
            <span v-if="r.dist" class="sr-dist">· {{ r.dist }}</span>
          </div>
        </div>
      </div>

      <div v-if="!searching && query && !results.length" class="sr-empty">未找到相关地点</div>
    </div>

    <!-- 右侧工具栏 -->
    <div class="map-tools">
      <button class="layer-btn" :class="{ active: mapLayer === 'standard' }" @click="switchLayer('standard')">标准</button>
      <button class="layer-btn" :class="{ active: mapLayer === 'satellite' }" @click="switchLayer('satellite')">卫星</button>
      <button class="tool-btn" @click="showCoordSwitcher = !showCoordSwitcher" :title="'坐标系: ' + coordSystem.toUpperCase()">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
        </svg>
      </button>
      <button class="tool-btn" @click="showRoutePanel = !showRoutePanel" title="路线规划">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="6" cy="19" r="3"/>
          <path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/>
          <circle cx="18" cy="5" r="3"/>
        </svg>
      </button>
      <button class="tool-btn" @click="showMultiPanel = !showMultiPanel" title="多点锁定">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <circle cx="5" cy="5" r="2"/>
          <circle cx="19" cy="5" r="2"/>
          <circle cx="5" cy="19" r="2"/>
          <circle cx="19" cy="19" r="2"/>
        </svg>
      </button>
      <button class="tool-btn" @click="showShareModal = true" title="分享位置">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="18" cy="5" r="3"/>
          <circle cx="6" cy="12" r="3"/>
          <circle cx="18" cy="19" r="3"/>
          <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/>
          <line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/>
        </svg>
      </button>
    </div>

    <!-- 坐标系切换浮层 -->
    <transition name="fade">
      <div class="coord-switcher" v-if="showCoordSwitcher">
        <div class="cs-title">坐标系</div>
        <button v-for="cs in coordSystems" :key="cs.key"
          class="cs-btn" :class="{ active: coordSystem === cs.key }"
          @click="setCoordSystem(cs.key)">
          <span class="cs-label">{{ cs.label }}</span>
          <span class="cs-desc">{{ cs.desc }}</span>
        </button>
      </div>
    </transition>

    <!-- 路线规划面板 -->
    <transition name="slide-up">
      <div class="route-panel" v-if="showRoutePanel">
        <div class="rp-hd">
          <span>路线规划</span>
          <button class="rp-close" @click="showRoutePanel = false">✕</button>
        </div>
        <div class="rp-body">
          <div class="rp-field">
            <label>起点</label>
            <input v-model="routeStart" placeholder="纬度,经度 或搜索地点" @keydown.enter="searchRoutePoint('start')">
            <button class="rp-use" @click="useCurrentAsStart">用当前点</button>
          </div>
          <div class="rp-field">
            <label>终点</label>
            <input v-model="routeEnd" placeholder="纬度,经度 或搜索地点" @keydown.enter="searchRoutePoint('end')">
            <button class="rp-use" @click="useCurrentAsEnd">用当前点</button>
          </div>
          <div class="rp-field">
            <label>速度</label>
            <select v-model="routeSpeed">
              <option value="5">步行 5 km/h</option>
              <option value="15">骑行 15 km/h</option>
              <option value="40">驾车 40 km/h</option>
              <option value="80">快速 80 km/h</option>
            </select>
          </div>
          <div class="rp-actions">
            <button class="rp-btn primary" @click="startRoute" :disabled="routeRunning">
              {{ routeRunning ? '模拟中...' : '开始模拟' }}
            </button>
            <button class="rp-btn" @click="stopRoute" :disabled="!routeRunning">停止</button>
          </div>
          <div class="rp-info" v-if="routeInfo">
            {{ routeInfo }}
          </div>
        </div>
      </div>
    </transition>

    <!-- 多点锁定面板 -->
    <transition name="slide-up">
      <div class="route-panel" v-if="showMultiPanel">
        <div class="rp-hd">
          <span>多点锁定</span>
          <button class="rp-close" @click="showMultiPanel = false">✕</button>
        </div>
        <div class="rp-body">
          <div class="mp-list">
            <div class="mp-empty" v-if="!multiPoints.length">点击「添加当前点」加入队列</div>
            <div class="mp-item" v-for="(p,i) in multiPoints" :key="i">
              <span class="mp-idx">{{ i+1 }}</span>
              <span class="mp-coord">{{ p.lat.toFixed(5) }}, {{ p.lng.toFixed(5) }}</span>
              <button class="mp-del" @click="multiPoints.splice(i,1)">✕</button>
            </div>
          </div>
          <div class="rp-field">
            <label>切换间隔（秒）</label>
            <input type="number" v-model.number="multiInterval" min="1" max="3600">
          </div>
          <div class="rp-actions">
            <button class="rp-btn" @click="addMultiPoint">添加当前点</button>
            <button class="rp-btn primary" @click="startMulti" :disabled="multiRunning">
              {{ multiRunning ? '运行中' : '开始切换' }}
            </button>
            <button class="rp-btn" @click="stopMulti" :disabled="!multiRunning">停止</button>
          </div>
        </div>
      </div>
    </transition>

    <div v-if="mapError" class="map-error">
      <div class="error-content">
        <div class="error-icon">⚠️</div>
        <div class="error-text">地图加载失败，请检查网络连接</div>
        <button class="error-retry" @click="initMap">重试</button>
      </div>
    </div>

    <!-- 底部操作栏（可收起） -->
    <div class="bottom-bar" :class="{ collapsed: bottomCollapsed }">
      <button class="bb-toggle" @click="bottomCollapsed = !bottomCollapsed">
        {{ bottomCollapsed ? '▲' : '▼' }}
      </button>
      <div class="bb-content" v-show="!bottomCollapsed">
        <div class="coord-line">
          <span class="coord-val" @click="copyCoord" title="点击复制">
            {{ displayLat.toFixed(6) }},{{ displayLng.toFixed(6) }}
            <span class="coord-sys">[{{ coordSystem.toUpperCase() }}]</span>
          </span>
          <span class="accuracy-hint">精度约 {{ accuracyMeters }}m</span>
          <button class="fav-toggle" @click="addFav">{{ isFaved ? '★' : '☆' }}</button>
        </div>
        <div class="status-line" v-if="hitCount > 0">
          <span class="status-dot active"></span>
          <span>已篡改 {{ hitCount }} 次定位请求</span>
          <span class="status-time" v-if="lastHitAgo">{{ lastHitAgo }}</span>
        </div>
        <div class="status-line" v-else-if="locked">
          <span class="status-dot waiting"></span>
          <span>等待 iOS 定位请求...</span>
        </div>
        <button
          class="lock-btn"
          :class="{ locked }"
          @click="lockLoc"
          :disabled="locking"
        >{{ locking ? '锁定中' : locked ? '已锁定 · 点击解锁' : '锁定此位置' }}</button>
      </div>
    </div>

    <!-- 摇杆（锁定后显示） -->
    <div class="joystick" v-if="locked && !bottomCollapsed">
      <div class="jk-base" @touchstart.prevent="jkStart" @touchmove.prevent="jkMove" @touchend.prevent="jkEnd"
           @mousedown.prevent="jkStart" @mousemove.prevent="jkMove" @mouseup.prevent="jkEnd" @mouseleave="jkEnd">
        <div class="jk-stick" :style="{ transform: `translate(${jkX}px, ${jkY}px)` }"></div>
      </div>
      <div class="jk-hint">摇杆微调位置</div>
    </div>

    <!-- 收藏/历史抽屉 -->
    <div class="drawer-mask" v-if="showPanel" @click="showPanel = false"></div>
    <transition name="drawer">
      <div class="drawer" v-if="showPanel">
        <div class="drawer-hd">
          <span>收藏 / 历史</span>
          <div class="drawer-actions">
            <button class="drawer-action" @click="exportFavs">导出</button>
            <label class="drawer-action">
              导入
              <input type="file" accept=".json" @change="importFavs" hidden>
            </label>
            <button class="drawer-close" @click="showPanel = false">✕</button>
          </div>
        </div>

        <div class="drawer-empty" v-if="!favs.length && !hist.length">
          <div class="empty-icon">📍</div>
          <div class="empty-text">暂无记录</div>
          <div class="empty-hint">搜索或锁定位置后自动保存</div>
        </div>

        <template v-if="favs.length">
          <div class="drawer-label">收藏 · 按城市分组</div>
          <div class="drawer-group" v-for="(items, city) in groupedFavs" :key="city">
            <div class="dg-city">{{ city || '未分类' }} ({{ items.length }})</div>
            <div class="drawer-row" v-for="(f,i) in items" :key="'f'+city+i" @click="jump(f)">
              <div class="dr-info">
                <div class="dr-name">{{ f.name }}</div>
                <div class="dr-coord">{{ f.lat.toFixed(5) }}, {{ f.lng.toFixed(5) }}</div>
              </div>
              <button class="dr-del" @click.stop="delFav(f)">✕</button>
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
            <input v-model="favName" placeholder="请输入收藏名称（如：公司、家等）" class="modal-input"
              @keydown.enter="confirmAddFav" ref="favInput" />
          </div>
          <button class="btn-primary" @click="confirmAddFav">确认收藏</button>
        </div>
      </div>
    </transition>

    <!-- 分享弹窗 -->
    <div class="modal-mask" v-if="showShareModal" @click="showShareModal = false"></div>
    <transition name="modal">
      <div class="modal" v-if="showShareModal">
        <div class="modal-hd">
          <span class="modal-title">分享位置</span>
          <button class="modal-close" @click="showShareModal = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="share-info">
            <div class="share-coord">{{ displayLat.toFixed(6) }}, {{ displayLng.toFixed(6) }}</div>
            <div class="share-sys">坐标系: {{ coordSystem.toUpperCase() }}</div>
          </div>
          <div class="share-link">
            <input ref="shareInput" :value="shareUrl" readonly class="modal-input">
            <button class="btn-primary" @click="copyShare">复制链接</button>
          </div>
          <div class="share-qr" v-html="qrSvg"></div>
          <div class="share-hint">扫描二维码或打开链接可快速定位到此坐标</div>
        </div>
      </div>
    </transition>

    <!-- Toast -->
    <transition name="toast">
      <div class="toast" v-if="toastMsg">{{ toastMsg }}</div>
    </transition>
  </div>
</template>

<script>
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { locationAPI, vpnAPI } from '../api'
import { COORD_SYSTEMS, toWgs84, fromWgs84, gcj02ToWgs84, wgs84ToGcj02, distance, formatDistance } from '../utils/coord'

export default {
  name: 'MapView',
  data: () => ({
    map: null,
    marker: null,
    tileLayer: null,
    mapLayer: 'standard',
    lastLockedLocation: null,
    currentLocationName: null,
    // 内部始终用 WGS84 存储
    lat: 39.90923, lng: 116.397428,
    // 用户选择的坐标系（仅影响显示和输入）
    coordSystem: 'wgs84',
    coordSystems: COORD_SYSTEMS,
    showCoordSwitcher: false,
    accuracyMeters: 65,
    locked: false, locking: false,
    query: '', results: [], focused: false, timer: null,
    showPanel: false,
    showFavModal: false,
    showShareModal: false,
    showRoutePanel: false,
    showMultiPanel: false,
    bottomCollapsed: false,
    favName: '',
    mapLoaded: false,
    mapError: false,
    searching: false,
    toastMsg: '',
    // 拦截统计
    hitCount: 0,
    lastHitTime: 0,
    lastHitAgo: '',
    statusTimer: null,
    // 从VPNView获取的客户端IP
    clientIP: '',
    // 搜索历史
    searchHistory: JSON.parse(localStorage.getItem('search_history') || '[]'),
    hotPlaces: [
      { name: '北京天安门', address: '北京市东城区', lat: 39.908722, lng: 116.397499 },
      { name: '上海外滩', address: '上海市黄浦区', lat: 31.239708, lng: 121.490817 },
      { name: '广州塔', address: '广州市海珠区', lat: 23.106480, lng: 113.324520 },
      { name: '深圳市民中心', address: '深圳市福田区', lat: 22.547778, lng: 114.085947 }
    ],
    favs: JSON.parse(localStorage.getItem('map_favs') || '[]'),
    hist: JSON.parse(localStorage.getItem('map_hist') || '[]'),
    // 路线规划
    routeStart: '', routeEnd: '', routeSpeed: '15',
    routeRunning: false, routeTimer: null, routeInfo: '',
    routePoints: [], routeIdx: 0,
    // 多点锁定
    multiPoints: [], multiInterval: 10, multiRunning: false, multiTimer: null,
    // 摇杆
    jkX: 0, jkY: 0, jkDragging: false, jkTimer: null
  }),
  computed: {
    // 显示坐标（按用户选择的坐标系转换）
    displayLat() {
      const c = fromWgs84(this.lng, this.lat, this.coordSystem)
      return c.lat
    },
    displayLng() {
      const c = fromWgs84(this.lng, this.lat, this.coordSystem)
      return c.lng
    },
    isFaved() {
      return this.favs.some(f => Math.abs(f.lat - this.lat) < 0.0001 && Math.abs(f.lng - this.lng) < 0.0001)
    },
    groupedFavs() {
      const groups = {}
      this.favs.forEach(f => {
        const city = f.city || this.guessCity(f.name)
        if (!groups[city]) groups[city] = []
        groups[city].push(f)
      })
      return groups
    },
    shareUrl() {
      const c = fromWgs84(this.lng, this.lat, this.coordSystem)
      return `${window.location.origin}/#/?lat=${c.lat.toFixed(6)}&lng=${c.lng.toFixed(6)}&sys=${this.coordSystem}`
    },
    qrSvg() {
      // 简易 QR 占位（实际用第三方库更好，这里用文字代替）
      const url = this.shareUrl
      if (url.length > 100) return '<div class="qr-fallback">链接较长<br>请复制使用</div>'
      // 用 SVG 画个简化方块图
      return `<div class="qr-fallback">📋<br>复制链接分享</div>`
    }
  },
  mounted() {
    this.initMap()
    this.startStatusPolling()
    // 从VPNView获取客户端IP
    this.clientIP = localStorage.getItem('vpn_client_ip') || ''
    // 如果没有，先尝试检查VPN获取一次
    if (!this.clientIP) {
      this.tryGetClientIP()
    }
    // 解析 URL 参数
    const params = new URLSearchParams(window.location.search)
    const lat = parseFloat(params.get('lat'))
    const lng = parseFloat(params.get('lng'))
    const sys = params.get('sys')
    if (lat && lng) {
      const wgs = toWgs84(lng, lat, sys || 'wgs84')
      this.lat = wgs.lat
      this.lng = wgs.lng
      if (sys) this.coordSystem = sys
      this.map?.setView([this.lat, this.lng], 15)
    }
  },
  beforeUnmount() {
    this.map?.remove()
    clearInterval(this.statusTimer)
    this.stopRoute()
    this.stopMulti()
    clearInterval(this.jkTimer)
  },
  methods: {
    async tryGetClientIP() {
      try {
        const res = await vpnAPI.checkConnection()
        if (res.data.client_ip) {
          this.clientIP = res.data.client_ip
          localStorage.setItem('vpn_client_ip', this.clientIP)
        }
      } catch {}
    },
    initMap() {
      try {
        this.map = L.map(this.$refs.mapEl, {
          center: [this.lat, this.lng], zoom: 15, zoomControl: false
        })
        this.tileLayer = L.tileLayer('https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}', {
          subdomains: ['1', '2', '3', '4'],
          attribution: '© 高德地图',
          maxZoom: 18
        }).addTo(this.map)
        L.control.zoom({ position: 'bottomright' }).addTo(this.map)
               this.map.on('move', () => {
          const c = this.map.getCenter()
          const wgs = gcj02ToWgs84(c.lng, c.lat)
          this.lat = wgs.lat; this.lng = wgs.lng
          this.currentLocationName = null
          // 修复：移动地图不再取消锁定，但移除标记
          if (this.marker && !this.locked) {
            this.map.removeLayer(this.marker)
            this.marker = null
          }
        }, { passive: true })
        // 长按选点
        this.map.on('contextmenu', (e) => {
          const wgs = gcj02ToWgs84(e.latlng.lng, e.latlng.lat)
          this.lat = wgs.lat; this.lng = wgs.lng
          this.map.panTo(e.latlng)
          this.showToast('已选点')
        })
        this.mapLoaded = true
      } catch (e) {
        console.error('地图初始化失败:', e)
        this.mapLoaded = false
        this.mapError = true
      }
    },
    setCoordSystem(sys) {
      this.coordSystem = sys
      this.showCoordSwitcher = false
      this.showToast(`已切换到 ${sys.toUpperCase()}`)
    },
    typeIcon(type) {
      const icons = {
        restaurant: '🍴', amenity: '🏥', shop: '🛍', tourism: '🏛',
        education: '🎓', transport: '🚉', place: '📍'
      }
      return icons[type] || '▸'
    },
    onInput() {
      clearTimeout(this.timer)
      if (!this.query.trim()) { this.results = []; return }
      this.searching = true
      this.timer = setTimeout(this.doSearch, 500)
    },
    async doSearch() {
      if (!this.query.trim()) return
      // 支持坐标输入（按当前坐标系解析）
      const coordMatch = this.query.trim().match(/^(-?\d+\.?\d*)[,，\s]+(-?\d+\.?\d*)$/)
      if (coordMatch) {
        const inLat = parseFloat(coordMatch[1])
        const inLng = parseFloat(coordMatch[2])
        if (inLat >= -90 && inLat <= 90 && inLng >= -180 && inLng <= 180) {
          const wgs = toWgs84(inLng, inLat, this.coordSystem)
          this.results = [{ name: `${inLat.toFixed(5)}, ${inLng.toFixed(5)}`, address: `${this.coordSystem.toUpperCase()} 坐标`, lat: wgs.lat, lng: wgs.lng, dist: '' }]
          return
        }
      }
      try {
        const res = await fetch(`/search?q=${encodeURIComponent(this.query)}`)
        const data = await res.json()
        if (data.features) {
          this.results = data.features.slice(0, 6).map(f => {
            const p = f.properties
            const name = p.name || p.city || p.street || this.query
            const parts = [p.city, p.state, p.country].filter(Boolean)
            const rLat = f.geometry.coordinates[1]
            const rLng = f.geometry.coordinates[0]
            const d = distance(this.lat, this.lng, rLat, rLng)
            return {
              name,
              address: parts.join(', ') || name,
              lat: rLat, lng: rLng,
              type: p.osm_value || p.osm_key,
              dist: formatDistance(d)
            }
          })
        } else { this.results = [] }
      } catch { this.results = [] }
      finally { this.searching = false }
    },
    pick(r) {
      this.query = r.name; this.results = []; this.focused = false; this.searching = false
      const gcj = wgs84ToGcj02(r.lng, r.lat)
      this.map.setView([gcj.lat, gcj.lng], 15)
      this.currentLocationName = r.name
      this.addToSearchHistory(r)
    },
    async lockLoc() {
      if (this.locked) {
        // 解锁
        this.locked = false
        if (this.marker) { this.map.removeLayer(this.marker); this.marker = null }
        this.showToast('已解锁')
        return
      }
      this.locking = true
      try {
        await locationAPI.setSpoofLocation({ latitude: this.lat, longitude: this.lng }, this.clientIP)
        this.locked = true
        this.lastLockedLocation = { lat: this.lat, lng: this.lng }
        const locationName = this.currentLocationName || `位置 ${this.lat.toFixed(4)},${this.lng.toFixed(4)}`
        this.pushHist({ lat: this.lat, lng: this.lng, name: locationName })
        if (this.marker) this.map.removeLayer(this.marker)
        const gcj = wgs84ToGcj02(this.lng, this.lat)
        this.marker = L.marker([gcj.lat, gcj.lng], {
          icon: L.divIcon({ className: 'lock-marker', html: '📍', iconSize: [32, 32], iconAnchor: [16, 32] })
        }).addTo(this.map)
        this.showToast('已锁定')
      } catch { this.showToast('锁定失败，请确认 VPN 已连接') }
      finally { this.locking = false }
    },
    addFav() {
      if (this.isFaved) return
      this.favName = this.currentLocationName || `位置 ${this.lat.toFixed(4)},${this.lng.toFixed(4)}`
      this.showFavModal = true
      this.$nextTick(() => this.$refs.favInput?.focus())
    },
    confirmAddFav() {
      if (!this.favName.trim()) return
      this.favs.unshift({ lat: this.lat, lng: this.lng, name: this.favName.trim(), city: this.guessCity(this.favName) })
      localStorage.setItem('map_favs', JSON.stringify(this.favs))
      this.showFavModal = false
      this.favName = ''
      this.showToast('已收藏')
    },
    delFav(f) {
      const i = this.favs.indexOf(f)
      if (i >= 0) { this.favs.splice(i, 1); localStorage.setItem('map_favs', JSON.stringify(this.favs)) }
    },
    guessCity(name) {
      const cities = ['北京','上海','广州','深圳','杭州','成都','武汉','南京','重庆','西安','天津','苏州','长沙','厦门','青岛']
      for (const c of cities) if (name.includes(c)) return c
      return '其他'
    },
    pushHist(item) {
      this.hist = [item, ...this.hist.filter(h => Math.abs(h.lat - item.lat) > 0.0001)].slice(0, 20)
      localStorage.setItem('map_hist', JSON.stringify(this.hist))
    },
    delHist(i) {
      this.hist.splice(i, 1)
      localStorage.setItem('map_hist', JSON.stringify(this.hist))
    },
    clearHist() { this.hist = []; localStorage.removeItem('map_hist') },
    copyCoord() {
      const coord = `${this.displayLat.toFixed(6)},${this.displayLng.toFixed(6)}`
      navigator.clipboard.writeText(coord).then(() => this.showToast('已复制')).catch(() => this.showToast('复制失败'))
    },
    addToSearchHistory(item) {
      const historyItem = { name: item.name, address: item.address, lat: item.lat, lng: item.lng }
      this.searchHistory = [historyItem, ...this.searchHistory.filter(h => h.name !== item.name)].slice(0, 10)
      localStorage.setItem('search_history', JSON.stringify(this.searchHistory))
    },
    locateUser() {
      if (!navigator.geolocation) return this.showToast('浏览器不支持定位')
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          this.map.setView([pos.coords.latitude, pos.coords.longitude], 15)
          this.lat = pos.coords.latitude; this.lng = pos.coords.longitude
          if (pos.coords.accuracy) this.accuracyMeters = Math.round(pos.coords.accuracy)
        },
        (err) => {
          const msgs = { 1: '定位权限被拒绝', 2: '无法获取位置', 3: '定位超时' }
          this.showToast(msgs[err.code] || '定位失败')
        },
        { timeout: 10000, enableHighAccuracy: true }
      )
    },
    switchLayer(layer) {
      this.mapLayer = layer
      if (this.tileLayer) this.map.removeLayer(this.tileLayer)
      if (layer === 'standard') {
        this.tileLayer = L.tileLayer('https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}', {
          subdomains: ['1','2','3','4'], attribution: '© 高德地图', maxZoom: 18
        })
      } else {
        this.tileLayer = L.tileLayer('https://webst0{s}.is.autonavi.com/appmaptile?style=6&x={x}&y={y}&z={z}', {
          subdomains: ['1','2','3','4'], attribution: '© 高德地图', maxZoom: 18
        })
      }
      this.tileLayer.addTo(this.map)
    },
    returnToLastLocked() {
      if (!this.lastLockedLocation) return this.showToast('没有上次锁定记录')
      const gcj = wgs84ToGcj02(this.lastLockedLocation.lng, this.lastLockedLocation.lat)
      this.map.setView([gcj.lat, gcj.lng], 15)
      this.lat = this.lastLockedLocation.lat; this.lng = this.lastLockedLocation.lng

      this.locked = true
      if (this.marker) this.map.removeLayer(this.marker)
      this.marker = L.marker([gcj.lat, gcj.lng], {
        icon: L.divIcon({ className: 'lock-marker', html: '📍', iconSize: [32, 32], iconAnchor: [16, 32] })
      }).addTo(this.map)
    },
    jump(item) {
      const gcj = wgs84ToGcj02(item.lng, item.lat)
      this.map.setView([gcj.lat, gcj.lng], 15)
      this.showPanel = false
    },
    showToast(msg) {
      this.toastMsg = msg
      clearTimeout(this._toastTimer)
      this._toastTimer = setTimeout(() => this.toastMsg = '', 2000)
    },
    // 状态轮询
    startStatusPolling() {
      this.fetchStatus()
      this.statusTimer = setInterval(() => {
        this.fetchStatus()
        if (this.lastHitTime) {
          const ago = Math.floor(Date.now() / 1000) - this.lastHitTime
          this.lastHitAgo = ago < 60 ? `${ago}秒前` : `${Math.floor(ago/60)}分钟前`
        }
      }, 5000)
    },
    async fetchStatus() {
      try {
        const res = await locationAPI.getStatus(this.clientIP)
        this.hitCount = res.data.hit_count || 0
        this.lastHitTime = res.data.last_hit || 0
      } catch {}
    },
    // 导入导出收藏
    exportFavs() {
      const blob = new Blob([JSON.stringify(this.favs, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = `favs-${Date.now()}.json`; a.click()
      URL.revokeObjectURL(url)
      this.showToast('已导出')
    },
    importFavs(e) {
      const file = e.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = (ev) => {
        try {
          const arr = JSON.parse(ev.target.result)
          if (Array.isArray(arr)) {
            this.favs = [...this.favs, ...arr]
            localStorage.setItem('map_favs', JSON.stringify(this.favs))
            this.showToast(`导入 ${arr.length} 条`)
          }
        } catch { this.showToast('导入失败：格式错误') }
      }
      reader.readAsText(file)
      e.target.value = ''
    },
    copyShare() {
      this.$refs.shareInput?.select()
      navigator.clipboard.writeText(this.shareUrl).then(() => this.showToast('链接已复制'))
    },
    // 路线规划
    useCurrentAsStart() { this.routeStart = `${this.lat.toFixed(6)},${this.lng.toFixed(6)}` },
    useCurrentAsEnd() { this.routeEnd = `${this.lat.toFixed(6)},${this.lng.toFixed(6)}` },
    searchRoutePoint(which) {
      const q = which === 'start' ? this.routeStart : this.routeEnd
      const m = q.match(/^(-?\d+\.?\d*)[,，\s]+(-?\d+\.?\d*)$/)
      if (m) return
      // 搜索地点
      fetch(`/search?q=${encodeURIComponent(q)}`).then(r => r.json()).then(data => {
        if (data.features && data.features[0]) {
          const f = data.features[0]
          const coord = `${f.geometry.coordinates[1].toFixed(6)},${f.geometry.coordinates[0].toFixed(6)}`
          if (which === 'start') this.routeStart = coord
          else this.routeEnd = coord
        }
      })
    },
    parseCoord(str) {
      const m = str.match(/^(-?\d+\.?\d*)[,，\s]+(-?\d+\.?\d*)$/)
      if (!m) return null
      return { lat: parseFloat(m[1]), lng: parseFloat(m[2]) }
    },
    startRoute() {
      const s = this.parseCoord(this.routeStart)
      const e = this.parseCoord(this.routeEnd)
      if (!s || !e) return this.showToast('请输入正确的起终点坐标')
      // 生成插值点（直线模拟）
      const steps = 50
      this.routePoints = []
      for (let i = 0; i <= steps; i++) {
        const t = i / steps
        this.routePoints.push({
          lat: s.lat + (e.lat - s.lat) * t,
          lng: s.lng + (e.lng - s.lng) * t
        })
      }
      const dist = distance(s.lat, s.lng, e.lat, e.lng)
      const speed = parseFloat(this.routeSpeed) * 1000 / 3600 // m/s
      const totalTime = dist / speed
      const interval = (totalTime / steps) * 1000
      this.routeIdx = 0
      this.routeRunning = true
      this.routeInfo = `总距离 ${formatDistance(dist)} · 预计 ${Math.round(totalTime/60)} 分钟`
      this.routeTimer = setInterval(() => {
        if (this.routeIdx >= this.routePoints.length) {
          this.stopRoute()
          this.showToast('路线模拟完成')
          return
        }
        const p = this.routePoints[this.routeIdx]
        this.lat = p.lat; this.lng = p.lng
        this.map.panTo([p.lat, p.lng])
        locationAPI.setSpoofLocation({ latitude: p.lat, longitude: p.lng }, this.clientIP)
        this.routeIdx++
      }, Math.max(500, interval))
    },
    stopRoute() {
      if (this.routeTimer) { clearInterval(this.routeTimer); this.routeTimer = null }
      this.routeRunning = false
    },
    // 多点锁定
    addMultiPoint() {
      this.multiPoints.push({ lat: this.lat, lng: this.lng })
    },
    startMulti() {
      if (this.multiPoints.length < 2) return this.showToast('至少需要 2 个点')
      this.multiRunning = true
      let idx = 0
      const run = () => {
        if (!this.multiRunning || idx >= this.multiPoints.length) {
          if (this.multiRunning) idx = 0
          else return
        }
        const p = this.multiPoints[idx]
        this.lat = p.lat; this.lng = p.lng
        this.map.panTo([p.lat, p.lng])
        locationAPI.setSpoofLocation({ latitude: p.lat, longitude: p.lng }, this.clientIP)
        this.showToast(`已切换到点 ${idx + 1}`)
        idx++
      }
      run()
      this.multiTimer = setInterval(run, this.multiInterval * 1000)
    },
    stopMulti() {
      if (this.multiTimer) { clearInterval(this.multiTimer); this.multiTimer = null }
      this.multiRunning = false
    },
    // 摇杆
    jkStart(e) { this.jkDragging = true; this.jkMove(e) },
    jkMove(e) {
      if (!this.jkDragging) return
      const touch = e.touches ? e.touches[0] : e
      const rect = e.currentTarget.getBoundingClientRect()
      const cx = rect.left + rect.width / 2
      const cy = rect.top + rect.height / 2
      let dx = touch.clientX - cx
      let dy = touch.clientY - cy
      const max = 40
      const d = Math.sqrt(dx*dx + dy*dy)
      if (d > max) { dx = dx/d*max; dy = dy/d*max }
      this.jkX = dx; this.jkY = dy
    },
    jkEnd() {
      this.jkDragging = false
      this.jkX = 0; this.jkY = 0
      clearInterval(this.jkTimer)
      this.jkTimer = null
    },
    jkApply() {
      if (!this.locked) return
      // 根据摇杆偏移微调位置
      const step = 0.00005
      this.lat -= this.jkY * step
      this.lng += this.jkX * step
      this.map.panTo([this.lat, this.lng])
      locationAPI.setSpoofLocation({ latitude: this.lat, longitude: this.lng }, this.clientIP)
    }
  },
  watch: {
    jkDragging(v) {
      if (v && !this.jkTimer) {
        this.jkTimer = setInterval(this.jkApply, 200)
      }
    }
  }
}
</script>

<style scoped>
.map-view { position: fixed; inset: 0; background: var(--bg); }
#map { position: absolute; inset: 0; z-index: 1; }

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
  animation: crosshair-pulse 2s ease-in-out infinite;
}
@keyframes crosshair-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
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
.nav-fav, .nav-locate, .nav-return { background: none; border: none; color: var(--text4); cursor: pointer; flex-shrink: 0; padding: 2px 4px; transition: transform .1s; display: flex; align-items: center; justify-content: center; }
.nav-fav:active, .nav-locate:active, .nav-return:active { transform: scale(0.9); }

.search-wrap {
  flex: 1; display: flex; align-items: center;
  background: var(--bg3); border: 1px solid var(--border);
  border-radius: 8px; padding: 8px 10px;
  transition: border-color .2s;
}
.search-wrap.focus { border-color: var(--text4); }
.search-wrap input { flex: 1; border: none; background: transparent; font-size: 16px; color: var(--text); outline: none; }
.search-wrap input::placeholder { color: var(--text5); }
.search-clear { background: none; border: none; font-size: 11px; color: var(--text4); cursor: pointer; padding: 0 2px; }

/* 搜索结果 */
.search-results {
  position: absolute;
  top: calc(max(var(--safe-top), 10px) + 54px);
  left: 0; right: 0; margin: 0 12px;
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: 10px; z-index: 700;
  max-height: 44vh; overflow-y: auto;
}
.sr-row { display: flex; align-items: flex-start; gap: 10px; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; transition: background .15s; }
.sr-row:last-child { border-bottom: none; }
.sr-row:active { background: var(--bg3); transform: scale(0.98); }
.sr-icon { font-size: 14px; flex-shrink: 0; margin-top: 1px; }
.sr-name { font-size: 14px; color: var(--text2); font-weight: 500; }
.sr-addr { font-size: 12px; color: var(--text4); margin-top: 2px; line-height: 1.4; }
.sr-dist { color: var(--text3); margin-left: 4px; }
.sr-loading { padding: 20px; text-align: center; color: var(--text4); font-size: 13px; }
.sr-section-label { font-size: 11px; color: var(--text5); text-transform: uppercase; letter-spacing: .8px; padding: 12px 13px 8px; }
.sr-empty { padding: 20px; text-align: center; color: var(--text5); font-size: 13px; }

/* 右侧工具栏 */
.map-tools {
  position: absolute; right: 12px;
  top: calc(max(var(--safe-top), 10px) + 60px);
  display: flex; flex-direction: column; gap: 8px;
  z-index: 600;
}
.layer-btn, .tool-btn {
  padding: 8px 12px;
  background: var(--bar-bg, rgba(10,10,10,.93));
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text3);
  font-size: 12px;
  cursor: pointer;
  backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
  transition: all .2s, transform .1s;
  display: flex; align-items: center; justify-content: center;
  min-width: 36px; min-height: 32px;
}
.layer-btn:active, .tool-btn:active { transform: scale(0.95); }
.layer-btn.active, .tool-btn.active {
  background: var(--text2); color: var(--bg); border-color: var(--text2);
}
@media (prefers-color-scheme: light) {
  .layer-btn, .tool-btn { --bar-bg: rgba(242,242,247,.93); }
}

/* 坐标系切换 */
.coord-switcher {
  position: absolute; right: 12px;
  top: calc(max(var(--safe-top), 10px) + 220px);
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: 10px; padding: 8px;
  z-index: 700; min-width: 140px;
  box-shadow: 0 4px 16px rgba(0,0,0,.3);
}
.cs-title { font-size: 11px; color: var(--text5); padding: 4px 8px; text-transform: uppercase; }
.cs-btn {
  width: 100%; display: flex; justify-content: space-between;
  padding: 8px 10px; background: none; border: none;
  color: var(--text3); font-size: 13px; cursor: pointer;
  border-radius: 6px; transition: background .15s;
}
.cs-btn:hover { background: var(--bg3); }
.cs-btn.active { background: var(--bg3); color: var(--text); }
.cs-desc { font-size: 11px; color: var(--text5); }

/* 路线/多点面板 */
.route-panel {
  position: absolute; left: 12px; right: 12px;
  bottom: calc(max(var(--safe-bottom), 10px) + 120px);
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: 12px; z-index: 700;
  box-shadow: 0 4px 24px rgba(0,0,0,.4);
  max-height: 50vh; overflow-y: auto;
}
.rp-hd {
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 14px; border-bottom: 1px solid var(--border);
  font-size: 14px; font-weight: 600; color: var(--text2);
}
.rp-close { background: none; border: none; color: var(--text4); cursor: pointer; font-size: 14px; }
.rp-body { padding: 12px 14px; display: flex; flex-direction: column; gap: 10px; }
.rp-field { display: flex; flex-direction: column; gap: 4px; }
.rp-field label { font-size: 11px; color: var(--text4); }
.rp-field input, .rp-field select {
  padding: 8px 10px; background: var(--bg3); border: 1px solid var(--border);
  border-radius: 6px; color: var(--text); font-size: 14px; outline: none;
}
.rp-use {
  align-self: flex-start; background: none; border: 1px solid var(--border);
  padding: 4px 8px; border-radius: 4px; color: var(--text4);
  font-size: 11px; cursor: pointer;
}
.rp-actions { display: flex; gap: 8px; }
.rp-btn {
  flex: 1; padding: 10px; background: var(--bg3);
  border: 1px solid var(--border); border-radius: 8px;
  color: var(--text3); font-size: 13px; cursor: pointer;
}
.rp-btn.primary { background: var(--btn-bg); color: var(--btn-fg); border-color: var(--btn-border); }
.rp-btn:disabled { opacity: .5; cursor: not-allowed; }
.rp-info { font-size: 12px; color: var(--text4); padding: 4px 0; }

.mp-list { display: flex; flex-direction: column; gap: 4px; max-height: 120px; overflow-y: auto; }
.mp-empty { font-size: 12px; color: var(--text5); padding: 8px 0; text-align: center; }
.mp-item {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 8px; background: var(--bg3); border-radius: 6px;
  font-size: 12px; color: var(--text3);
}
.mp-idx { background: var(--text5); color: var(--bg); width: 18px; height: 18px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 10px; }
.mp-coord { flex: 1; font-family: monospace; }
.mp-del { background: none; border: none; color: var(--text5); cursor: pointer; }

/* 底部栏 */
.bottom-bar {
  position: absolute; bottom: 0; left: 0; right: 0;
  padding-bottom: max(var(--safe-bottom), 10px);
  padding-top: 8px; padding-left: 14px; padding-right: 14px;
  background: var(--bar-bg, rgba(10,10,10,.93));
  border-top: 1px solid var(--border);
  z-index: 600;
  backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
  transition: padding .2s;
}
.bottom-bar.collapsed { padding-top: 4px; padding-bottom: max(var(--safe-bottom), 6px); }
@media (prefers-color-scheme: light) {
  .bottom-bar { --bar-bg: rgba(242,242,247,.95); }
}
.bb-toggle {
  width: 100%; background: none; border: none;
  color: var(--text5); font-size: 10px; cursor: pointer;
  padding: 2px 0; text-align: center;
}
.bb-content { display: flex; flex-direction: column; gap: 8px; }

.coord-line { display: flex; align-items: center; gap: 8px; }
.coord-val { flex: 1; font-size: 12px; font-family: "SF Mono","Menlo",monospace; color: var(--text4); cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.coord-val:active { background: var(--bg3); }
.coord-sys { color: var(--text5); font-size: 10px; margin-left: 4px; }
.accuracy-hint { font-size: 11px; color: var(--text5); }
.fav-toggle { background: none; border: none; font-size: 16px; color: var(--text4); cursor: pointer; padding: 2px 4px; }

.status-line {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--text4);
}
.status-dot { width: 6px; height: 6px; border-radius: 50%; }
.status-dot.active { background: #34c759; box-shadow: 0 0 6px #34c759; animation: pulse 1.5s infinite; }
.status-dot.waiting { background: #ff9f1c; animation: pulse 1.5s infinite; }
.status-time { margin-left: auto; color: var(--text5); }
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.lock-btn {
  width: 100%; padding: 13px;
  background: transparent; color: var(--text2);
  border: 1px solid var(--border); border-radius: 8px;
  font-size: 15px; font-weight: 500; cursor: pointer;
  transition: all .2s;
}
.lock-btn:active:not(:disabled) { background: var(--bg2); }
.lock-btn.locked {
  background: #34c759; color: white; border-color: #34c759;
  font-weight: 600;
  animation: lock-pulse 2s infinite;
}
@keyframes lock-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(52,199,89,.4); }
  50% { box-shadow: 0 0 0 8px rgba(52,199,89,0); }
}
.lock-btn:disabled { color: var(--text5); border-color: var(--border2); cursor: not-allowed; }

/* 摇杆 */
.joystick {
  position: absolute; left: 20px; bottom: calc(max(var(--safe-bottom), 10px) + 140px);
  z-index: 600;
}
.jk-base {
  width: 80px; height: 80px; border-radius: 50%;
  background: var(--bar-bg, rgba(10,10,10,.6));
  border: 2px solid var(--border);
  position: relative;
  display: flex; align-items: center; justify-content: center;
  backdrop-filter: blur(8px);
}
.jk-stick {
  width: 30px; height: 30px; border-radius: 50%;
  background: var(--text2);
  transition: transform .05s;
  pointer-events: none;
}
.jk-hint { font-size: 10px; color: var(--text5); text-align: center; margin-top: 4px; }

/* 抽屉 */
.drawer-mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 800; }
.drawer {
  position: fixed; left: 0; right: 0; bottom: 0;
  margin: 0 auto; width: 100%; max-width: 420px;
  max-height: 70vh; background: var(--bg);
  border-top: 1px solid var(--border);
  border-radius: 14px 14px 0 0;
  z-index: 900; overflow-y: auto;
  padding: 0 14px max(var(--safe-bottom), 16px);
}
.drawer::-webkit-scrollbar { width: 4px; }
.drawer::-webkit-scrollbar-thumb { background: var(--border); border-radius: 2px; }
.drawer-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 0 10px; position: sticky; top: 0;
  background: var(--bg); border-bottom: 1px solid var(--border);
  font-size: 15px; font-weight: 600; color: var(--text2);
}
.drawer-actions { display: flex; align-items: center; gap: 8px; }
.drawer-action {
  background: none; border: 1px solid var(--border);
  padding: 4px 8px; border-radius: 4px;
  color: var(--text4); font-size: 11px; cursor: pointer;
}
.drawer-close { background: none; border: none; font-size: 13px; color: var(--text4); cursor: pointer; padding: 4px 6px; }
.drawer-label { font-size: 10px; color: var(--text5); text-transform: uppercase; letter-spacing: .8px; padding: 12px 0 6px; }
.drawer-group { background: var(--bg2); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; margin-bottom: 8px; }
.dg-city { padding: 8px 13px; font-size: 12px; color: var(--text3); background: var(--bg3); border-bottom: 1px solid var(--border2); }
.drawer-row { display: flex; align-items: center; padding: 11px 13px; border-bottom: 1px solid var(--border2); cursor: pointer; }
.drawer-row:last-child { border-bottom: none; }
.drawer-row:active { background: var(--bg3); }
.dr-info { flex: 1; }
.dr-name { font-size: 14px; color: var(--text2); }
.dr-coord { font-size: 11px; color: var(--text4); font-family: monospace; margin-top: 2px; }
.dr-del { background: none; border: none; font-size: 12px; color: var(--text5); cursor: pointer; padding: 4px 6px; }
.drawer-clear { width: 100%; padding: 12px; background: none; border: none; color: var(--text4); font-size: 13px; cursor: pointer; }
.drawer-empty { text-align: center; color: var(--text5); font-size: 13px; padding: 32px 0; }
.empty-icon { font-size: 32px; margin-bottom: 12px; }
.empty-text { font-size: 14px; color: var(--text3); margin-bottom: 8px; }
.empty-hint { font-size: 12px; color: var(--text5); }

.drawer-enter-active, .drawer-leave-active { transition: transform .25s ease; }
.drawer-enter-from, .drawer-leave-to { transform: translateY(100%); }

/* 地图错误 */
.map-error { position: fixed; inset: 0; background: var(--bg); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.error-content { text-align: center; padding: 32px; }
.error-icon { font-size: 48px; margin-bottom: 16px; }
.error-text { font-size: 15px; color: var(--text3); margin-bottom: 20px; }
.error-retry { padding: 12px 24px; background: var(--text2); color: var(--bg); border: none; border-radius: 8px; font-size: 14px; cursor: pointer; }

/* 弹窗 */
.modal-mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 1000; display: flex; align-items: flex-end; justify-content: center; padding: 0 16px; }
.modal { position: relative; width: 100%; max-width: 360px; background: var(--bg); border-radius: 16px; z-index: 1001; padding: 0 16px 20px; border: 1px solid var(--border); margin-bottom: env(safe-area-inset-bottom, 20px); }
.modal-hd { display: flex; align-items: center; justify-content: space-between; padding: 16px 0 12px; border-bottom: 1px solid var(--border); }
.modal-title { font-size: 17px; font-weight: 600; color: var(--text); }
.modal-close { background: none; border: none; font-size: 18px; color: var(--text4); cursor: pointer; }
.modal-body { display: flex; flex-direction: column; gap: 14px; padding: 18px 0 4px; }
.modal-input { width: 100%; padding: 12px 14px; background: var(--bg3); border: 1px solid var(--border); border-radius: 8px; font-size: 15px; color: var(--text); outline: none; }

.share-info { text-align: center; padding: 8px 0; }
.share-coord { font-size: 16px; font-family: monospace; color: var(--text); }
.share-sys { font-size: 12px; color: var(--text4); margin-top: 4px; }
.share-link { display: flex; flex-direction: column; gap: 8px; }
.share-qr { text-align: center; padding: 12px; }
.qr-fallback { font-size: 13px; color: var(--text4); padding: 16px; background: var(--bg3); border-radius: 8px; }
.share-hint { font-size: 11px; color: var(--text5); text-align: center; }

.modal-enter-active, .modal-leave-active { transition: opacity .2s ease, transform .2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: translateY(20px); }

/* Toast */
.toast {
  position: fixed; left: 50%; top: 50%;
  transform: translate(-50%, -50%);
  background: rgba(0,0,0,.85); color: #fff;
  padding: 12px 20px; border-radius: 8px;
  font-size: 14px; z-index: 9999;
  white-space: nowrap;
}
.toast-enter-active, .toast-leave-active { transition: opacity .2s; }
.toast-enter-from, .toast-leave-to { opacity: 0; }

/* 过渡 */
.fade-enter-active, .fade-leave-active { transition: opacity .2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.slide-up-enter-active, .slide-up-leave-active { transition: transform .25s ease, opacity .2s; }
.slide-up-enter-from, .slide-up-leave-to { transform: translateY(20px); opacity: 0; }

/* 锁定标记 */
.lock-marker { font-size: 28px; text-shadow: 0 2px 8px rgba(0,0,0,0.3); filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2)); }
</style>
