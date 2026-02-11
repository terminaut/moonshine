const imageCache = new Set()

export function preloadImages(sources) {
  if (!Array.isArray(sources) || sources.length === 0) {
    return
  }

  for (const source of sources) {
    if (!source || imageCache.has(source)) {
      continue
    }

    const image = new Image()
    image.decoding = 'async'
    image.src = source
    imageCache.add(source)
  }
}

