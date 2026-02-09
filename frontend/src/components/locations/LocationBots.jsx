import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { botAPI } from '../../lib/api'
import { useAuth } from '../../context/AuthContext'
import './LocationBots.css'

export default function LocationBots({ slug }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [bots, setBots] = useState([])

  const botLocationSlug = useMemo(() => {
    const currentSlug = user?.locationSlug || user?.location?.slug || ''
    if (slug === 'wayward_pines') {
      return currentSlug.endsWith('cell') ? currentSlug : '29cell'
    }
    return currentSlug || slug || ''
  }, [slug, user?.location?.slug, user?.locationSlug])

  useEffect(() => {
    if (!botLocationSlug) {
      setBots([])
      return
    }

    botAPI.getBots(botLocationSlug)
      .then((data) => {
        setBots(data || [])
      })
      .catch(() => {
        setBots([])
      })
  }, [botLocationSlug])

  if (!bots || bots.length === 0) {
    return null
  }

  const handleAttack = async (botSlug) => {
    if (!botSlug) {
      return
    }
    try {
      await botAPI.attack(botSlug)
      navigate('/fight')
    } catch (error) {
      alert(error.message || 'Ошибка при атаке бота')
    }
  }

  return (
    <div className="location-bots">
      <h3 className="location-bots-title">Боты</h3>
      <div className="location-bots-list">
        {bots.map((bot) => (
          <div key={bot.id} className="location-bot-item">
            <span className="location-bot-name">
              {bot.name} [{bot.level}]
            </span>
              <a
                href="#"
                onClick={(e) => {
                  e.preventDefault()
                  handleAttack(bot.slug)
                }}
                className="location-bot-attack-link"
              >
              напасть
            </a>
          </div>
        ))}
      </div>
    </div>
  )
}
