import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MapGrid from './MapGrid'
import { botAPI } from '../../lib/api'
import { useAuth } from '../../context/AuthContext'
import './WaywardPines.css'

export default function WaywardPines() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [bots, setBots] = useState([])
  const [loading, setLoading] = useState(true)
  const currentSlug = user?.locationSlug || user?.location?.slug || ''
  const currentCellSlug = currentSlug.endsWith('cell') ? currentSlug : '29cell'

  useEffect(() => {
    botAPI.getBots(currentCellSlug)
      .then((data) => {
        setBots(data)
        setLoading(false)
      })
      .catch((err) => {
        console.error('[WaywardPines] Error loading bots:', err)
        setLoading(false)
      })
  }, [currentCellSlug])

  const handleAttack = async (botSlug) => {
    if (!botSlug) {
      return
    }
    try {
      await botAPI.attack(botSlug)
      navigate('/fight')
    } catch (err) {
      console.error('[WaywardPines] Error attacking bot:', err)
      alert(err.message || 'Ошибка при атаке бота')
    }
  }

  return (
    <div className="location-inner-content">
      <div className="wayward-pines-content">
        <div className="wayward-pines-map">
          <div className="wayward-pines-header">
            <img
              src="/assets/images/locations/wayward_pines/icon.png"
              alt="Wayward Pines"
              className="wayward-pines-icon"
            />
            <h2>Wayward Pines</h2>
          </div>
          <MapGrid locationSlug="wayward_pines" />
        </div>
        <div className="wayward-pines-bots">
          <h3>Боты</h3>
          {loading ? (
            <div>Загрузка...</div>
          ) : bots.length === 0 ? (
            <div>Боты не найдены</div>
          ) : (
            <div className="bots-list">
              {bots.map((bot) => (
                <div key={bot.id} className="bot-item">
                  <span>[{bot.level}] {bot.name} </span>
                  <a
                    href="#"
                    onClick={(e) => {
                      e.preventDefault()
                      handleAttack(bot.slug)
                    }}
                    className="bot-attack-link"
                  >
                    атаковать
                  </a>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
