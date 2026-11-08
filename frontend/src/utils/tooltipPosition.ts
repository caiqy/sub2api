export interface ViewportBounds {
  width: number
  height: number
}

export interface TargetRect {
  left: number
  right: number
  top: number
  bottom: number
  width: number
  height: number
}

export interface PanelSize {
  width: number
  height: number
}

export function computeTooltipPosition(
  targetRect: TargetRect,
  panelSize: PanelSize,
  viewport: ViewportBounds,
  margin = 12
): { x: number; y: number } {
  const defaultX = targetRect.right + 8
  const defaultY = targetRect.top + targetRect.height / 2

  let x = defaultX
  let y = defaultY

  // Horizontal clamping: if placing to the right overflows the viewport
  if (viewport.width > 0 && defaultX + panelSize.width + margin > viewport.width) {
    const leftPlacement = targetRect.left - panelSize.width - 8
    if (leftPlacement >= margin) {
      x = leftPlacement
    } else {
      const maxX = Math.max(margin, viewport.width - panelSize.width - margin)
      x = Math.max(margin, Math.min(maxX, defaultX))
    }
  }

  // Vertical clamping: ensure panel (centered vertically via -translate-y-1/2) fits in viewport
  if (viewport.height > 0) {
    const minY = margin + panelSize.height / 2
    const maxY = Math.max(minY, viewport.height - margin - panelSize.height / 2)
    y = Math.max(minY, Math.min(maxY, defaultY))
  }

  return { x, y }
}
