import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MapGrid from './MapGrid'
import { botAPI, resourceAPI } from '../../lib/api'
import { preloadImages } from '../../lib/imageCache'
import { useAuth } from '../../context/AuthContext'
import './WaywardPines.css'

export default function WaywardPines() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [bots, setBots] = useState([])
  const [resources, setResources] = useState([])
  const [botsLoading, setBotsLoading] = useState(true)
  const [resourcesLoading, setResourcesLoading] = useState(true)
  const currentSlug = user?.locationSlug || user?.location?.slug || ''
  const currentCellSlug = currentSlug.endsWith('cell') ? currentSlug : '29cell'

  useEffect(() => {
    preloadImages(['/assets/images/locations/wayward_pines/icon.png'])
  }, [])

  useEffect(() => {
    let cancelled = false

    setBotsLoading(true)
    setResourcesLoading(true)

    botAPI.getBots(currentCellSlug)
      .then((data) => {
        if (!cancelled) {
          setBots(data)
        }
      })
      .catch((err) => {
        console.error('[WaywardPines] Error loading bots:', err)
      })
      .finally(() => {
        if (!cancelled) {
          setBotsLoading(false)
        }
      })

    resourceAPI.getResources(currentCellSlug)
      .then((data) => {
        if (!cancelled) {
          setResources(data || [])
        }
      })
      .catch((err) => {
        console.error('[WaywardPines] Error loading resources:', err)
      })
      .finally(() => {
        if (!cancelled) {
          setResourcesLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
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

  const handleGather = async (resourceSlug) => {
    if (!resourceSlug) {
      return
    }
    navigate(`/resources/${currentCellSlug}?slug=${resourceSlug}`)
  }

  const normalizeImagePath = (img) => {
    if (!img) return null
    let p = img
    if (p.startsWith('/')) p = p.slice(1)
    p = p.replace(/^frontend\/assets\/images\//, '')
    if (p.startsWith('assets/images/')) p = p.replace(/^assets\/images\//, '')
    if (!p.startsWith('images/')) p = `images/${p}`
    return `/assets/${p}`
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
          {botsLoading ? (
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
          <h3 className="wayward-pines-resources-title">Ресурсы</h3>
          {resourcesLoading ? (
            <div>Загрузка...</div>
          ) : resources.length === 0 ? (
            <div>Ресурсы не найдены</div>
          ) : (
            <>
              {resources[0]?.image ? (
                <img
                  src={normalizeImagePath(resources[0].image)}
                  alt={resources[0].name}
                  className="resource-preview-image"
                />
              ) : null}
              <div className="resources-list">
                {resources.map((resource) => (
                  <div key={resource.id} className="resource-item">
                    <span>{resource.name}</span>
                  </div>
                ))}
              </div>
              <a
                href="#"
                className="resource-gather-link resource-gather-link-bottom"
                onClick={(e) => {
                  e.preventDefault()
                  handleGather(resources[0]?.slug)
                }}
              >
                собрать
              </a>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
