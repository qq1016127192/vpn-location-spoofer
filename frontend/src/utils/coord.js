// 坐标系转换工具
// WGS84: GPS 原始坐标
// GCJ02: 高德/腾讯坐标（火星坐标）
// BD09:  百度坐标

const PI = Math.PI
const A = 6378245.0
const EE = 0.00669342162296594323

function outOfChina(lng, lat) {
  return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271
}

function transformLat(x, y) {
  let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x))
  ret += (20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0 / 3.0
  ret += (20.0 * Math.sin(y * PI) + 40.0 * Math.sin(y / 3.0 * PI)) * 2.0 / 3.0
  ret += (160.0 * Math.sin(y / 12.0 * PI) + 320 * Math.sin(y * PI / 30.0)) * 2.0 / 3.0
  return ret
}

function transformLng(x, y) {
  let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x))
  ret += (20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0 / 3.0
  ret += (20.0 * Math.sin(x * PI) + 40.0 * Math.sin(x / 3.0 * PI)) * 2.0 / 3.0
  ret += (150.0 * Math.sin(x / 12.0 * PI) + 300.0 * Math.sin(x / 30.0 * PI)) * 2.0 / 3.0
  return ret
}

// WGS84 -> GCJ02
export function wgs84ToGcj02(lng, lat) {
  if (outOfChina(lng, lat)) return { lng, lat }
  let dLat = transformLat(lng - 105.0, lat - 35.0)
  let dLng = transformLng(lng - 105.0, lat - 35.0)
  const radLat = lat / 180.0 * PI
  let magic = Math.sin(radLat)
  magic = 1 - EE * magic * magic
  const sqrtMagic = Math.sqrt(magic)
  dLat = (dLat * 180.0) / ((A * (1 - EE)) / (magic * sqrtMagic) * PI)
  dLng = (dLng * 180.0) / (A / sqrtMagic * Math.cos(radLat) * PI)
  return { lng: lng + dLng, lat: lat + dLat }
}

// GCJ02 -> WGS84
export function gcj02ToWgs84(lng, lat) {
  if (outOfChina(lng, lat)) return { lng, lat }
  const gcj = wgs84ToGcj02(lng, lat)
  return { lng: lng * 2 - gcj.lng, lat: lat * 2 - gcj.lat }
}

// GCJ02 -> BD09
export function gcj02ToBd09(lng, lat) {
  const z = Math.sqrt(lng * lng + lat * lat) + 0.00002 * Math.sin(lat * PI * 3000.0 / 180.0)
  const theta = Math.atan2(lat, lng) + 0.000003 * Math.cos(lng * PI * 3000.0 / 180.0)
  return { lng: z * Math.cos(theta), lat: z * Math.sin(theta) }
}

// BD09 -> GCJ02
export function bd09ToGcj02(lng, lat) {
  const x = lng - 0.0065
  const y = lat - 0.006
  const z = Math.sqrt(x * x + y * y) - 0.00002 * Math.sin(y * PI * 3000.0 / 180.0)
  const theta = Math.atan2(y, x) - 0.000003 * Math.cos(x * PI * 3000.0 / 180.0)
  return { lng: z * Math.cos(theta), lat: z * Math.sin(theta) }
}

// BD09 -> WGS84
export function bd09ToWgs84(lng, lat) {
  const gcj = bd09ToGcj02(lng, lat)
  return gcj02ToWgs84(gcj.lng, gcj.lat)
}

// WGS84 -> BD09
export function wgs84ToBd09(lng, lat) {
  const gcj = wgs84ToGcj02(lng, lat)
  return gcj02ToBd09(gcj.lng, gcj.lat)
}

// 坐标系定义
export const COORD_SYSTEMS = [
  { key: 'wgs84', label: 'WGS84', desc: 'GPS 原始' },
  { key: 'gcj02', label: 'GCJ02', desc: '高德/腾讯' },
  { key: 'bd09', label: 'BD09', desc: '百度' }
]

// 从指定坐标系转换到 WGS84
export function toWgs84(lng, lat, from) {
  switch (from) {
    case 'gcj02': return gcj02ToWgs84(lng, lat)
    case 'bd09': return bd09ToWgs84(lng, lat)
    default: return { lng, lat }
  }
}

// 从 WGS84 转换到指定坐标系
export function fromWgs84(lng, lat, to) {
  switch (to) {
    case 'gcj02': return wgs84ToGcj02(lng, lat)
    case 'bd09': return wgs84ToBd09(lng, lat)
    default: return { lng, lat }
  }
}

// 计算两点距离（米）
export function distance(lat1, lng1, lat2, lng2) {
  const R = 6371000
  const dLat = (lat2 - lat1) * PI / 180
  const dLng = (lng2 - lng1) * PI / 180
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * PI / 180) * Math.cos(lat2 * PI / 180) * Math.sin(dLng / 2) ** 2
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

// 格式化距离
export function formatDistance(m) {
  if (m < 1000) return `${Math.round(m)}m`
  return `${(m / 1000).toFixed(1)}km`
}
