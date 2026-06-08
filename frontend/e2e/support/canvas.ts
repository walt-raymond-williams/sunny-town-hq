import type { Locator } from '@playwright/test'

export async function hasNonBlankCanvasPixels(canvas: Locator): Promise<boolean> {
  return canvas.evaluate((element) => {
    if (!(element instanceof HTMLCanvasElement)) {
      return false
    }
    const context = element.getContext('2d')
    if (!context || element.width === 0 || element.height === 0) {
      return false
    }

    const sampleWidth = Math.min(element.width, 160)
    const sampleHeight = Math.min(element.height, 120)
    const image = context.getImageData(0, 0, sampleWidth, sampleHeight).data
    for (let index = 3; index < image.length; index += 4) {
      if (image[index] !== 0) {
        return true
      }
    }
    return false
  })
}
