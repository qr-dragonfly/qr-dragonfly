import QRCode from 'qrcode'

export type QrFormat = 'png' | 'jpeg' | 'svg' | 'eps'

export async function generateQrDataUrl(text: string): Promise<string> {
  return QRCode.toDataURL(text, {
    errorCorrectionLevel: 'M',
    margin: 2,
    scale: 6,
  })
}

export async function generateQrInFormat(text: string, format: QrFormat): Promise<string> {
  const options = {
    errorCorrectionLevel: 'M',
    margin: 2,
    scale: 6,
  } as const

  switch (format) {
    case 'png':
      return QRCode.toDataURL(text, { ...options, type: 'image/png' })
    
    case 'jpeg':
      return QRCode.toDataURL(text, { ...options, type: 'image/jpeg' })
    
    case 'svg': {
      const svgString = await QRCode.toString(text, {
        type: 'svg',
        errorCorrectionLevel: 'M',
        margin: 2,
      })
      const blob = new Blob([svgString], { type: 'image/svg+xml' })
      return URL.createObjectURL(blob)
    }
    
    case 'eps': {
      // EPS generation via canvas conversion
      const canvas = document.createElement('canvas')
      await QRCode.toCanvas(canvas, text, {
        errorCorrectionLevel: 'M',
        margin: 2,
        scale: 6,
      })
      
      // Convert canvas to EPS format
      const width = canvas.width
      const height = canvas.height
      const ctx = canvas.getContext('2d')
      if (!ctx) throw new Error('Failed to get canvas context')
      
      const imageData = ctx.getImageData(0, 0, width, height)
      const pixels = imageData.data
      
      // Generate EPS header and data
      let eps = `%!PS-Adobe-3.0 EPSF-3.0\n`
      eps += `%%BoundingBox: 0 0 ${width} ${height}\n`
      eps += `%%Creator: QR Dragonfly\n`
      eps += `%%Title: QR Code\n`
      eps += `%%EndComments\n\n`
      eps += `/pix ${width} string def\n`
      eps += `${width} ${height} 8\n`
      eps += `[${width} 0 0 -${height} 0 ${height}]\n`
      eps += `{currentfile pix readhexstring pop}\n`
      eps += `false 3\n`
      eps += `colorimage\n`
      
      // Convert pixel data to hex
      for (let i = 0; i < pixels.length; i += 4) {
        const r = (pixels[i] ?? 0).toString(16).padStart(2, '0')
        const g = (pixels[i + 1] ?? 0).toString(16).padStart(2, '0')
        const b = (pixels[i + 2] ?? 0).toString(16).padStart(2, '0')
        eps += r + g + b
        if ((i / 4 + 1) % 13 === 0) eps += '\n'
      }
      
      eps += '\n%%EOF\n'
      
      const blob = new Blob([eps], { type: 'application/postscript' })
      return URL.createObjectURL(blob)
    }
    
    default:
      throw new Error(`Unsupported format: ${format}`)
  }
}

export function getFormatExtension(format: QrFormat): string {
  return format
}

export function getFormatMimeType(format: QrFormat): string {
  switch (format) {
    case 'png':
      return 'image/png'
    case 'jpeg':
      return 'image/jpeg'
    case 'svg':
      return 'image/svg+xml'
    case 'eps':
      return 'application/postscript'
    default:
      return 'application/octet-stream'
  }
}
