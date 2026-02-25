import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { userAPI, potionsAPI } from '../lib/api'
import { fightAPI } from '../lib/fightAPI'
import EquipmentDisplay from '../components/EquipmentDisplay'
import GoldIcon from '../components/GoldIcon'
import { useAuth } from '../context/AuthContext'
import './Fight.css'

const BODY_PARTS = [
  { value: 'HEAD', label: 'Голова' },
  { value: 'NECK', label: 'Шея' },
  { value: 'CHEST', label: 'Грудь' },
  { value: 'BELT', label: 'Пояс' },
  { value: 'LEGS', label: 'Ноги' },
]

export default function Fight() {
  const navigate = useNavigate()
  const { logout, user: authUser, refetchUser } = useAuth()
  const [user, setUser] = useState(null)
  const [bot, setBot] = useState(null)
  const [fight, setFight] = useState(null)
  const [equippedItems, setEquippedItems] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [selectedAttack, setSelectedAttack] = useState('HEAD')
  const [selectedDefense, setSelectedDefense] = useState('HEAD')
  const [hitting, setHitting] = useState(false)
  const [usingPotion, setUsingPotion] = useState(null)

  useEffect(() => {
    setLoading(true)
    Promise.all([
      fightAPI.getCurrentFight(),
      userAPI.getEquippedItems()
    ])
      .then(([fightData, equipped]) => {
        console.log('[Fight] Initial load - fightData:', fightData)
        console.log('[Fight] Initial load - rounds:', fightData.fight?.rounds)
        console.log('[Fight] Initial load - status:', fightData.fight?.status)
        
        setUser(fightData.user)
        setBot(fightData.bot)
        setFight(fightData.fight)
        setEquippedItems(equipped)
        
        if (fightData.fight?.status === 'FINISHED') {
          refetchUser().catch(err => {
            console.error('[Fight] Error refetching user on load:', err)
          })
        }
        
        setLoading(false)
      })
      .catch((err) => {
        console.error('[Fight] Error loading fight data:', err)
        if (err.message.includes('no active fight')) {
          navigate('/', { replace: true })
        } else if (err.message.includes('user not found') || err.message.includes('Unauthorized')) {
          localStorage.removeItem('token')
          navigate('/signin', { replace: true })
        } else {
          setError(err.message || 'Ошибка загрузки данных боя')
        }
        setLoading(false)
      })
  }, [navigate])

  const handleLogout = () => {
    logout()
    localStorage.clear()
    navigate('/signin')
  }

  const handleHit = async () => {
    if (!selectedAttack || !selectedDefense) {
      return
    }

    if (fight?.status === 'FINISHED') {
      setError('Бой уже завершен')
      return
    }

    setHitting(true)
    setError(null)
    try {
      console.log('[Fight] Hitting with:', { attack: selectedAttack, defense: selectedDefense })
      const hitResponse = await fightAPI.hit(selectedAttack, selectedDefense)
      console.log('[Fight] Hit response:', hitResponse)
      console.log('[Fight] Hit response rounds:', hitResponse.fight?.rounds)
      
      if (!hitResponse || !hitResponse.user || !hitResponse.bot || !hitResponse.fight) {
        console.error('[Fight] Invalid response structure:', hitResponse)
        setError('Неверный формат ответа от сервера')
        return
      }
      
      setUser(hitResponse.user)
      setBot(hitResponse.bot)
      setFight(hitResponse.fight)
      
      console.log('[Fight] Updated state - rounds:', hitResponse.fight.rounds)
      console.log('[Fight] First round HP:', hitResponse.fight.rounds[0])
      console.log('[Fight] Fight status:', hitResponse.fight.status)
      
      if (hitResponse.fight.status === 'FINISHED') {
        refetchUser().catch(err => {
          console.error('[Fight] Error refetching user after fight:', err)
        })
      }
      
      const equipped = await userAPI.getEquippedItems()
      setEquippedItems(equipped)
    } catch (err) {
      console.error('[Fight] Error hitting:', err)
      if (err.message && err.message.includes('no active fight')) {
        setError('Бой завершен')
        const fightData = await fightAPI.getCurrentFight().catch(() => null)
        if (fightData?.fight) {
          setFight(fightData.fight)
        }
      } else {
        setError(err.message || 'Ошибка при ударе')
      }
    } finally {
      setHitting(false)
    }
  }

  const handleFinishFight = async () => {
    await refetchUser().catch(err => {
      console.error('[Fight] Error refetching user before navigation:', err)
    })
    
    const locationSlug = authUser?.locationSlug || user?.locationSlug || 'moonshine'
    
    if (locationSlug.includes('cell')) {
      navigate('/locations/wayward_pines')
    } else {
      navigate(`/locations/${locationSlug}`)
    }
  }

  const potionStatToType = (stat) => {
    switch (stat) {
      case 'CURRENT_HP': return 'current_hp'
      case 'ATTACK': return 'attack'
      case 'DEFENSE': return 'defense'
      default: return 'current_hp'
    }
  }

  const handleUsePotion = async (slot) => {
    if (usingPotion || fight?.status === 'FINISHED') return
    setUsingPotion(slot)
    try {
      await potionsAPI.use(slot)
      const [fightData, equipped] = await Promise.all([
        fightAPI.getCurrentFight(),
        userAPI.getEquippedItems(),
      ])
      setUser(fightData.user)
      setBot(fightData.bot)
      setFight(fightData.fight)
      setEquippedItems(equipped)
    } catch (err) {
      console.error('[Fight] Error using potion:', err)
      setError(err.message || 'Ошибка при использовании эликсира')
    } finally {
      setUsingPotion(null)
    }
  }

  if (loading) {
    return (
      <div className="fight-container">
        <div className="fight-main-block">
          <div className="fight-header">
            <div className="fight-header-actions">
              <button onClick={handleLogout} className="fight-logout-button">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" fill="currentColor" />
                </svg>
              </button>
            </div>
          </div>
          <div className="fight-content">
            <p>Загрузка боя...</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="fight-container">
        <div className="fight-main-block">
          <div className="fight-header">
            <div className="fight-header-actions">
              <button onClick={handleLogout} className="fight-logout-button">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" fill="currentColor" />
                </svg>
              </button>
            </div>
          </div>
          <div className="fight-content">
            <p className="fight-error-text">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  if (!user || !bot) {
    console.error('[Fight] Missing data:', { user: !!user, bot: !!bot, fight: !!fight })
    return (
      <div className="fight-container">
        <div className="fight-main-block">
          <div className="fight-header">
            <div className="fight-header-actions">
              <button onClick={handleLogout} className="fight-logout-button">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" fill="currentColor" />
                </svg>
              </button>
            </div>
          </div>
          <div className="fight-content">
            <p className="fight-error-text">Данные боя не найдены</p>
          </div>
        </div>
      </div>
    )
  }

  const getBotImageSrc = () => {
    if (!bot.avatar) {
      return '/assets/images/bots/rat2.jpg'
    }
    
    let path = bot.avatar.trim()
    
    if (path.startsWith('/')) {
      path = path.slice(1)
    }
    
    path = path.replace(/^frontend\/assets\/images\//, '')
    path = path.replace(/^assets\/images\//, '')
    
    if (path.startsWith('images/')) {
      path = path.replace(/^images\//, '')
    }
    
    if (!path.includes('.')) {
      path = `${path}.jpg`
    }
    
    if (path.includes('rat') && !path.includes('bots/')) {
      path = `bots/${path}`
    } else if (!path.startsWith('bots/')) {
      path = `bots/${path}`
    }
    
    return `/assets/images/${path}`
  }
  
  const botImageSrc = getBotImageSrc()

  const firstRound = fight?.rounds?.[0]
  
  console.log('[Fight] Fight status:', fight?.status)
  console.log('[Fight] All rounds:', fight?.rounds?.map(r => ({ 
    playerHp: r.playerHp, 
    botHp: r.botHp, 
    status: r.status 
  })))
  console.log('[Fight] firstRound (rounds[0]):', { 
    playerHp: firstRound?.playerHp, 
    botHp: firstRound?.botHp, 
    status: firstRound?.status 
  })
  
  const playerCurrentHp = firstRound?.playerHp ?? 0
  const playerMaxHp = user.hp || 0
  const botCurrentHp = firstRound?.botHp ?? 0
  const botMaxHp = bot.hp || 0

  console.log('[Fight] Display values - playerCurrentHp:', playerCurrentHp, 'botCurrentHp:', botCurrentHp)
  console.log('[Fight] Fight exp:', fight?.exp, 'droppedGold:', fight?.droppedGold)

  const botUserData = {
    username: bot.name,
    name: bot.name,
    level: bot.level,
    hp: bot.hp,
    currentHp: botCurrentHp,
    avatar: bot.avatar,
    attack: bot.attack,
    defense: bot.defense,
  }

  const finishedRounds = (fight?.rounds?.filter(round => {
    const status = round.status?.toUpperCase()
    return status === 'FINISHED'
  }) || []).reverse()

  return (
    <div className="fight-container">
      <div className="fight-main-block">
        <div className="fight-header">
          <div className="fight-header-actions">
            <button onClick={handleLogout} className="fight-logout-button">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" fill="currentColor" />
              </svg>
            </button>
          </div>
        </div>
        
        <div className="fight-potions-bar">
          {['potion1', 'potion2', 'potion3'].map((slot) => {
            const potion = equippedItems?.[slot]
            const hasPotion = potion && potion.stat
            return (
              <button
                key={slot}
                className={`fight-potion-slot ${hasPotion ? 'has-potion' : ''}`}
                disabled={!hasPotion || fight?.status === 'FINISHED' || usingPotion === slot}
                onClick={() => handleUsePotion(slot)}
                title={hasPotion ? `Использовать: ${potion.name}` : 'Пусто'}
              >
                <img
                  src={hasPotion
                    ? `/assets/images/potions/${potionStatToType(potion.stat)}/big.png`
                    : '/assets/images/equipment_items/grid/bag.png'
                  }
                  alt={slot}
                  className="fight-potion-icon"
                />
                {usingPotion === slot && <span className="fight-potion-loading">...</span>}
              </button>
            )
          })}
        </div>

        <div className="fight-content">
          <div className="fight-player-section">
            <div className="fight-player-info">
              <div className="fight-player-title">
                <h2>{user.username}</h2>
                <span className="fight-player-level">[{user.level}]</span>
              </div>
              <div className="fight-hp-bar">
                <div 
                  className="fight-hp-fill" 
                  style={{ 
                    width: `${playerMaxHp > 0 ? Math.round((playerCurrentHp / playerMaxHp) * 100) : 0}%`,
                    backgroundColor: '#dc3545'
                  }}
                ></div>
                <span className="fight-hp-text">
                  {playerCurrentHp}/{playerMaxHp}
                </span>
              </div>
            </div>
            <EquipmentDisplay 
              user={user} 
              equippedItems={equippedItems}
              readonly={true}
            />
          </div>

          <div className="fight-arena-section">
            <div className="fight-vs-text">VS</div>
            
            {fight?.status !== 'FINISHED' && (
              <div className="fight-controls">
                <div className="fight-controls-column">
                  <div className="fight-controls-title">Защита</div>
                  <div className="fight-body-parts-list">
                    {BODY_PARTS.map((part) => (
                      <button
                        key={part.value}
                        className={`fight-body-part-button ${selectedDefense === part.value ? 'selected' : ''}`}
                        onClick={() => setSelectedDefense(part.value)}
                      >
                        {part.label}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="fight-controls-column">
                  <div className="fight-controls-title">Атака</div>
                  <div className="fight-body-parts-list">
                    {BODY_PARTS.map((part) => (
                      <button
                        key={part.value}
                        className={`fight-body-part-button ${selectedAttack === part.value ? 'selected' : ''}`}
                        onClick={() => setSelectedAttack(part.value)}
                      >
                        {part.label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {fight?.status === 'FINISHED' ? (
              <button
                className="fight-hit-button"
                onClick={handleFinishFight}
              >
                Завершить бой
              </button>
            ) : (
              <button
                className="fight-hit-button"
                onClick={handleHit}
                disabled={!selectedAttack || !selectedDefense || hitting || fight?.status === 'FINISHED'}
              >
                {hitting ? 'Удар...' : 'Ударить'}
              </button>
            )}
          </div>

          <div className="fight-bot-section">
            <div className="fight-bot-info">
              <div className="fight-bot-title">
                <h2>{bot.name}</h2>
                <span className="fight-bot-level">[{bot.level}]</span>
              </div>
              <div className="fight-hp-bar">
                <div 
                  className="fight-hp-fill" 
                  style={{ 
                    width: `${botMaxHp > 0 ? Math.round((botCurrentHp / botMaxHp) * 100) : 0}%`,
                    backgroundColor: '#dc3545'
                  }}
                ></div>
                <span className="fight-hp-text">
                  {botCurrentHp}/{botMaxHp}
                </span>
              </div>
            </div>
            <EquipmentDisplay 
              user={botUserData} 
              equippedItems={{}}
              readonly={true}
            />
          </div>
        </div>

        {finishedRounds.length > 0 && (
          <div className="fight-rounds-history">
            <h3 className="fight-rounds-history-title">Лог боя</h3>
            {fight?.status === 'FINISHED' && (fight.exp > 0 || fight.droppedGold > 0) && (
              <div className="fight-rewards">
                <span className="fight-reward-item">
                  Получено {fight.exp} опыта. С {bot.name}[{bot.level}] выпало{' '}
                  <span className="fight-reward-gold">
                    <GoldIcon className="fight-reward-gold-icon" width={18} height={18} />
                    {fight.droppedGold}
                  </span>{' '}
                  золота.
                </span>
              </div>
            )}
            <div className="fight-rounds-list">
              {finishedRounds.map((round, index) => {
                const playerName = `${user.username}[${user.level}]`
                const botName = `${bot.name}[${bot.level}]`
                
                const playerBlocked = round.playerDefensePoint && round.botAttackPoint && 
                  round.playerDefensePoint.toUpperCase() === round.botAttackPoint.toUpperCase()
                const botBlocked = round.botDefensePoint && round.playerAttackPoint && 
                  round.botDefensePoint.toUpperCase() === round.playerAttackPoint.toUpperCase()
                
                const formatTime = (dateString) => {
                  const date = new Date(dateString)
                  return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
                }
                
                const parts = []
                parts.push(formatTime(round.createdAt))
                
                if (playerBlocked) {
                  parts.push(`${playerName} заблокировал удар`)
                  parts.push(`${botName} нанес ${round.botDamage} урона`)
                  if (round.playerDamage > 0) {
                    parts.push(`${playerName} нанес ${round.playerDamage} урона`)
                  }
                } else if (botBlocked) {
                  parts.push(`${botName} заблокировал удар`)
                  parts.push(`${playerName} нанес ${round.playerDamage} урона`)
                  if (round.botDamage > 0) {
                    parts.push(`${botName} нанес ${round.botDamage} урона`)
                  }
                } else {
                  if (round.playerDamage > 0) {
                    parts.push(`${playerName} нанес ${round.playerDamage} урона`)
                  }
                  if (round.botDamage > 0) {
                    parts.push(`${botName} нанес ${round.botDamage} урона`)
                  }
                }
                
                const roundText = parts.join('. ') + '.'
                
                const renderRoundText = (text) => {
                  const parts = text.split(/(\d+ урона)/)
                  return parts.map((part, i) => {
                    if (part.match(/^\d+ урона$/)) {
                      return <span key={i} className="fight-round-damage-value">{part}</span>
                    }
                    return part
                  })
                }
                
                return (
                  <React.Fragment key={round.id || index}>
                    <div className="fight-round-item">
                      <span className="fight-round-text">
                        {renderRoundText(roundText)}
                      </span>
                    </div>
                    {index < finishedRounds.length - 1 && <div className="fight-round-divider"></div>}
                  </React.Fragment>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
