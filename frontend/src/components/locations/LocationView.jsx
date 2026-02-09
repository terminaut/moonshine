import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { locationAPI } from '../../lib/api'
import PlayerHeader from '../PlayerHeader'
import './LocationView.css'

export default function LocationView({ slug, children }) {
  const { user, logout, refetchUser } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    localStorage.clear()
    navigate('/signin')
  }

  const handleLocationMove = async (e, targetSlug) => {
    e.preventDefault()
    
    try {
      await locationAPI.move(targetSlug)
      await refetchUser()
      navigate(`/locations/${targetSlug}`)
    } catch (error) {
      console.error('[LocationView] Error moving to location:', error)
      let errorMessage = 'Неизвестная ошибка'
      if (error.message.includes('not connected')) {
        errorMessage = 'Невозможно переместиться в эту локацию отсюда'
      } else if (error.message.includes('already at this location')) {
        errorMessage = 'Вы уже находитесь в этой локации'
      } else if (error.message.includes('not found')) {
        errorMessage = 'Локация не найдена'
      } else {
        errorMessage = error.message
      }
      alert(`Ошибка перемещения: ${errorMessage}`)
    }
  }

  const isCity = slug === 'moonshine'
  const isWaywardPines = slug === 'wayward_pines'

  const cityLinks = [
    { to: '/locations/weapon_shop', label: 'Оружейная' },
    { to: '/locations/shop_of_artifacts', label: 'Артефакты' },
    { to: '/locations/wayward_pines', label: 'Выйти из города' },
  ]

  const shopLinks = [
    { to: '/locations/moonshine', label: 'Главная площадь' },
  ]

  const isOnCell29 = user?.locationSlug === '29cell'

  const waywardPinesLinks = isOnCell29
    ? [{ to: '/locations/moonshine', label: 'Город' }]
    : []

  let links = shopLinks
  if (isCity) {
    links = cityLinks
  } else if (isWaywardPines) {
    links = waywardPinesLinks
  }

  return (
    <div className="location-view">
      <div className="location-main-block">
        <div className="location-header">
          <PlayerHeader />
          <div className="location-nav-links">
            {links.map((link) => {
              const targetSlug = link.to.split('/').pop()
              return (
                <a
                  key={link.to}
                  href={link.to}
                  onClick={(e) => handleLocationMove(e, targetSlug)}
                  className="fantasy-button"
                >
                  {link.label}
                </a>
              )
            })}
            <button 
              onClick={handleLogout} 
              className="logout-door-button"
              title="Выйти из игры"
            >
              <svg 
                width="24" 
                height="24" 
                viewBox="0 0 24 24" 
                fill="none" 
                xmlns="http://www.w3.org/2000/svg"
              >
                <path 
                  d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" 
                  fill="currentColor"
                />
              </svg>
            </button>
          </div>
        </div>
        <div className="location-content">
          {children}
        </div>
      </div>
    </div>
  )
}
