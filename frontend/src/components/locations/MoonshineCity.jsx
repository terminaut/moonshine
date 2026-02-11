import { useEffect } from 'react'
import { preloadImages } from '../../lib/imageCache'
import './MoonshineCity.css'

export default function MoonshineCity() {
  const imageUrl = '/assets/images/locations/cities/moonshine/bg.jpg'

  useEffect(() => {
    preloadImages([imageUrl])
  }, [imageUrl])
  
  return (
    <div className="moonshine-city-container">
      <div className="moonshine-city-bg">
        <img 
          src={imageUrl}
          alt="Moonshine City Background" 
          className="moonshine-city-bg-image"
          decoding="async"
        />
      </div>
    </div>
  )
}
