import type { Ref } from 'vue'

interface SunnyTownRendererOptions {
  drawScene: (context: CanvasRenderingContext2D, width: number, height: number) => void
  onFrame: (deltaSeconds: number) => void
}

export function useSunnyTownRenderer(
  canvas: Ref<HTMLCanvasElement | null>,
  options: SunnyTownRendererOptions,
) {
  let animationFrame = 0
  let lastRenderTime = 0

  function start() {
    animationFrame = window.requestAnimationFrame(renderLoop)
  }

  function stop() {
    window.cancelAnimationFrame(animationFrame)
    animationFrame = 0
    lastRenderTime = 0
  }

  function draw() {
    const target = canvas.value
    if (!target) {
      return
    }

    const context = target.getContext('2d')
    if (!context) {
      return
    }

    const rect = target.getBoundingClientRect()
    const scale = window.devicePixelRatio || 1
    const width = Math.max(1, Math.floor(rect.width * scale))
    const height = Math.max(1, Math.floor(rect.height * scale))
    if (target.width !== width || target.height !== height) {
      target.width = width
      target.height = height
    }

    context.setTransform(scale, 0, 0, scale, 0, 0)
    context.clearRect(0, 0, rect.width, rect.height)
    options.drawScene(context, rect.width, rect.height)
  }

  function renderLoop() {
    const now = performance.now()
    const deltaSeconds = lastRenderTime ? (now - lastRenderTime) / 1000 : 0
    lastRenderTime = now
    options.onFrame(deltaSeconds)
    draw()
    animationFrame = window.requestAnimationFrame(renderLoop)
  }

  return {
    draw,
    start,
    stop,
  }
}
