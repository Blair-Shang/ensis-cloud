/** 与工作台 chat 围栏同一套：内联 SVG，不引入 ECharts。 */

export type UsageChartSpec = {
  type: 'bar' | 'line'
  title?: string
  labels: string[]
  values: number[]
}

const NS = 'http://www.w3.org/2000/svg'

function el(name: string, attrs: Record<string, string | number>, text?: string): SVGElement {
  const node = document.createElementNS(NS, name)
  for (const [k, v] of Object.entries(attrs)) {
    node.setAttribute(k, String(v))
  }
  if (text != null) node.textContent = text
  return node
}

function palette(i: number): string {
  const hues = [210, 32, 152, 258, 12, 198]
  return `hsl(${hues[i % hues.length]} 62% 52%)`
}

export function renderUsageChartSvg(spec: UsageChartSpec): string {
  const width = 420
  const height = 240
  const svg = el('svg', {
    viewBox: `0 0 ${width} ${height}`,
    width: '100%',
    role: 'img',
    'aria-label': spec.title || 'chart',
  })
  if (spec.title) {
    svg.appendChild(
      el(
        'text',
        {
          x: width / 2,
          y: 20,
          'text-anchor': 'middle',
          fill: 'currentColor',
          'font-size': 13,
          'font-weight': 600,
        },
        spec.title,
      ),
    )
  }
  if (spec.type === 'line') drawLine(svg, spec, width, height)
  else drawBars(svg, spec, width, height)
  return new XMLSerializer().serializeToString(svg)
}

function drawBars(svg: SVGElement, spec: UsageChartSpec, width: number, height: number): void {
  const top = spec.title ? 36 : 16
  const bottom = height - 36
  const left = 36
  const right = width - 12
  const max = Math.max(...spec.values, 1)
  const n = Math.max(spec.values.length, 1)
  const gap = 8
  const slot = (right - left) / n
  const barW = Math.max(6, slot - gap)
  spec.values.forEach((v, i) => {
    const h = ((bottom - top) * Math.max(0, v)) / max
    const x = left + i * slot + (slot - barW) / 2
    const y = bottom - h
    svg.appendChild(el('rect', { x, y, width: barW, height: Math.max(h, 0), rx: 3, fill: palette(i) }))
    svg.appendChild(
      el(
        'text',
        { x: x + barW / 2, y: height - 14, 'text-anchor': 'middle', fill: 'currentColor', 'font-size': 10 },
        spec.labels[i] ?? '',
      ),
    )
  })
}

function drawLine(svg: SVGElement, spec: UsageChartSpec, width: number, height: number): void {
  const top = spec.title ? 36 : 16
  const bottom = height - 36
  const left = 36
  const right = width - 16
  const max = Math.max(...spec.values, 1)
  const n = spec.values.length
  if (n === 0) return
  const pts = spec.values.map((v, i) => {
    const x = n === 1 ? (left + right) / 2 : left + (i * (right - left)) / (n - 1)
    const y = bottom - ((bottom - top) * Math.max(0, v)) / max
    return `${x},${y}`
  })
  svg.appendChild(
    el('polyline', {
      points: pts.join(' '),
      fill: 'none',
      stroke: palette(0),
      'stroke-width': 2.2,
      'stroke-linejoin': 'round',
      'stroke-linecap': 'round',
    }),
  )
  spec.values.forEach((v, i) => {
    const x = n === 1 ? (left + right) / 2 : left + (i * (right - left)) / (n - 1)
    const y = bottom - ((bottom - top) * Math.max(0, v)) / max
    svg.appendChild(el('circle', { cx: x, cy: y, r: 3.5, fill: palette(0) }))
    svg.appendChild(
      el(
        'text',
        { x, y: height - 14, 'text-anchor': 'middle', fill: 'currentColor', 'font-size': 10 },
        spec.labels[i] ?? '',
      ),
    )
  })
}

export function shortUsageLabel(row: { displayName?: string; email?: string; userId?: string }): string {
  const name = (row.displayName || '').trim()
  if (name) return name.slice(0, 6)
  const email = (row.email || '').trim()
  if (email.includes('@')) return email.slice(0, email.indexOf('@')).slice(0, 8)
  return (row.userId || '—').slice(0, 8)
}
