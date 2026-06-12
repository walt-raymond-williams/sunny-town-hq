import type {
  SunnyTownCollectible,
  SunnyTownNpc,
  SunnyTownPlayer,
} from '../../../types/sunnyTown'

interface DrawPlayerOptions {
  selfId: string
  selectedToolKey: string
  toolUseProgress: (toolKey: string) => number | null
}

export function drawPlayer(
  context: CanvasRenderingContext2D,
  player: SunnyTownPlayer,
  cameraX: number,
  cameraY: number,
  options: DrawPlayerOptions,
) {
  const x = player.x - cameraX
  const y = player.y - cameraY
  const isSelf = player.id === options.selfId

  context.fillStyle = 'rgba(0, 0, 0, 0.18)'
  context.beginPath()
  context.ellipse(x, y + 16, 18, 7, 0, 0, Math.PI * 2)
  context.fill()

  const gearKey = player.equipment?.gear || ''
  const accessoryKey = player.equipment?.accessory || ''
  const toolKey = isSelf ? options.selectedToolKey : player.equipment?.tool || ''
  const toolProgress = isSelf ? options.toolUseProgress(toolKey) : null

  context.fillStyle = gearKey === 'sunny_hoodie' ? '#f06f38' : (isSelf ? '#27746f' : '#5c6bc0')
  context.beginPath()
  context.arc(x, y, 16, 0, Math.PI * 2)
  context.fill()

  if (toolKey === 'pickaxe') {
    drawPickaxe(context, x, y, player.facing, toolProgress)
  }

  if (gearKey === 'sunny_hoodie') {
    context.fillStyle = '#2f7d72'
    context.fillRect(x - 10, y + 2, 20, 8)
    context.strokeStyle = '#f4d48e'
    context.lineWidth = 2
    context.beginPath()
    context.moveTo(x, y + 2)
    context.lineTo(x, y + 10)
    context.stroke()
  }

  if (accessoryKey === 'star_cap') {
    context.fillStyle = '#f2c84b'
    context.beginPath()
    context.ellipse(x, y - 14, 14, 6, 0, 0, Math.PI * 2)
    context.fill()
    context.fillStyle = '#365d9f'
    context.fillRect(x - 9, y - 20, 18, 8)
    context.fillStyle = '#ffffff'
    context.beginPath()
    context.moveTo(x, y - 21)
    context.lineTo(x + 3, y - 16)
    context.lineTo(x + 8, y - 16)
    context.lineTo(x + 4, y - 13)
    context.lineTo(x + 6, y - 8)
    context.lineTo(x, y - 11)
    context.lineTo(x - 6, y - 8)
    context.lineTo(x - 4, y - 13)
    context.lineTo(x - 8, y - 16)
    context.lineTo(x - 3, y - 16)
    context.closePath()
    context.fill()
  }

  context.fillStyle = '#ffffff'
  context.beginPath()
  context.arc(x - 5, y - 4, 3, 0, Math.PI * 2)
  context.arc(x + 5, y - 4, 3, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#17212b'
  context.font = '700 12px Inter, sans-serif'
  context.textAlign = 'center'
  context.fillText(player.displayName, x, y - 24)
}

export function drawPickaxe(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  facing: SunnyTownPlayer['facing'],
  swingProgress: number | null,
) {
  const swinging = swingProgress !== null
  const direction = directionVector(facing)
  const side = facing === 'left' || facing === 'right' ? -1 : 1
  const baseX = x + direction.x * 17 + (facing === 'up' || facing === 'down' ? 13 : 0)
  const baseY = y + direction.y * 14 + (facing === 'left' || facing === 'right' ? 2 : 6)
  const swingAngle = swinging ? (Math.sin(swingProgress * Math.PI) * 1.2 - 0.6) * side : 0
  const restingAngle = facing === 'left'
    ? -0.8
    : facing === 'right'
      ? 0.8
      : facing === 'up'
        ? -0.35
        : 0.35

  context.save()
  context.translate(baseX, baseY)
  context.rotate(restingAngle + swingAngle)
  context.lineCap = 'round'
  context.strokeStyle = swinging ? '#f6d56f' : '#7b4b24'
  context.lineWidth = swinging ? 5 : 4
  context.beginPath()
  context.moveTo(0, 12)
  context.lineTo(0, -13)
  context.stroke()
  context.strokeStyle = '#5f6b75'
  context.lineWidth = swinging ? 6 : 5
  context.beginPath()
  context.moveTo(-10, -14)
  context.quadraticCurveTo(0, -21, 12, -14)
  context.stroke()
  context.restore()
}

export function directionVector(facing: SunnyTownPlayer['facing']) {
  switch (facing) {
    case 'up':
      return { x: 0, y: -1 }
    case 'down':
      return { x: 0, y: 1 }
    case 'left':
      return { x: -1, y: 0 }
    case 'right':
      return { x: 1, y: 0 }
  }
}

export function drawNpc(
  context: CanvasRenderingContext2D,
  npc: SunnyTownNpc,
  cameraX: number,
  cameraY: number,
  nearbyNpcId: string,
) {
  const x = npc.x - cameraX
  const y = npc.y - cameraY
  const isNearby = nearbyNpcId === npc.id

  context.fillStyle = 'rgba(0, 0, 0, 0.18)'
  context.beginPath()
  context.ellipse(x, y + 16, 18, 7, 0, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = npc.spriteKey === 'keeper' ? '#8b4f9f' : npc.spriteKey === 'teacher' ? '#2f6b8f' : '#b96b4f'
  context.beginPath()
  context.arc(x, y, 16, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#f7d7b5'
  context.beginPath()
  context.arc(x, y - 4, 9, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#17212b'
  context.beginPath()
  context.arc(x - 3, y - 6, 1.5, 0, Math.PI * 2)
  context.arc(x + 3, y - 6, 1.5, 0, Math.PI * 2)
  context.fill()

  context.strokeStyle = isNearby ? '#f1d28f' : 'rgba(255, 255, 255, 0.38)'
  context.lineWidth = isNearby ? 3 : 2
  context.beginPath()
  context.arc(x, y, 19, 0, Math.PI * 2)
  context.stroke()

  context.fillStyle = '#17212b'
  context.font = '700 12px Inter, sans-serif'
  context.textAlign = 'center'
  context.fillText(npc.name, x, y - 28)

  drawNpcRoutineCue(context, npc.routineStatus, x, y - 45)
}

export function drawNpcRoutineCue(
  context: CanvasRenderingContext2D,
  status: SunnyTownNpc['routineStatus'],
  x: number,
  y: number,
) {
  const cue = npcRoutineCue(status)
  if (!cue) {
    return
  }
  context.save()
  context.fillStyle = cue.background
  context.strokeStyle = 'rgba(23, 33, 43, 0.36)'
  context.lineWidth = 1
  context.beginPath()
  context.arc(x, y, 8, 0, Math.PI * 2)
  context.fill()
  context.stroke()
  context.fillStyle = cue.foreground
  context.font = '700 10px Inter, sans-serif'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(cue.symbol, x, y)
  context.restore()
}

export function npcRoutineCue(status: SunnyTownNpc['routineStatus']) {
  switch (status) {
    case 'traveling':
      return { symbol: '>', background: '#f1d28f', foreground: '#17212b' }
    case 'resting':
      return { symbol: 'Z', background: '#82b6d9', foreground: '#17212b' }
    case 'working':
      return { symbol: 'W', background: '#8fd19e', foreground: '#17212b' }
    case 'blocked':
      return { symbol: '!', background: '#d86657', foreground: '#ffffff' }
    default:
      return null
  }
}

export function drawCollectible(
  context: CanvasRenderingContext2D,
  collectible: SunnyTownCollectible,
  cameraX: number,
  cameraY: number,
) {
  const x = collectible.x - cameraX
  const y = collectible.y - cameraY
  context.save()
  context.translate(x, y)
  context.fillStyle = '#f6c945'
  context.strokeStyle = '#7a5a00'
  context.lineWidth = 2
  context.beginPath()
  for (let index = 0; index < 10; index++) {
    const radius = index % 2 === 0 ? 15 : 7
    const angle = -Math.PI / 2 + (index * Math.PI) / 5
    const px = Math.cos(angle) * radius
    const py = Math.sin(angle) * radius
    if (index === 0) {
      context.moveTo(px, py)
    } else {
      context.lineTo(px, py)
    }
  }
  context.closePath()
  context.fill()
  context.stroke()
  context.restore()
}
