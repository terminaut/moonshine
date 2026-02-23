import { useCallback, useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import PlayerHeader from '../components/PlayerHeader'
import EquipmentDisplay from '../components/EquipmentDisplay'
import { useAuth } from '../context/AuthContext'
import { avatarAPI, equipmentAPI, userAPI } from '../lib/api'
import './EquipmentItems.css'
import './Profile.css'

export default function Profile() {
  const { user: authUser, logout, refetchUser } = useAuth()
  const navigate = useNavigate()
  const [user, setUser] = useState(authUser)
  const [avatarCacheBuster] = useState(() => Date.now())
  const [loading, setLoading] = useState(!authUser)
  const [error, setError] = useState(null)
  const [activeTab, setActiveTab] = useState('inventory')
  const [inventory, setInventory] = useState([])
  const [inventoryLoading, setInventoryLoading] = useState(false)
  const [equippedItems, setEquippedItems] = useState({})
  const [avatars, setAvatars] = useState([])
  const [avatarsLoading, setAvatarsLoading] = useState(false)
  const [notification, setNotification] = useState(null)
  const [equipping, setEquipping] = useState(false)
  
  const updateInProgressRef = useRef(false)

  useEffect(() => {
    if (authUser) {
      setUser(authUser)
    }
  }, [authUser])

  useEffect(() => {
    if (notification) {
      const timer = setTimeout(() => {
        setNotification(null)
      }, 3000)
      return () => clearTimeout(timer)
    }
  }, [notification])

  const showNotification = (message, type = 'error') => {
    setNotification({ message, type })
  }

  const fetchUserData = useCallback(() => {
    return userAPI.getCurrentUser()
      .then((userData) => {
        setUser(userData)
        return userData
      })
      .catch((err) => {
        console.error('[Profile] Error loading profile:', err)
        setError('Ошибка загрузки профиля')
        throw err
      })
  }, [])

  useEffect(() => {
    let mounted = true
    
    const loadData = async () => {
      setLoading(true)
      try {
        await fetchUserData()
        if (!mounted) return
        
        userAPI.getEquippedItems()
          .then((equipped) => {
            if (mounted) {
              setEquippedItems(equipped)
            }
          })
          .catch((err) => {
            if (mounted) {
              console.error('[Profile] Error loading equipped items:', err)
              setEquippedItems({})
            }
          })
      } catch (err) {
        if (mounted) {
          console.error('[Profile] Error loading profile:', err)
          setError('Ошибка загрузки профиля')
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }
    
    loadData()
    
    return () => {
      mounted = false
    }
  }, [fetchUserData])

  useEffect(() => {
    if (activeTab === 'inventory') {
      setInventoryLoading(true)
      userAPI.getInventory()
        .then((items) => {
          setInventory(items)
          setInventoryLoading(false)
        })
        .catch((err) => {
          console.error('[Profile] Error loading inventory:', err)
          setInventory([])
          setInventoryLoading(false)
        })
    } else if (activeTab === 'settings') {
      if (avatars.length === 0) {
        setAvatarsLoading(true)
        avatarAPI.getAll()
          .then((avatarsList) => {
            setAvatars(avatarsList)
            setAvatarsLoading(false)
          })
          .catch((err) => {
            console.error('[Profile] Error loading avatars:', err)
            setAvatars([])
            setAvatarsLoading(false)
          })
      }
    }
  }, [activeTab, avatars.length])

  if (loading) return <div className="profile-container">Загрузка...</div>
  if (error) return <div className="profile-container">{error}</div>
  if (!user) return <div className="profile-container">Пользователь не найден</div>

  const handleLogout = () => {
    logout()
    localStorage.clear()
    navigate('/signin')
  }

  const handleBack = () => {
    navigate(-1)
  }

  const handleTakeOn = async (item) => {
    if (!item.slug) {
      showNotification('Slug предмета не определен', 'error')
      return
    }

    if (user.level < item.requiredLevel) {
      showNotification(`Недостаточный уровень! Требуется уровень ${item.requiredLevel}`, 'error')
      return
    }

    if (updateInProgressRef.current) {
      return
    }

    try {
      updateInProgressRef.current = true
      setEquipping(true)
      await equipmentAPI.takeOn(item.slug)
      
      const [items, equipped, updatedUser] = await Promise.all([
        userAPI.getInventory(),
        userAPI.getEquippedItems(),
        refetchUser(),
      ])
      
      if (updatedUser) {
        setUser(updatedUser)
      }
      setInventory(items)
      setEquippedItems(equipped)
    } catch (error) {
      console.error('[Profile] Error equipping item:', error)
      let errorMessage = 'Неизвестная ошибка'
      if (error.message.includes('not in inventory')) {
        errorMessage = 'Предмет не в инвентаре'
      } else if (error.message.includes('insufficient level')) {
        errorMessage = 'Недостаточный уровень'
      } else if (error.message.includes('not found')) {
        errorMessage = 'Предмет не найден'
      } else if (error.message.includes('invalid equipment type')) {
        errorMessage = 'Неверный тип экипировки'
      } else {
        errorMessage = error.message
      }
      showNotification(errorMessage, 'error')
    } finally {
      updateInProgressRef.current = false
      setEquipping(false)
    }
  }

  const handleTakeOff = async (slotName) => {
    const equippedItem = equippedItems[slotName] || equippedItems[slotName.toLowerCase()]
    if (!equippedItem) {
      return
    }

    if (updateInProgressRef.current) {
      return
    }

    try {
      updateInProgressRef.current = true
      setEquipping(true)
      await equipmentAPI.takeOff(slotName)
      
      const [items, equipped, updatedUser] = await Promise.all([
        userAPI.getInventory(),
        userAPI.getEquippedItems(),
        refetchUser(),
      ])
      
      if (updatedUser) {
        setUser(updatedUser)
      }
      setInventory(items)
      setEquippedItems(equipped)
    } catch (error) {
      console.error('[Profile] Error removing item:', error)
      let errorMessage = 'Неизвестная ошибка'
      if (error.message.includes('no item equipped')) {
        errorMessage = 'В этом слоте нет предмета'
      } else if (error.message.includes('invalid slot')) {
        errorMessage = 'Неверный слот'
      } else {
        errorMessage = error.message
      }
      showNotification(errorMessage, 'error')
    } finally {
      updateInProgressRef.current = false
      setEquipping(false)
    }
  }

  const handleSelectAvatar = async (avatarId) => {
    try {
      await userAPI.updateProfile({ avatarId })
      await refetchUser()
    } catch (error) {
      console.error('[Profile] Error updating avatar:', error)
      let errorMessage = 'Неизвестная ошибка'
      if (error.message.includes('not found')) {
        errorMessage = 'Аватар не найден'
      } else if (error.message.includes('Unauthorized')) {
        errorMessage = 'Необходимо войти в систему'
      } else {
        errorMessage = error.message
      }
      showNotification(errorMessage, 'error')
    }
  }

  const handleSell = async (item) => {
    if (!item.slug) {
      showNotification('Slug предмета не определен', 'error')
      return
    }

    try {
      await equipmentAPI.sell(item.slug)
      const items = await userAPI.getInventory()
      await refetchUser()
      setInventory(items)
      showNotification(`Продано за ${item.price} золота`, 'success')
    } catch (error) {
      console.error('[Profile] Error selling item:', error)
      let errorMessage = 'Неизвестная ошибка'
      if (error.message.includes('not owned')) {
        errorMessage = 'У вас нет этого предмета'
      } else if (error.message.includes('not found')) {
        errorMessage = 'Предмет не найден'
      } else {
        errorMessage = error.message
      }
      showNotification(errorMessage, 'error')
    }
  }

  const normalizeImagePath = (img, bustAvatarCache = false) => {
    if (!img) return null
    let p = img
    if (p.startsWith('/')) p = p.slice(1)
    p = p.replace(/^frontend\/assets\/images\//, '')
    if (p.startsWith('assets/images/')) p = p.replace(/^assets\/images\//, '')
    const normalizedPath = `/assets/images/${p}`
    if (bustAvatarCache && p.startsWith('players/avatars/')) {
      return `${normalizedPath}?v=${avatarCacheBuster}`
    }
    return normalizedPath
  }

  const handleSlotClick = (slotName) => {
    if (!equipping) {
      handleTakeOff(slotName)
    }
  }

  return (
    <div className="profile-container">
      {notification && (
        <div className={`notification-toast notification-${notification.type}`}>
          {notification.message}
        </div>
      )}

      <div className="profile-main-block">
        <div className="profile-header">
          <PlayerHeader 
            user={user} 
            fullWidth={true}
          />
          <div className="profile-header-actions">
            <button 
              onClick={handleBack}
              className="profile-back-button"
              title="Назад"
            >
              ← Назад
            </button>
            <button 
              onClick={handleLogout} 
              className="profile-logout-button"
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
        <div className="profile-content">
          <div className="profile-equipment-wrapper">
            <EquipmentDisplay
              user={user}
              equippedItems={equippedItems}
              onSlotClick={handleSlotClick}
              avatarCacheBuster={avatarCacheBuster}
            />
          </div>

        <div className="profile-sidebar">
          <div className="profile-tabs">
            <button
              className={`profile-tab ${activeTab === 'inventory' ? 'active' : ''}`}
              onClick={() => setActiveTab('inventory')}
            >
              Инвентарь
            </button>
            <button
              className={`profile-tab ${activeTab === 'settings' ? 'active' : ''}`}
              onClick={() => setActiveTab('settings')}
            >
              Настройки
            </button>
          </div>

          {activeTab === 'inventory' && (
            <div className="profile-inventory-content">
              {inventoryLoading ? (
                <div>Загрузка инвентаря...</div>
              ) : (
                <div className="equipment-items-list">
                  {inventory.length === 0 ? (
                    <p>Инвентарь пуст</p>
                  ) : (
                    inventory.map((item, index) => (
                      <div key={`${item.id}-${item.slug}-${index}`} className="equipment-item-card">
                        {item.image && (
                          <div className="equipment-item-image-frame">
                            <img
                              src={normalizeImagePath(item.image)}
                              alt={item.name}
                              className="equipment-item-image"
                            />
                          </div>
                        )}
                        <div className="equipment-item-info">
                          <h3>{item.name}</h3>
                          {item.equipment_type && (
                            <div className="equipment-item-type">Тип: {item.equipment_type}</div>
                          )}
                          <div className="equipment-item-stats">
                            <div>Уровень: {item.requiredLevel}</div>
                            {item.attack > 0 && <div>Атака: {item.attack}</div>}
                            {item.defense > 0 && <div>Защита: {item.defense}</div>}
                            {item.hp > 0 && <div>HP: {item.hp}</div>}
                          </div>
                          <div className="equipment-item-buttons">
                            <button 
                              className="equipment-item-equip-button"
                              onClick={() => handleTakeOn(item)}
                              disabled={user.level < item.requiredLevel || equipping}
                            >
                              {user.level < item.requiredLevel ? `Нужен ${item.requiredLevel} ур.` : (equipping ? '...' : 'Надеть')}
                            </button>
                            <button 
                              className="equipment-item-sell-button"
                              onClick={() => handleSell(item)}
                            >
                              Продать ({item.price})
                            </button>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>
          )}

          {activeTab === 'settings' && (
            <div className="profile-settings-content">
              <h2>Выбор аватара</h2>
              {avatarsLoading ? (
                <div>Загрузка аватаров...</div>
              ) : (
                <div className="avatars-list">
                  {avatars.length === 0 ? (
                    <p>Аватары не найдены</p>
                  ) : (
                    avatars.map((avatar) => {
                      const isSelected = user.avatar === avatar.image
                      return (
                        <div
                          key={avatar.id}
                          className={`avatar-item ${isSelected ? 'selected' : ''}`}
                          onClick={() => handleSelectAvatar(avatar.id)}
                        >
                          <img 
                            src={normalizeImagePath(avatar.image, true)} 
                            alt={`Avatar ${avatar.id}`}
                            className="avatar-image-select"
                            onError={(e) => {
                              e.target.style.display = 'none'
                            }}
                          />
                          {isSelected && <div className="avatar-selected-badge">✓</div>}
                        </div>
                      )
                    })
                  )}
                </div>
              )}
            </div>
          )}
        </div>
        </div>
      </div>
    </div>
  )
}
