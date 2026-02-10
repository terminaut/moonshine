import './MoonshineCity.css'

export default function MoonshineCity() {
  const imageUrl = `/assets/images/locations/cities/moonshine/bg.jpg?v=${Date.now()}`
  
  return (
    <div className="moonshine-city-container">
      <div className="moonshine-city-bg">
        <img 
          src={imageUrl}
          alt="Moonshine City Background" 
          className="moonshine-city-bg-image"
        />
      </div>
    </div>
  )
}
