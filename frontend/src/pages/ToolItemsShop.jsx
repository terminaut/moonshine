import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toolItemsAPI } from '../lib/api'
import PlayerHeader from '../components/PlayerHeader'
import { useAuth } from '../context/AuthContext'
import './EquipmentItems.css'

export default function ToolItemsShop() {
  const { logout, refetchUser } = useAuth()
  const navigate = useNavigate()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [notification, setNotification] = useState(null)

  useEffect(() => {
    if (!notification) {
      return undefined
    }

    const timer = setTimeout(() => {
      setNotification(null)
    }, 3000)

    return () => clearTimeout(timer)
  }, [notification])

  useEffect(() => {
    setLoading(true)
    toolItemsAPI
      .getAll()
      .then((data) => {
        setItems(data)
        setLoading(false)
      })
      .catch((err) => {
        console.error('[ToolItemsShop] Error loading tool items:', err)
        setError('Ошибка загрузки предметов')
        setLoading(false)
      })
  }, [])

  const showNotification = (message, type = 'error') => {
    setNotification({ message, type })
  }

  const handleBack = () => {
    navigate('/locations/moonshine')
  }

  const handleLogout = () => {
    logout()
    localStorage.clear()
    navigate('/signin')
  }

  const handleBuy = async (itemSlug) => {
    if (!itemSlug) {
      showNotification('Slug предмета не определен', 'error')
      return
    }

    try {
      await toolItemsAPI.buy(itemSlug)
      await refetchUser()
      showNotification('Предмет успешно куплен!', 'success')
    } catch (err) {
      console.error('[ToolItemsShop] Error buying item:', err)
      let errorMessage = 'Неизвестная ошибка'
      if (err.message.includes('insufficient gold')) {
        errorMessage = 'Недостаточно золота'
      } else if (err.message.includes('not found')) {
        errorMessage = 'Предмет не найден'
      } else if (err.message.includes('unauthorized')) {
        errorMessage = 'Необходимо войти в систему'
      } else {
        errorMessage = err.message
      }
      showNotification(errorMessage, 'error')
    }
  }

  const normalizeImagePath = (img) => {
    if (!img) return null
    let p = img
    if (p.startsWith('/')) p = p.slice(1)
    p = p.replace(/^frontend\/assets\/images\//, '')
    if (p.startsWith('assets/images/')) p = p.replace(/^assets\/images\//, '')
    if (/^tool_items\/[^/]+\/[^/]+$/.test(p)) {
      const parts = p.split('/')
      p = `tool_items/${parts[2]}`
    }
    const filename = p.split('/').pop() || ''
    if (filename && !filename.includes('.')) {
      p = `${p}.png`
    }
    return `/assets/images/${p}`
  }

  if (loading) {
    return (
      <div className="equipment-items-container">
        <div className="equipment-items-main-block">
          <div className="equipment-items-header">
            <PlayerHeader />
          </div>
          <div className="equipment-items-content">Загрузка...</div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="equipment-items-container">
        <div className="equipment-items-main-block">
          <div className="equipment-items-header">
            <PlayerHeader />
          </div>
          <div className="equipment-items-content">{error}</div>
        </div>
      </div>
    )
  }

  return (
    <div className="equipment-items-container">
      {notification && (
        <div className={`notification-toast notification-${notification.type}`}>
          {notification.message}
        </div>
      )}

      <div className="equipment-items-main-block">
        <div className="equipment-items-header">
          <PlayerHeader />
          <div className="equipment-items-header-actions">
            <button
              onClick={handleBack}
              className="equipment-items-back-button"
              title="Назад"
            >
              ← Назад
            </button>
            <button
              onClick={handleLogout}
              className="equipment-items-logout-button"
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
        <div className="equipment-items-content">
          <div className="equipment-items-list">
            {items.length === 0 ? (
              <p>Предметы не найдены</p>
            ) : (
              items.map((item) => (
                <div key={item.id} className="equipment-item-card">
                  {item.image && (
                    <img
                      src={normalizeImagePath(item.image)}
                      alt={item.name}
                      className="equipment-item-image"
                    />
                  )}
                  <div className="equipment-item-info">
                    <h3>{item.name}</h3>
                    <div className="equipment-item-stats">
                      <div>Цена: {item.price} зол.</div>
                    </div>
                    <button
                      className="equipment-item-buy-button"
                      onClick={() => handleBuy(item.slug)}
                    >
                      Купить
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
